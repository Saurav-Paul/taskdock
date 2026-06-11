package dispatcher

// Status + log endpoints so the taskdock UI can show whether the dispatcher
// is alive and what it has been deciding. taskdock proxies these (the
// container reaches the host via host.docker.internal).

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const logBufferSize = 500

// ringLog tees log output to stderr and keeps the last N lines for /log.
type ringLog struct {
	mu    sync.Mutex
	lines []string
}

func (r *ringLog) Write(p []byte) (int, error) {
	r.mu.Lock()
	r.lines = append(r.lines, string(p))
	if len(r.lines) > logBufferSize {
		r.lines = r.lines[len(r.lines)-logBufferSize:]
	}
	r.mu.Unlock()
	return os.Stderr.Write(p)
}

func (r *ringLog) Lines() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.lines...)
}

var logBuffer = &ringLog{}

func init() {
	log.SetOutput(io.Writer(logBuffer))
}

// statusResponse is what GET /health returns.
type statusResponse struct {
	Running   bool            `json:"running"`
	DryRun    bool            `json:"dry_run"`
	Paused    bool            `json:"paused"`
	StartedAt time.Time       `json:"started_at"`
	Sessions  []sessionStatus `json:"sessions"`
	Projects  []string        `json:"projects"`
}

type sessionStatus struct {
	Project string `json:"project"`
	Key     string `json:"key"`
	PaneID  string `json:"pane_id"`
}

var startedAt = time.Now().UTC()

// handleHealth reports liveness, pause state, and running sessions.
func (d *Dispatcher) handleHealth(w http.ResponseWriter, r *http.Request) {
	d.mu.Lock()
	sessions := make([]sessionStatus, 0, len(d.sessions))
	for _, s := range d.sessions {
		sessions = append(sessions, sessionStatus{Project: s.project, Key: s.key, PaneID: s.paneID})
	}
	d.mu.Unlock()

	projects := make([]string, 0, len(d.cfg.Projects))
	for key := range d.cfg.Projects {
		projects = append(projects, key)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statusResponse{
		Running:   true,
		DryRun:    d.dryRun,
		Paused:    d.cfg.Paused(),
		StartedAt: startedAt,
		Sessions:  sessions,
		Projects:  projects,
	})
}

// handleLog returns the buffered log lines.
func (d *Dispatcher) handleLog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"lines": logBuffer.Lines()})
}
