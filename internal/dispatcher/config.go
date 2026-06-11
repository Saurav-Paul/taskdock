// Package dispatcher reacts to taskdock webhook events by launching Claude
// Code sessions in tmux windows — assign a ticket to claude and a session
// starts in the mapped repo, works it, and the queue drains itself.
//
// This runs on the HOST (it drives tmux), not in the Docker container.
package dispatcher

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ProjectConfig maps a taskdock project to a repository on disk.
type ProjectConfig struct {
	Repo string `json:"repo"` // absolute path to the repo the sessions run in
}

// Config is loaded from dispatcher.json.
type Config struct {
	TaskdockURL string `json:"taskdock_url"` // e.g. http://localhost:8860
	Listen      string `json:"listen"`       // webhook listen address, default :9876
	NtfyURL     string `json:"ntfy_url"`     // optional: POST notifications here too
	// Command is the session command launched inside the tmux window; the
	// kickoff prompt is appended as a quoted argument. Overridable so the
	// whole launch/drain machinery can be tested with a stub.
	Command     string                   `json:"command"`
	TmuxSession string                   `json:"tmux_session"` // default "taskdock"
	PauseFile   string                   `json:"pause_file"`   // default ~/.taskdock-dispatcher-pause
	Projects    map[string]ProjectConfig `json:"projects"`     // project key → repo; unmapped = ignored
}

// LoadConfig reads and validates dispatcher.json, applying defaults.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.TaskdockURL == "" {
		cfg.TaskdockURL = "http://localhost:8860"
	}
	if cfg.Listen == "" {
		cfg.Listen = ":9876"
	}
	if cfg.Command == "" {
		cfg.Command = "claude --permission-mode acceptEdits"
	}
	if cfg.TmuxSession == "" {
		cfg.TmuxSession = "taskdock"
	}
	if cfg.PauseFile == "" {
		home, _ := os.UserHomeDir()
		cfg.PauseFile = filepath.Join(home, ".taskdock-dispatcher-pause")
	}
	if len(cfg.Projects) == 0 {
		return nil, fmt.Errorf("config has no projects mapped")
	}
	for key, p := range cfg.Projects {
		if p.Repo == "" {
			return nil, fmt.Errorf("project %s has no repo path", key)
		}
	}

	return &cfg, nil
}

// Paused reports whether the pause file exists — when it does, the
// dispatcher logs every decision but launches nothing.
func (c *Config) Paused() bool {
	_, err := os.Stat(c.PauseFile)
	return err == nil
}
