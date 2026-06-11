// Package runners exposes dispatcher runner registration and status.
// A runner is a dispatcher process anchored to a path; it registers here
// (which doubles as its heartbeat) and appears as an assignable identity.
package runners

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/Saurav-Paul/taskdock/internal/api/issues"
	"github.com/Saurav-Paul/taskdock/internal/api/users"
)

// RegisterRequest is the JSON body for POST /api/runners/register.
type RegisterRequest struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Hostname string `json:"hostname"`
}

// RunnerStatus is a runner plus what it is currently working on.
type RunnerStatus struct {
	users.User
	CurrentIssue *issues.IssueRef `json:"current_issue,omitempty"`
}

// Register wires up the runners endpoints.
func Register(g *echo.Group, db *gorm.DB, userService *users.Service) {
	handler := &Handler{db: db, users: userService}

	g.POST("/register", handler.register)  // POST   /api/runners/register (also the heartbeat)
	g.GET("", handler.list)                // GET    /api/runners
	g.DELETE("/:name", handler.deregister) // DELETE /api/runners/zz-runner (remove a stale runner)
}

// Handler provides the HTTP layer for runners.
type Handler struct {
	db    *gorm.DB
	users *users.Service
}

func (h *Handler) register(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}
	if req.Name == "" || req.Path == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "'name' and 'path' are required")
	}

	runner, err := h.users.RegisterRunner(req.Name, req.Path, req.Hostname)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, runner)
}

// deregister removes a stale runner row. Only runner users can be deleted
// this way; issues assigned to it become unassigned (FK SET NULL).
func (h *Handler) deregister(c echo.Context) error {
	result := h.db.Where("name = ? AND kind = 'runner'", c.Param("name")).Delete(&users.User{})
	if result.Error != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "runner not found")
	}
	return c.NoContent(http.StatusNoContent)
}

// list returns all runners with online state and their current in-progress
// issue (if any) — what the UI runners panel renders.
func (h *Handler) list(c echo.Context) error {
	runnerUsers, err := h.users.Runners()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	out := make([]RunnerStatus, 0, len(runnerUsers))
	for _, runner := range runnerUsers {
		status := RunnerStatus{User: runner}

		var current issues.Issue
		err := h.db.Preload("Project").
			Where("assignee_id = ? AND status = 'in_progress'", runner.ID).
			Order("updated_at DESC").
			First(&current).Error
		if err == nil {
			ref := current.Key()
			status.CurrentIssue = &issues.IssueRef{Key: ref, Title: current.Title, Status: current.Status}
		}

		out = append(out, status)
	}
	return c.JSON(http.StatusOK, out)
}
