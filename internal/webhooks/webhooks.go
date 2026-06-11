// Package webhooks delivers fire-and-forget event notifications to
// per-project webhook URLs. The intended consumer is automation: assign an
// issue to claude → webhook fires → a script starts a Claude Code session.
package webhooks

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

var client = &http.Client{Timeout: 10 * time.Second}

// Event is the JSON body POSTed to the webhook URL.
type Event struct {
	Event     string    `json:"event"` // e.g. "issue.created", "comment.created"
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}

// Notify POSTs an event to url in the background. No-op when url is empty.
// Failures are retried once after 5s, then logged and dropped — webhook
// delivery must never block or fail the originating request.
func Notify(url, event string, data any) {
	if url == "" {
		return
	}

	body, err := json.Marshal(Event{Event: event, Timestamp: time.Now().UTC(), Data: data})
	if err != nil {
		log.Printf("webhook %s: marshal failed: %v", event, err)
		return
	}

	go func() {
		if post(url, body) {
			return
		}
		time.Sleep(5 * time.Second)
		if !post(url, body) {
			log.Printf("webhook %s -> %s: delivery failed after retry", event, url)
		}
	}()
}

func post(url string, body []byte) bool {
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 300
}
