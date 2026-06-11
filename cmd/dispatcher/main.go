// The taskdock runner: anchor it to a path, and tickets assigned to it in
// taskdock become Claude Code sessions in tmux windows at that path.
//
//	dispatcher .                     # runner named after the folder
//	dispatcher -name hotfix ~/repo   # explicit name and path
//	TASKDOCK_URL=http://... dispatcher .
//
// Pause: touch ~/.taskdock-dispatcher-pause
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/Saurav-Paul/taskdock/internal/dispatcher"
)

func main() {
	name := flag.String("name", "", "runner name (default: folder basename)")
	taskdockURL := flag.String("taskdock", "", "taskdock url (default: $TASKDOCK_URL or http://localhost:8860)")
	command := flag.String("command", "claude --permission-mode acceptEdits", "session command (kickoff prompt appended)")
	tmuxSession := flag.String("tmux-session", "taskdock", "tmux session to open windows in")
	dryRun := flag.Bool("dry-run", false, "log launch decisions without starting sessions")
	flag.Parse()

	// Positional path, default current directory.
	rawPath := flag.Arg(0)
	if rawPath == "" {
		rawPath = "."
	}
	path, err := filepath.Abs(rawPath)
	if err != nil {
		log.Fatalf("resolve path: %v", err)
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		log.Fatalf("not a directory: %s", path)
	}

	if *name == "" {
		*name = filepath.Base(path)
	}
	if *taskdockURL == "" {
		*taskdockURL = os.Getenv("TASKDOCK_URL")
		if *taskdockURL == "" {
			*taskdockURL = "http://localhost:8860"
		}
	}

	hostname, _ := os.Hostname()
	home, _ := os.UserHomeDir()

	runner := dispatcher.New(dispatcher.Options{
		TaskdockURL: *taskdockURL,
		Name:        *name,
		Path:        path,
		Hostname:    hostname,
		Command:     *command,
		TmuxSession: *tmuxSession,
		PauseFile:   filepath.Join(home, ".taskdock-dispatcher-pause"),
		DryRun:      *dryRun,
	})
	log.Fatal(runner.Run())
}
