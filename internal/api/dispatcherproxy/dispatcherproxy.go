// Package dispatcherproxy lets the UI see the host-side dispatcher: the
// browser talks to taskdock, taskdock forwards to the dispatcher daemon
// (host.docker.internal from inside the container).
package dispatcherproxy

import (
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

var client = &http.Client{Timeout: 3 * time.Second}

// Register wires up the proxy endpoints.
func Register(g *echo.Group, dispatcherURL string) {
	handler := &Handler{baseURL: dispatcherURL}

	g.GET("/status", handler.status) // GET /api/dispatcher/status
	g.GET("/log", handler.log)       // GET /api/dispatcher/log
}

// Handler forwards to the dispatcher daemon.
type Handler struct {
	baseURL string
}

// status proxies /health; an unreachable dispatcher is not an error —
// it just means "not running", which the UI shows as a gray dot.
func (h *Handler) status(c echo.Context) error {
	resp, err := client.Get(h.baseURL + "/health")
	if err != nil {
		return c.JSON(http.StatusOK, echo.Map{"running": false})
	}
	defer resp.Body.Close()
	return c.Stream(http.StatusOK, "application/json", resp.Body)
}

func (h *Handler) log(c echo.Context) error {
	resp, err := client.Get(h.baseURL + "/log")
	if err != nil {
		return c.JSON(http.StatusOK, echo.Map{"lines": []string{}})
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.JSON(http.StatusOK, echo.Map{"lines": []string{}})
	}
	return c.Blob(http.StatusOK, "application/json", body)
}
