package dispatcher

// Thin client for the taskdock API — the dispatcher asks for the next task
// (MCP, so the blocked-issue and priority logic is reused), reads issue
// state, and posts comments when sessions fail.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

// Task is the slice of the issue payload the dispatcher cares about.
type Task struct {
	Key    string `json:"key"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Branch string `json:"branch"`
}

// Taskdock talks to one taskdock instance.
type Taskdock struct {
	baseURL string
}

// NewTaskdock creates a client for the given base url.
func NewTaskdock(baseURL string) *Taskdock {
	return &Taskdock{baseURL: strings.TrimRight(baseURL, "/")}
}

// NextTask returns the highest-priority unblocked todo/backlog issue
// assigned to claude in the project, or nil when the queue is empty.
// Goes through MCP so priority/overdue/blocked ordering stays server-side.
func (t *Taskdock) NextTask(project string) (*Task, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "taskdock_get_next_task",
			"arguments": map[string]any{"project": project},
		},
	})

	resp, err := httpClient.Post(t.baseURL+"/mcp", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rpc struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rpc); err != nil {
		return nil, err
	}
	if len(rpc.Result.Content) == 0 {
		return nil, fmt.Errorf("empty MCP response")
	}

	text := rpc.Result.Content[0].Text
	if !strings.HasPrefix(strings.TrimSpace(text), "{") {
		return nil, nil // "No open tasks for 'claude' ..." — queue is empty
	}

	var task Task
	if err := json.Unmarshal([]byte(text), &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// GetIssue fetches current issue state by key.
func (t *Taskdock) GetIssue(key string) (*Task, error) {
	resp, err := httpClient.Get(t.baseURL + "/api/issues/" + key)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("issue %s: HTTP %d", key, resp.StatusCode)
	}

	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, err
	}
	return &task, nil
}

// Comment posts a comment on an issue as claude.
func (t *Taskdock) Comment(key, body string) error {
	reqBody, _ := json.Marshal(map[string]string{"author": "claude", "body": body})
	resp, err := httpClient.Post(t.baseURL+"/api/issues/"+key+"/comments", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("comment on %s: HTTP %d", key, resp.StatusCode)
	}
	return nil
}
