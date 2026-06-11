package dispatcher

// Core reactor: webhook event in → filter → launch session → observe →
// drain the queue. One session per project at a time; everything else is
// logged and picked up by the drain when the current session exits.

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	monitorInterval = 5 * time.Second
	waitingInterval = 2 * time.Minute
	waitingAfter    = 10 * time.Minute
	// A ticket whose session died without progress is not auto-relaunched
	// for this long — re-assigning it in taskdock bypasses the cooldown.
	failureCooldown = 30 * time.Minute
)

// session tracks one running tmux window.
type session struct {
	project     string
	key         string // issue being worked
	paneID      string
	startStatus string // issue status at launch (for the waiting detector)
	lastContent string // pane content hash from the previous waiting sweep
	lastChange  time.Time
	warned      bool // waiting notification sent
}

// Dispatcher holds the reactor state.
type Dispatcher struct {
	cfg      *Config
	taskdock *Taskdock
	notifier *Notifier
	dryRun   bool

	mu       sync.Mutex
	sessions map[string]*session  // project key → running session
	failed   map[string]time.Time // issue key → when its session died without progress
}

// New creates a dispatcher.
func New(cfg *Config, dryRun bool) *Dispatcher {
	return &Dispatcher{
		cfg:      cfg,
		taskdock: NewTaskdock(cfg.TaskdockURL),
		notifier: NewNotifier(cfg.NtfyURL),
		dryRun:   dryRun,
		sessions: make(map[string]*session),
		failed:   make(map[string]time.Time),
	}
}

// webhookEvent is the body taskdock POSTs.
type webhookEvent struct {
	Event string `json:"event"`
	Data  struct {
		Key      string `json:"key"`
		Project  string `json:"project"`
		Status   string `json:"status"`
		Assignee string `json:"assignee"`
	} `json:"data"`
}

// HandleWebhook is the POST /hook handler.
func (d *Dispatcher) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	var ev webhookEvent
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)

	// Filter: only fresh assignments to claude in an unstarted status.
	// claude's own transitions (in_progress, in_review, …) never match,
	// so the dispatcher can't trigger off its own sessions' work.
	switch {
	case ev.Event != "issue.created" && ev.Event != "issue.updated":
		return // comment.created etc — v1 ignores
	case ev.Data.Assignee != "claude":
		log.Printf("skip %s %s: assignee %q", ev.Event, ev.Data.Key, ev.Data.Assignee)
		return
	case ev.Data.Status != "todo" && ev.Data.Status != "backlog":
		log.Printf("skip %s %s: status %q", ev.Event, ev.Data.Key, ev.Data.Status)
		return
	}

	if _, mapped := d.cfg.Projects[ev.Data.Project]; !mapped {
		log.Printf("skip %s: project %s not in config", ev.Data.Key, ev.Data.Project)
		return
	}
	if d.cfg.Paused() {
		log.Printf("skip %s: dispatcher paused (%s exists)", ev.Data.Key, d.cfg.PauseFile)
		return
	}

	log.Printf("trigger: %s %s (project %s)", ev.Event, ev.Data.Key, ev.Data.Project)
	// Explicit (re-)assignment clears any failure cooldown for the issue.
	d.mu.Lock()
	delete(d.failed, ev.Data.Key)
	d.mu.Unlock()
	go d.tryLaunch(ev.Data.Project)
}

// tryLaunch starts a session for the project's next task, unless one is
// already running. The dispatcher (not the session) picks the ticket, so
// it always knows which issue a window is working.
func (d *Dispatcher) tryLaunch(project string) {
	d.mu.Lock()
	if _, running := d.sessions[project]; running {
		d.mu.Unlock()
		log.Printf("%s: session already running — queue will drain on exit", project)
		return
	}
	// Reserve the slot before the (slow) network/tmux work.
	d.sessions[project] = &session{project: project}
	d.mu.Unlock()

	release := func() {
		d.mu.Lock()
		delete(d.sessions, project)
		d.mu.Unlock()
	}

	task, err := d.taskdock.NextTask(project)
	if err != nil {
		log.Printf("%s: next-task failed: %v", project, err)
		release()
		return
	}
	if task == nil {
		log.Printf("%s: queue empty", project)
		release()
		return
	}

	// Don't loop on a ticket whose session just died without progress —
	// next-task would hand us the same one forever.
	d.mu.Lock()
	failedAt, cooling := d.failed[task.Key]
	d.mu.Unlock()
	if cooling && time.Since(failedAt) < failureCooldown {
		log.Printf("%s: %s failed %s ago — cooling down, not relaunching (re-assign to retry)",
			project, task.Key, time.Since(failedAt).Round(time.Second))
		release()
		return
	}

	if d.dryRun {
		log.Printf("DRY-RUN %s: would launch session for %s (%s) in %s",
			project, task.Key, task.Title, d.cfg.Projects[project].Repo)
		release()
		return
	}

	prompt := kickoffPrompt(task)
	paneID, err := LaunchWindow(d.cfg.TmuxSession, task.Key, d.cfg.Projects[project].Repo, d.cfg.Command, prompt)
	if err != nil {
		log.Printf("%s: launch failed: %v", project, err)
		d.notifier.Send("launch failed for " + task.Key + ": " + err.Error())
		release()
		return
	}

	d.mu.Lock()
	d.sessions[project] = &session{
		project:     project,
		key:         task.Key,
		paneID:      paneID,
		startStatus: task.Status,
		lastChange:  time.Now(),
	}
	d.mu.Unlock()

	log.Printf("%s: launched session for %s in tmux pane %s", project, task.Key, paneID)
	d.notifier.Send("session started: " + task.Key + " — " + task.Title)
}

