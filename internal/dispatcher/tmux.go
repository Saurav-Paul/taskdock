package dispatcher

// tmux integration — sessions launch as interactive windows so output
// streams live and Claude's questions can be answered in the terminal.

import (
	"fmt"
	"os/exec"
	"strings"
)

// LaunchWindow opens a new tmux window named after the issue key, running
// the session command in the repo directory. Returns the pane id used to
// observe liveness. Creates the dispatcher's tmux session if needed.
func LaunchWindow(tmuxSession, windowName, repo, command, prompt string) (string, error) {
	// Ensure the parent tmux session exists (detached is fine — the user
	// attaches whenever they want to watch or answer questions).
	if err := exec.Command("tmux", "has-session", "-t", tmuxSession).Run(); err != nil {
		if err := exec.Command("tmux", "new-session", "-d", "-s", tmuxSession).Run(); err != nil {
			return "", fmt.Errorf("create tmux session: %w", err)
		}
	}

	// cd into the repo and run the session command with the prompt as its
	// final argument. Single-quote the prompt for the shell, escaping any
	// embedded single quotes.
	shellCmd := fmt.Sprintf("cd %s && %s '%s'", shellQuote(repo), command, strings.ReplaceAll(prompt, "'", `'\''`))

	out, err := exec.Command(
		"tmux", "new-window",
		"-t", tmuxSession,
		"-n", windowName,
		"-P", "-F", "#{pane_id}",
		shellCmd,
	).Output()
	if err != nil {
		return "", fmt.Errorf("tmux new-window: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// PaneAlive reports whether the pane still exists (session still running —
// tmux closes the window when its command exits).
func PaneAlive(paneID string) bool {
	out, err := exec.Command("tmux", "list-panes", "-a", "-F", "#{pane_id}").Output()
	if err != nil {
		return false
	}
	for _, id := range strings.Fields(string(out)) {
		if id == paneID {
			return true
		}
	}
	return false
}

// PaneContent captures the visible pane content — the waiting-detector
// hashes this to notice sessions that have gone quiet.
func PaneContent(paneID string) string {
	out, err := exec.Command("tmux", "capture-pane", "-p", "-t", paneID).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// MarkWindowDone renames a finished session's window so the tmux window
// list shows which sessions are complete (windows stay open for review).
func MarkWindowDone(paneID, key string) {
	_ = exec.Command("tmux", "rename-window", "-t", paneID, "✓ "+key).Run()
}
