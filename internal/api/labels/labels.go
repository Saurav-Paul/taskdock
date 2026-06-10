// Package labels handles global issue labels.
package labels

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// Label is the GORM model for the "labels" table.
type Label struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	Name  string `gorm:"uniqueIndex;not null" json:"name"`
	Color string `gorm:"not null;default:'#8b8b8b'" json:"color"`
}

func (Label) TableName() string { return "labels" }

// LabelCreate is the JSON body for POST /api/labels.
type LabelCreate struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// Service handles label lookups; issues use GetOrCreate to attach labels by name.
type Service struct {
	db *gorm.DB
}

// List returns all labels ordered by name.
func (s *Service) List() ([]Label, error) {
	var labels []Label
	err := s.db.Order("name").Find(&labels).Error
	return labels, err
}

// GetOrCreate resolves label names to rows, creating any that don't exist yet.
// Used when saving an issue with a list of label names.
func (s *Service) GetOrCreate(names []string) ([]Label, error) {
	labels := make([]Label, 0, len(names))
	for _, name := range names {
		if name == "" {
			continue
		}
		var label Label
		err := s.db.Where("name = ?", name).FirstOrCreate(&label, Label{Name: name}).Error
		if err != nil {
			return nil, err
		}
		labels = append(labels, label)
	}
	return labels, nil
}

// Register wires up the labels domain and returns the service.
func Register(g *echo.Group, db *gorm.DB) *Service {
	service := &Service{db: db}
	handler := &Handler{service: service}

	g.GET("", handler.list)    // GET  /api/labels
	g.POST("", handler.create) // POST /api/labels

	return service
}

// Handler provides the HTTP layer for labels.
type Handler struct {
	service *Service
}

func (h *Handler) list(c echo.Context) error {
	labels, err := h.service.List()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, labels)
}

func (h *Handler) create(c echo.Context) error {
	var req LabelCreate
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "'name' is required")
	}

	label := Label{Name: req.Name, Color: req.Color}
	if label.Color == "" {
		label.Color = "#8b8b8b"
	}
	if err := h.service.db.Create(&label).Error; err != nil {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}
	return c.JSON(http.StatusCreated, label)
}
