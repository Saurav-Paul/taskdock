package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/Saurav-Paul/taskdock/internal/api/attachments"
	"github.com/Saurav-Paul/taskdock/internal/api/comments"
	"github.com/Saurav-Paul/taskdock/internal/api/dispatcherproxy"
	"github.com/Saurav-Paul/taskdock/internal/api/issues"
	"github.com/Saurav-Paul/taskdock/internal/api/labels"
	"github.com/Saurav-Paul/taskdock/internal/api/mcp"
	"github.com/Saurav-Paul/taskdock/internal/api/projects"
	"github.com/Saurav-Paul/taskdock/internal/api/users"
	"github.com/Saurav-Paul/taskdock/internal/config"
	"github.com/Saurav-Paul/taskdock/internal/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Open SQLite and apply pending Goose migrations on startup.
	db, err := database.Setup(cfg)
	if err != nil {
		log.Fatalf("Failed to setup database: %v", err)
	}

	// Branch names (UI copy button + MCP payloads) honor the configured prefix.
	issues.SetBranchPrefix(cfg.BranchPrefix)

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.CORS())

	e.GET("/api/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, echo.Map{"status": "ok"})
	})

	// --- Domain registration (drop pattern) ---
	// Register() wires repo → service → handler and returns the service
	// so dependent domains can use it.
	userService := users.Register(e.Group("/api/users"), db)
	projectService := projects.Register(e.Group("/api/projects"), db)
	labelService := labels.Register(e.Group("/api/labels"), db)

	issuesGroup := e.Group("/api/issues")
	issueService := issues.Register(issuesGroup, db, projectService, userService, labelService)
	commentService := comments.Register(issuesGroup, db, issueService, userService)

	// --- Attachments (pasted images) ---
	attachments.Register(e.Group("/api/attachments"), cfg)
	e.Static("/files", cfg.FilesDir)

	// --- Dispatcher status/log proxy (UI visibility for the host daemon) ---
	dispatcherproxy.Register(e.Group("/api/dispatcher"), cfg.DispatcherURL)

	// --- MCP endpoint (POST /mcp) ---
	mcpService := mcp.NewService(projectService, issueService, commentService, cfg.FilesDir)
	mcp.Register(e, mcpService, cfg.Port)

	// --- Static frontend (prod) ---
	// The built React app is copied to ./static in the Docker image.
	// HTML5 mode serves index.html for app routes like /PRO-967, so issue
	// deep links pasted into a new tab load the app with that issue open.
	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:  "static",
		HTML5: true,
		Skipper: func(c echo.Context) bool {
			p := c.Request().URL.Path
			return strings.HasPrefix(p, "/api/") ||
				p == "/mcp" || strings.HasPrefix(p, "/mcp/") ||
				strings.HasPrefix(p, "/files/")
		},
	}))

	e.Logger.Fatal(e.Start(":" + cfg.Port))
}
