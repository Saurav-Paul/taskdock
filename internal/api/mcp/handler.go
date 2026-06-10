// Package mcp implements the MCP protocol endpoint (JSON-RPC 2.0 over HTTP).
// Structure ported from shelf's backend/api/mcp — same protocol version and
// method set, no external MCP library.
package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

const protocolVersion = "2024-11-05"

// jsonrpcRequest is a single incoming JSON-RPC 2.0 request.
type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// toolCallParams are the params for the "tools/call" method.
type toolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// Register wires the MCP endpoint onto the Echo instance.
func Register(e *echo.Echo, service *Service, port string) {
	handler := &Handler{service: service, port: port}

	e.POST("/mcp", handler.handle)            // JSON-RPC protocol endpoint (all projects)
	e.GET("/mcp/config.json", handler.config) // Claude-compatible config snippet

	// Project-scoped variant: same tools, but issue listing/creation is
	// locked to one project — paste into a repo's MCP config so that
	// repo's tickets all live in that project.
	e.POST("/mcp/:project", handler.handle)
	e.GET("/mcp/:project/config.json", handler.config)
}

// Handler dispatches JSON-RPC requests to the tool service.
type Handler struct {
	service *Service
	port    string
}

// handle processes a single or batch JSON-RPC request.
// For /mcp/:project routes, the scope param locks tools to that project.
func (h *Handler) handle(c echo.Context) error {
	scope := c.Param("project")

	var raw json.RawMessage
	if err := json.NewDecoder(c.Request().Body).Decode(&raw); err != nil {
		return c.JSON(http.StatusOK, errorResponse(nil, -32700, "Parse error"))
	}

	// Batch request: a JSON array of requests.
	if len(raw) > 0 && raw[0] == '[' {
		var reqs []jsonrpcRequest
		if err := json.Unmarshal(raw, &reqs); err != nil {
			return c.JSON(http.StatusOK, errorResponse(nil, -32600, "Invalid Request"))
		}
		results := make([]map[string]any, 0, len(reqs))
		for _, req := range reqs {
			result := h.dispatch(req, scope)
			if req.ID != nil {
				results = append(results, result)
			}
		}
		return c.JSON(http.StatusOK, results)
	}

	// Single request.
	var req jsonrpcRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return c.JSON(http.StatusOK, errorResponse(nil, -32600, "Invalid Request"))
	}
	return c.JSON(http.StatusOK, h.dispatch(req, scope))
}

// dispatch routes a JSON-RPC method to its implementation.
func (h *Handler) dispatch(req jsonrpcRequest, scope string) map[string]any {
	switch req.Method {
	case "initialize":
		return successResponse(req.ID, map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities": map[string]any{
				"tools": map[string]any{"listChanged": false},
			},
			"serverInfo": map[string]any{
				"name":    "taskdock",
				"version": "1.0.0",
			},
		})

	case "notifications/initialized":
		return successResponse(req.ID, map[string]any{})

	case "tools/list":
		return successResponse(req.ID, map[string]any{"tools": toolDefinitions})

	case "tools/call":
		var params toolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return errorResponse(req.ID, -32602, "Invalid params")
		}
		content, isError := h.service.CallTool(params.Name, params.Arguments, scope)
		return successResponse(req.ID, map[string]any{
			"content": content,
			"isError": isError,
		})

	case "ping":
		return successResponse(req.ID, map[string]any{})

	default:
		return errorResponse(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

// config returns a ready-to-paste Claude MCP config snippet —
// project-scoped when requested via /mcp/:project/config.json.
func (h *Handler) config(c echo.Context) error {
	name := "taskdock"
	url := fmt.Sprintf("http://localhost:%s/mcp", h.port)
	if scope := c.Param("project"); scope != "" {
		name = "taskdock-" + strings.ToLower(scope)
		url += "/" + strings.ToUpper(scope)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"mcpServers": map[string]any{
			name: map[string]any{
				"type": "http",
				"url":  url,
			},
		},
	})
}

func successResponse(id any, result any) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "result": result}
}

func errorResponse(id any, code int, message string) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   map[string]any{"code": code, "message": message},
	}
}
