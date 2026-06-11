// Package dispatcher implements a taskdock RUNNER: a host-side process
// anchored to one path. It registers itself as an assignable identity,
// polls for tickets assigned to it, and works each one in an interactive
// tmux window (cwd = its path). Run one per repo/worktree you want agents
// working in:
//
//	dispatcher .                # runner named after the folder
//	dispatcher -name fix ~/repo # explicit name and path
package dispatcher

import (
	"log"
	"os"
	"time"
)

const (
	heartbeatInterval = 10 * time.Second
	pollInterval      = 5 * time.Second
	monitorInterval   = 5 * time.Second
	waitingInterval   = 2 * time.Minute
	waitingAfter      = 10 * time.Minute
	// A ticket whose session died without progress is not auto-relaunched
	// for this long — re-assigning it in taskdock resets the runner's view.
	failureCooldown = 30 * time.Minute
)

// Options configure a runner (from flags, no config file).
type Options struct {
	TaskdockURL string
	Name        string // runner identity, defaults to the folder basename
	Path        string // absolute path sessions run in
	Hostname    string
	Command     string // session command; kickoff prompt appended (stub-able for tests)
	TmuxSession string
	PauseFile   string
	DryRun      bool
}

// session tracks the one running tmux window (a runner is serial).
type session struct {
	key         string
	paneID      string
	startStatus string
	lastContent string
	lastChange  time.Time
	warned      bool
}

// Runner is the dispatcher process state.
type Runner struct {
	opts     Options
	taskdock *Taskdock
	notifier *Notifier

	current *session             // nil when idle
	failed  map[string]time.Time // issue key → failure time (cooldown)
}

// New creates a runner.
func New(opts Options) *Runner {
	return &Runner{
		opts:     opts,
		taskdock: NewTaskdock(opts.TaskdockURL),
		notifier: NewNotifier(""),
		failed:   make(map[string]time.Time),
	}
}

func (r *Runner) paused() bool {
	_, err := os.Stat(r.opts.PauseFile)
	return err == nil
}

// Run is the runner's main loop: heartbeat, poll for work when idle,
// watch the running session, nudge when it looks stuck. Single-threaded
// by design — a runner works one ticket at a time.
func (r *Runner) Run() error {
	if err := r.register(); err != nil {
		return err
	}
	log.Printf("runner %q online — path %s, taskdock %s, dry-run %v",
		r.opts.Name, r.opts.Path, r.opts.TaskdockURL, r.opts.DryRun)

	heartbeat := time.Tick(heartbeatInterval)
	poll := time.Tick(pollInterval)
	monitor := time.Tick(monitorInterval)
	waiting := time.Tick(waitingInterval)

	for {
		select {
		case <-heartbeat:
			if err := r.register(); err != nil {
				log.Printf("heartbeat failed: %v", err)
			}
		case <-poll:
			if r.current == nil && !r.paused() {
				r.pickUpWork()
			}
		case <-monitor:
			if r.current != nil {
				r.checkSession()
			}
		case <-waiting:
			if r.current != nil {
				r.checkWaiting()
			}
		}
	}
}

func (r *Runner) register() error {
	return r.taskdock.Register(r.opts.Name, r.opts.Path, r.opts.Hostname)
}

// pickUpWork asks for the next unblocked ticket assigned to this runner
// and launches a session for it.
func (r *Runner) pickUpWork() {
	task, err := r.taskdock.NextTask(r.opts.Name)
	if err != nil {
		log.Printf("next-task failed: %v", err)
		return
	}
	if task == nil {
		return // queue empty — stay idle, keep polling
	}

	if failedAt, cooling := r.failed[task.Key]; cooling && time.Since(failedAt) < failureCooldown {
		return // failed recently; re-assigning in taskdock clears this (new last assignment resets below)
	}

	if r.opts.DryRun {
		log.Printf("DRY-RUN: would launch session for %s (%s) in %s", task.Key, task.Title, r.opts.Path)
		// Mark as failed so dry-run doesn't relog every poll tick.
		r.failed[task.Key] = time.Now()
		return
	}

	prompt := kickoffPrompt(task)
	paneID, err := LaunchWindow(r.opts.TmuxSession, task.Key, r.opts.Path, r.opts.Command, prompt)
	if err != nil {
		log.Printf("launch failed for %s: %v", task.Key, err)
		r.notifier.Send("launch failed for " + task.Key + ": " + err.Error())
		r.failed[task.Key] = time.Now()
		return
	}

	r.current = &session{key: task.Key, paneID: paneID, startStatus: task.Status, lastChange: time.Now()}
	log.Printf("launched session for %s in tmux pane %s", task.Key, paneID)
	r.notifier.Send("session started: " + task.Key + " — " + task.Title)
}

// checkSession watches for the two endings: the ticket reached
// in_review/done (success — interactive sessions never exit on their own,
// so ticket state is the signal; the window stays open, renamed ✓), or
// the pane died (failure/abort).
func (r *Runner) checkSession() {
	s := r.current

	if !PaneAlive(s.paneID) {
		r.current = nil
		issue, err := r.taskdock.GetIssue(s.key)
		if err == nil && (issue.Status == "in_review" || issue.Status == "done" || issue.Status == "canceled") {
			log.Printf("%s: session closed, ticket %s", s.key, issue.Status)
			return
		}
		log.Printf("%s: session exited without a status change", s.key)
		r.failed[s.key] = time.Now()
		r.notifier.Send("⚠ " + s.key + " session ended without a status change")
		if err := r.taskdock.Comment(s.key,
			"runner "+r.opts.Name+": session ended without a status change — check the tmux window and re-assign to retry."); err != nil {
			log.Printf("%s: stuck-comment failed: %v", s.key, err)
		}
		return
	}

	issue, err := r.taskdock.GetIssue(s.key)
	if err != nil {
		return
	}
	if issue.Status == "in_review" || issue.Status == "done" {
		log.Printf("%s: ticket reached %s — session complete (window left open)", s.key, issue.Status)
		MarkWindowDone(s.paneID, s.key)
		r.notifier.Send(s.key + " → " + issue.Status)
		r.current = nil // next poll tick picks up further queued work
	}
}

// checkWaiting notifies once when the session pane has gone quiet without
// the ticket moving — usually Claude waiting for an answer.
func (r *Runner) checkWaiting() {
	s := r.current

	content := PaneContent(s.paneID)
	if content != s.lastContent {
		s.lastContent = content
		s.lastChange = time.Now()
		s.warned = false
		return
	}
	if s.warned || time.Since(s.lastChange) < waitingAfter {
		return
	}
	issue, err := r.taskdock.GetIssue(s.key)
	if err == nil && issue.Status != s.startStatus {
		return // moved at least once — working, just quiet
	}
	s.warned = true
	log.Printf("%s: pane silent — may be waiting for input", s.key)
	r.notifier.Send("⏸ " + s.key + " session may be waiting for input")
}

// kickoffPrompt starts the session on one specific ticket.
func kickoffPrompt(task *Task) string {
	return "You are working ticket " + task.Key + " from the taskdock tracker (connected via MCP). " +
		"Steps: call taskdock_get_issue for " + task.Key + " and read it fully (description, comments, images). " +
		"Set its status to in_progress. Create or check out the branch " + task.Branch + " unless the ticket says otherwise. " +
		"Implement the ticket. Log meaningful progress and decisions with taskdock_save_comment. " +
		"If you open a PR, attach it with taskdock_save_link. " +
		"When the work is ready for review, set status to in_review. " +
		"If you are blocked, comment why and stop."
}
