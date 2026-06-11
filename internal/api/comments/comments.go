// Package comments handles the markdown comment thread on issues.
// Claude logs work progress here via the taskdock_save_comment MCP tool.
package comments

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/Saurav-Paul/taskdock/internal/api/issues"
	"github.com/Saurav-Paul/taskdock/internal/api/users"
	"github.com/Saurav-Paul/taskdock/internal/webhooks"
)

// Comment is the GORM model for the "comments" table.
type Comment struct {
	ID        uint `gorm:"primaryKey"`
	IssueID   uint `gorm:"not null"`
	AuthorID  *uint
	Author    *users.User
	Body      string `gorm:"not null"`
	CreatedAt time.Time
}

func (Comment) TableName() string { return "comments" }

// CommentCreate is the JSON body for POST /api/issues/:key/comments.
type CommentCreate struct {
	Author string `json:"author"` // user handle
	Body   string `json:"body"`   // markdown
}

// CommentResponse is the JSON shape returned for a comment.
type CommentResponse struct {
	ID        uint      `json:"id"`
	Author    string    `json:"author,omitempty"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

func toResponse(c *Comment) CommentResponse {
	resp := CommentResponse{ID: c.ID, Body: c.Body, CreatedAt: c.CreatedAt}
	if c.Author != nil {
		resp.Author = c.Author.Name
	}
	return resp
}

// Service handles comment business logic.
type Service struct {
	db     *gorm.DB
	issues *issues.Service
	users  *users.Service
}

// ListForIssue returns all comments on an issue, oldest first.
func (s *Service) ListForIssue(issueKey string) ([]CommentResponse, error) {
	issue, err := s.issues.GetModel(issueKey)
	if err != nil {
		return nil, fmt.Errorf("issue not found: %s", issueKey)
	}

	var rows []Comment
	if err := s.db.Preload("Author").
		Where("issue_id = ?", issue.ID).
		Order("created_at").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]CommentResponse, 0, len(rows))
	for i := range rows {
		out = append(out, toResponse(&rows[i]))
	}
	return out, nil
}

// Create adds a comment to an issue.
func (s *Service) Create(issueKey string, req CommentCreate) (*CommentResponse, error) {
	if req.Body == "" {
		return nil, fmt.Errorf("'body' is required")
	}

	issue, err := s.issues.GetModel(issueKey)
	if err != nil {
		return nil, fmt.Errorf("issue not found: %s", issueKey)
	}

	comment := Comment{IssueID: issue.ID, Body: req.Body}
	if req.Author != "" {
		user, err := s.users.GetByName(req.Author)
		if err != nil {
			return nil, fmt.Errorf("unknown user: %s", req.Author)
		}
		comment.AuthorID = &user.ID
		comment.Author = user
	}

	if err := s.db.Create(&comment).Error; err != nil {
		return nil, err
	}

	resp := toResponse(&comment)
	webhooks.Notify(issue.Project.WebhookURL, "comment.created", map[string]any{
		"issue":   issueKey,
		"comment": resp,
	})
	return &resp, nil
}

// Register wires up comment routes on the issues group and returns the service.
// Routes live under /api/issues/:key/comments.
func Register(g *echo.Group, db *gorm.DB, issueService *issues.Service, userService *users.Service) *Service {
	service := &Service{db: db, issues: issueService, users: userService}
	handler := &Handler{service: service}

	g.GET("/:key/comments", handler.list)   // GET  /api/issues/TD-12/comments
	g.POST("/:key/comments", handler.create) // POST /api/issues/TD-12/comments

	return service
}

// Handler provides the HTTP layer for comments.
type Handler struct {
	service *Service
}

func (h *Handler) list(c echo.Context) error {
	result, err := h.service.ListForIssue(c.Param("key"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	return c.JSON(http.StatusOK, result)
}

func (h *Handler) create(c echo.Context) error {
	var req CommentCreate
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	result, err := h.service.Create(c.Param("key"), req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusCreated, result)
}
