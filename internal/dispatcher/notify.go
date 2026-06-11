package dispatcher

// Notifications: macOS notification center (osascript) plus an optional
// ntfy endpoint for phones. Best-effort — failures are logged, never fatal.

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

// Notifier fans a message out to the configured channels.
type Notifier struct {
	ntfyURL string
}

// NewNotifier creates a notifier; ntfyURL may be empty.
func NewNotifier(ntfyURL string) *Notifier {
	return &Notifier{ntfyURL: ntfyURL}
}

// Send shows a macOS notification and, when configured, POSTs to ntfy.
func (n *Notifier) Send(message string) {
	script := fmt.Sprintf("display notification %q with title %q", message, "taskdock dispatcher")
	if err := exec.Command("osascript", "-e", script).Run(); err != nil {
		log.Printf("notify: osascript failed: %v", err)
	}

	if n.ntfyURL != "" {
		resp, err := http.Post(n.ntfyURL, "text/plain", strings.NewReader(message))
		if err != nil {
			log.Printf("notify: ntfy failed: %v", err)
			return
		}
		resp.Body.Close()
	}
}