// Run starts the webhook server plus the monitor and waiting-detector
// loops. Blocks.
func (d *Dispatcher) Run() error {
	go d.monitorLoop()
	go d.waitingLoop()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /hook", d.HandleWebhook)
	mux.HandleFunc("GET /health", d.handleHealth)
	mux.HandleFunc("GET /log", d.handleLog)

	log.Printf("dispatcher listening on %s (taskdock: %s, dry-run: %v)", d.cfg.Listen, d.cfg.TaskdockURL, d.dryRun)
	return http.ListenAndServe(d.cfg.Listen, mux)
}

// monitorLoop notices dead panes: checks the outcome, surfaces silent
// failures on the ticket, releases the lock, and drains the queue.
func (d *Dispatcher) monitorLoop() {
	for range time.Tick(monitorInterval) {
		d.mu.Lock()
		var finished []*session
		for project, s := range d.sessions {
			if s.paneID != "" && !PaneAlive(s.paneID) {
				finished = append(finished, s)
				delete(d.sessions, project)
			}
		}
		d.mu.Unlock()

		for _, s := range finished {
			d.sessionFinished(s)
		}
	}
}

// sessionFinished handles a session whose tmux window closed.
func (d *Dispatcher) sessionFinished(s *session) {
	issue, err := d.taskdock.GetIssue(s.key)
	if err != nil {
		log.Printf("%s: post-exit issue fetch failed: %v", s.key, err)
	} else {
		switch issue.Status {
		case "todo", "backlog", "in_progress":
			// Session died without moving the ticket — make it visible
			// and put the ticket on cooldown so the drain doesn't loop.
			log.Printf("%s: session exited without a status change (still %s)", s.key, issue.Status)
			d.mu.Lock()
			d.failed[s.key] = time.Now()
			d.mu.Unlock()
			d.notifier.Send("⚠ " + s.key + " session ended without a status change")
			if err := d.taskdock.Comment(s.key,
				"dispatcher: session ended without a status change — check the tmux log and relaunch by re-assigning."); err != nil {
				log.Printf("%s: stuck-comment failed: %v", s.key, err)
			}
		default:
			log.Printf("%s: session finished, status %s", s.key, issue.Status)
			d.notifier.Send(s.key + " → " + issue.Status)
		}
	}

	// Drain: more assigned work in this project? Launch the next one.
	if d.cfg.Paused() {
		log.Printf("%s: paused — not draining queue", s.project)
		return
	}
	d.tryLaunch(s.project)
}

// waitingLoop notifies (once per session) when a pane has been silent for
// a while and its ticket hasn't moved — usually Claude waiting for input.
func (d *Dispatcher) waitingLoop() {
	for range time.Tick(waitingInterval) {
		d.mu.Lock()
		var candidates []*session
		for _, s := range d.sessions {
			if s.paneID != "" {
				candidates = append(candidates, s)
			}
		}
		d.mu.Unlock()

		for _, s := range candidates {
			content := PaneContent(s.paneID)
			if content != s.lastContent {
				s.lastContent = content
				s.lastChange = time.Now()
				s.warned = false
				continue
			}
			if s.warned || time.Since(s.lastChange) < waitingAfter {
				continue
			}
			issue, err := d.taskdock.GetIssue(s.key)
			if err == nil && issue.Status != s.startStatus {
				continue // ticket moved — it's working, just quiet
			}
			s.warned = true
			log.Printf("%s: pane silent for %s — may be waiting for input", s.key, waitingAfter)
			d.notifier.Send("⏸ " + s.key + " session may be waiting for input")
		}
	}
}

// kickoffPrompt is what the session starts with — it works one specific
// ticket, chosen by the dispatcher.
func kickoffPrompt(task *Task) string {
	return "You are working ticket " + task.Key + " from the taskdock tracker (connected via MCP). " +
		"Steps: call taskdock_get_issue for " + task.Key + " and read it fully (description, comments, images). " +
		"Set its status to in_progress. Create or check out the branch " + task.Branch + ". " +
		"Implement the ticket. Log meaningful progress and decisions with taskdock_save_comment. " +
		"If you open a PR, attach it with taskdock_save_link. " +
		"When the work is ready for review, set status to in_review and exit. " +
		"If you are blocked, comment why and exit without changing the status further."
}
