// Package users handles the identity rows (no auth in v1 — "saurav" and "claude").
package users

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// User is the GORM model for the "users" table. Runners (dispatcher
// processes) register as users with kind=runner so they are assignable
// like anyone else; their extra columns describe where they run.
type User struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `gorm:"uniqueIndex;not null" json:"name"`
	DisplayName string     `gorm:"not null;default:''" json:"display_name"`
	Kind        string     `gorm:"not null;default:'person'" json:"kind"` // person | runner
	Path        string     `gorm:"not null;default:''" json:"path,omitempty"`
	Hostname    string     `gorm:"not null;default:''" json:"hostname,omitempty"`
	LastSeen    *time.Time `json:"last_seen,omitempty"`
	// Online is computed, not stored: runner with a fresh heartbeat.
	Online    bool      `gorm:"-" json:"online"`
	CreatedAt time.Time `json:"created_at"`
}

// runnerOnlineWindow is how fresh a heartbeat must be to count as online.
const runnerOnlineWindow = 30 * time.Second

// ComputeOnline fills the Online field from the heartbeat age.
func (u *User) ComputeOnline() {
	u.Online = u.Kind == "runner" && u.LastSeen != nil &&
		time.Since(*u.LastSeen) < runnerOnlineWindow
}

func (User) TableName() string { return "users" }

// UserCreate is the JSON body for POST /api/users.
type UserCreate struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

// Service handles user lookups and creation. Other domains (issues, comments,
// mcp) depend on it to resolve assignee/author names to rows.
type Service struct {
	db *gorm.DB
}

// List returns all users ordered by id, with runner online state computed.
func (s *Service) List() ([]User, error) {
	var users []User
	if err := s.db.Order("id").Find(&users).Error; err != nil {
		return nil, err
	}
	for i := range users {
		users[i].ComputeOnline()
	}
	return users, nil
}

// RegisterRunner upserts a runner user and stamps its heartbeat.
// Registration and heartbeat are the same call.
func (s *Service) RegisterRunner(name, path, hostname string) (*User, error) {
	now := time.Now().UTC()

	var user User
	err := s.db.Where("name = ?", name).First(&user).Error
	if err != nil {
		user = User{Name: name, DisplayName: name, Kind: "runner", Path: path, Hostname: hostname, LastSeen: &now}
		if err := s.db.Create(&user).Error; err != nil {
			return nil, err
		}
	} else {
		updates := map[string]any{"kind": "runner", "path": path, "hostname": hostname, "last_seen": now}
		if err := s.db.Model(&user).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	user.ComputeOnline()
	return &user, nil
}

// Runners returns runner users only, online state computed.
func (s *Service) Runners() ([]User, error) {
	var runners []User
	if err := s.db.Where("kind = 'runner'").Order("name").Find(&runners).Error; err != nil {
		return nil, err
	}
	for i := range runners {
		runners[i].ComputeOnline()
	}
	return runners, nil
}

// GetByName returns the user with the given handle, or an error if not found.
func (s *Service) GetByName(name string) (*User, error) {
	var user User
	if err := s.db.Where("name = ?", name).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Create inserts a new user.
func (s *Service) Create(req UserCreate) (*User, error) {
	user := User{Name: req.Name, DisplayName: req.DisplayName}
	if err := s.db.Create(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Register wires up the users domain and returns the service
// for inter-domain dependencies (issues, comments, mcp).
func Register(g *echo.Group, db *gorm.DB) *Service {
	service := &Service{db: db}
	handler := &Handler{service: service}

	g.GET("", handler.list)   // GET  /api/users
	g.POST("", handler.create) // POST /api/users

	return service
}

// Handler provides the HTTP layer for users.
type Handler struct {
	service *Service
}

func (h *Handler) list(c echo.Context) error {
	users, err := h.service.List()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, users)
}

func (h *Handler) create(c echo.Context) error {
	var req UserCreate
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "'name' is required")
	}

	user, err := h.service.Create(req)
	if err != nil {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}
	return c.JSON(http.StatusCreated, user)
}
