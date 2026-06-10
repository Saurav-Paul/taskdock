package issues

// HTTP route handlers for the issues API.

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/Saurav-Paul/taskdock/internal/api/labels"
	"github.com/Saurav-Paul/taskdock/internal/api/projects"
	"github.com/Saurav-Paul/taskdock/internal/api/users"
)

// Register wires up the issues domain and returns the service
// for inter-domain dependencies (comments, mcp).
func Register(g *echo.Group, db *gorm.DB, p *projects.Service, u *users.Service, l *labels.Service) *Service {
	repo := NewRepository(db)
	service := NewService(repo, p, u, l)
	handler := &Handler{service: service}

	g.GET("", handler.list)           // GET    /api/issues?project=TD&status=todo&assignee=claude&q=...
	g.POST("", handler.create)        // POST   /api/issues
	g.GET("/:key", handler.get)       // GET    /api/issues/TD-12
	g.PATCH("/:key", handler.update)  // PATCH  /api/issues/TD-12
	g.DELETE("/:key", handler.delete) // DELETE /api/issues/TD-12

	g.POST("/:key/links", handler.addLink)          // POST   /api/issues/TD-12/links
	g.DELETE("/:key/links/:id", handler.deleteLink) // DELETE /api/issues/TD-12/links/3

	return service
}

// Handler provides the HTTP layer for issues.
type Handler struct {
	service *Service
}

func (h *Handler) list(c echo.Context) error {
	limit := 0
	if raw := c.QueryParam("limit"); raw != "" {
		limit, _ = strconv.Atoi(raw)
	}

	filters := ListFilters{
		Project:  c.QueryParam("project"),
		Status:   c.QueryParam("status"),
		Assignee: c.QueryParam("assignee"),
		Query:    c.QueryParam("q"),
		Limit:    limit,
	}

	result, err := h.service.List(filters)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, result)
}

func (h *Handler) get(c echo.Context) error {
	result, err := h.service.Get(c.Param("key"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Issue not found")
	}
	return c.JSON(http.StatusOK, result)
}

func (h *Handler) create(c echo.Context) error {
	var req IssueCreate
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	result, err := h.service.Create(req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusCreated, result)
}

func (h *Handler) update(c echo.Context) error {
	var req IssueUpdate
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	result, err := h.service.Update(c.Param("key"), req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusOK, result)
}

func (h *Handler) delete(c echo.Context) error {
	if err := h.service.Delete(c.Param("key")); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) addLink(c echo.Context) error {
	var req LinkCreate
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	link, err := h.service.AddLink(c.Param("key"), req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusCreated, link)
}

func (h *Handler) deleteLink(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid link id")
	}
	if err := h.service.DeleteLink(c.Param("key"), uint(id)); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}
