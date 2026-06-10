// Package attachments handles image uploads (pasted into the Tiptap editor).
// Files land in <DATA_DIR>/files and are served statically at /files/<name>,
// so markdown references them as ![](/files/<name>).
package attachments

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/Saurav-Paul/taskdock/internal/config"
)

// extensions we accept, keyed by MIME type.
var imageExtensions = map[string]string{
	"image/png":     ".png",
	"image/jpeg":    ".jpg",
	"image/gif":     ".gif",
	"image/webp":    ".webp",
	"image/svg+xml": ".svg",
}

// Register wires up the attachments domain.
func Register(g *echo.Group, cfg *config.Config) {
	handler := &Handler{cfg: cfg}

	// 10MB cap — pasted screenshots are well under this.
	g.Use(middleware.BodyLimit("10M"))
	g.POST("", handler.upload) // POST /api/attachments (multipart, field "file")
}

// Handler provides the HTTP layer for attachments.
type Handler struct {
	cfg *config.Config
}

func (h *Handler) upload(c echo.Context) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "multipart field 'file' is required")
	}

	contentType := fileHeader.Header.Get("Content-Type")
	ext, ok := imageExtensions[contentType]
	if !ok {
		return echo.NewHTTPError(http.StatusUnsupportedMediaType,
			fmt.Sprintf("unsupported type %q — images only", contentType))
	}

	src, err := fileHeader.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer src.Close()

	name := randomName() + ext
	dst, err := os.Create(filepath.Join(h.cfg.FilesDir, name))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, echo.Map{
		"url":      "/files/" + name,
		"filename": fileHeader.Filename,
	})
}

// randomName returns a 16-hex-char filename so uploads never collide
// and can't be guessed from the original name.
func randomName() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand only fails if the OS entropy source is broken.
		panic(err)
	}
	return strings.ToLower(hex.EncodeToString(b))
}
