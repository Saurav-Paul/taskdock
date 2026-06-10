// Package projects handles project CRUD. Every issue belongs to a project,
// and the project key prefixes issue identifiers (e.g. "TD" → TD-12).
package projects

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// Project is the GORM model for the "projects" table.
type Project struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"not null" json:"name"`
	Key         string    `gorm:"uniqueIndex;not null" json:"key"`
	Description string    `gorm:"not null;default:''" json:"description"`
	NextNumber  int       `gorm:"not null;default:1" json:"next_number"` // issue counter
	CreatedAt   time.Time `json:"created_at"`
}

func (Project) TableName() string { return "projects" }

// ProjectCreate is the JSON body for POST /api/projects.
// Key is optional — defaults to the first 3 letters of the name, uppercased.
type ProjectCreate struct {
	Name        string `json:"name"`
	Key         string `json:"key"`
	Description string `json:"description"`
}

// ProjectUpdate is the JSON body for PATCH /api/projects/:key.
// Pointer fields distinguish "not sent" (nil) from "sent as empty".
type ProjectUpdate struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// Service handles business logic for projects.
type Service struct {
	db *gorm.DB
}

// List returns all projects ordered by creation time.
func (s *Service) List() ([]Project, error) {
	var projects []Project
	err := s.db.Order("created_at").Find(&projects).Error
	return projects, err
}

// GetByKey returns the project with the given key (case-insensitive).
func (s *Service) GetByKey(key string) (*Project, error) {
	var project Project
	if err := s.db.Where("key = ?", strings.ToUpper(key)).First(&project).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

// Create inserts a new project, deriving the key from the name when not given.
func (s *Service) Create(req ProjectCreate) (*Project, error) {
	key := strings.ToUpper(strings.TrimSpace(req.Key))
	if key == "" {
		// Default key: first 3 letters of the name (e.g. "taskdock" → "TAS")
		clean := strings.ToUpper(strings.TrimSpace(req.Name))
		if len(clean) > 3 {
			clean = clean[:3]
		}
		key = clean
	}

	project := Project{Name: req.Name, Key: key, Description: req.Description}
	if err := s.db.Create(&project).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

// Update applies non-nil fields and returns the updated project.
func (s *Service) Update(key string, req ProjectUpdate) (*Project, error) {
	project, err := s.GetByKey(key)
	if err != nil {
		return nil, err
	}

	updates := map[string]any{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if len(updates) > 0 {
		if err := s.db.Model(project).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return project, nil
}

// Delete removes a project; issues cascade via the FK.
func (s *Service) Delete(key string) error {
	project, err := s.GetByKey(key)
	if err != nil {
		return err
	}
	return s.db.Delete(project).Error
}

// Register wires up the projects domain and returns the service.
func Register(g *echo.Group, db *gorm.DB) *Service {
	service := &Service{db: db}
	handler := &Handler{service: service}

	g.GET("", handler.list)           // GET    /api/projects
	g.POST("", handler.create)        // POST   /api/projects
	g.GET("/:key", handler.get)       // GET    /api/projects/TD
	g.PATCH("/:key", handler.update)  // PATCH  /api/projects/TD
	g.DELETE("/:key", handler.delete) // DELETE /api/projects/TD

	return service
}

// Handler provides the HTTP layer for projects.
type Handler struct {
	service *Service
}

func (h *Handler) list(c echo.Context) error {
	projects, err := h.service.List()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, projects)
}

func (h *Handler) get(c echo.Context) error {
	project, err := h.service.GetByKey(c.Param("key"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Project not found")
	}
	return c.JSON(http.StatusOK, project)
}

func (h *Handler) create(c echo.Context) error {
	var req ProjectCreate
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "'name' is required")
	}

	project, err := h.service.Create(req)
	if err != nil {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}
	return c.JSON(http.StatusCreated, project)
}

func (h *Handler) update(c echo.Context) error {
	var req ProjectUpdate
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	project, err := h.service.Update(c.Param("key"), req)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Project not found")
	}
	return c.JSON(http.StatusOK, project)
}

func (h *Handler) delete(c echo.Context) error {
	if err := h.service.Delete(c.Param("key")); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Project not found")
	}
	return c.NoContent(http.StatusNoContent)
}
