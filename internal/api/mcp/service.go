package mcp

// Tool implementations — each tool delegates to the domain services and
// returns markdown/JSON text for the MCP client.

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Saurav-Paul/taskdock/internal/api/comments"
	"github.com/Saurav-Paul/taskdock/internal/api/issues"
	"github.com/Saurav-Paul/taskdock/internal/api/projects"
)

// Service routes MCP tool calls to the domain services.
type Service struct {
	projects *projects.Service
	issues   *issues.Service
	comments *comments.Service
	filesDir string // where pasted-image attachments live on disk
}

// NewService creates the MCP tool service with its domain dependencies.
func NewService(p *projects.Service, i *issues.Service, c *comments.Service, filesDir string) *Service {
	return &Service{projects: p, issues: i, comments: c, filesDir: filesDir}
}

// CallTool executes a tool by name. Returns MCP content blocks and an
// isError flag. Most tools return a single text block; get_issue also
// attaches pasted images as image blocks so the model can see them.
// A non-empty scope (from /mcp/:project) locks the call to that project.
func (s *Service) CallTool(name string, args map[string]any, scope string) ([]map[string]any, bool) {
	if scope != "" {
		var errText string
		args, errText = s.applyScope(name, args, scope)
		if errText != "" {
			return textContent(errText), true
		}
	}

	if name == "taskdock_get_issue" {
		return s.getIssue(args)
	}

	var text string
	var isError bool
	switch name {
	case "taskdock_list_projects":
		text, isError = s.listProjects()
	case "taskdock_list_issues":
		text, isError = s.listIssues(args)
	case "taskdock_save_issue":
		text, isError = s.saveIssue(args)
	case "taskdock_delete_issue":
		text, isError = s.deleteIssue(args)
	case "taskdock_save_comment":
		text, isError = s.saveComment(args)
	case "taskdock_get_next_task":
		text, isError = s.getNextTask(args)
	default:
		text, isError = fmt.Sprintf("Unknown tool: %s", name), true
	}
	return textContent(text), isError
}

// textContent wraps a string in a single MCP text block.
func textContent(text string) []map[string]any {
	return []map[string]any{{"type": "text", "text": text}}
}

// applyScope rewrites tool arguments so the call stays inside one project:
// listing and next-task are filtered to it, new issues are created in it,
// and issue keys from other projects are rejected. Returns the adjusted
// args, or a non-empty error text.
func (s *Service) applyScope(name string, args map[string]any, scope string) (map[string]any, string) {
	project, err := s.projects.GetByKey(scope)
	if err != nil {
		return args, fmt.Sprintf("unknown project in MCP url: %s", scope)
	}

	if args == nil {
		args = map[string]any{}
	}

	// Reject direct references to issues outside the scoped project.
	for _, argName := range []string{"key", "issue"} {
		if v := argString(args, argName); v != "" &&
			!strings.HasPrefix(strings.ToUpper(v), project.Key+"-") {
			return args, fmt.Sprintf("issue %s is outside project %s (project-scoped MCP)", v, project.Key)
		}
	}

	switch name {
	case "taskdock_list_issues", "taskdock_get_next_task":
		args["project"] = project.Key
	case "taskdock_save_issue":
		if argString(args, "key") == "" { // create path → force the project
			args["project"] = project.Key
		}
	}
	return args, ""
}

func (s *Service) listProjects() (string, bool) {
	projectRows, err := s.projects.List()
	if err != nil {
		return err.Error(), true
	}
	if len(projectRows) == 0 {
		return "No projects yet.", false
	}
	return toJSON(projectRows), false
}

func (s *Service) listIssues(args map[string]any) (string, bool) {
	filters := issues.ListFilters{
		Project:  argString(args, "project"),
		Status:   argString(args, "status"),
		Assignee: argString(args, "assignee"),
		Query:    argString(args, "query"),
		Limit:    argInt(args, "limit", 25),
	}

	rows, err := s.issues.List(filters)
	if err != nil {
		return err.Error(), true
	}
	if len(rows) == 0 {
		return "No issues found.", false
	}
	return toJSON(rows), false
}

func (s *Service) getIssue(args map[string]any) ([]map[string]any, bool) {
	key := argString(args, "key")
	if key == "" {
		return textContent("'key' is required"), true
	}

	issue, err := s.issues.Get(key)
	if err != nil {
		return textContent(fmt.Sprintf("Issue not found: %s", key)), true
	}

	commentRows, err := s.comments.ListForIssue(key)
	if err != nil {
		return textContent(err.Error()), true
	}

	// Render the issue as markdown — nicer for the model to read than raw JSON.
	var b strings.Builder
	fmt.Fprintf(&b, "# [%s] %s\n\n", issue.Key, issue.Title)
	fmt.Fprintf(&b, "- **Project:** %s\n- **Status:** %s\n- **Priority:** %s\n", issue.Project, issue.Status, issue.Priority)
	if issue.Assignee != "" {
		fmt.Fprintf(&b, "- **Assignee:** %s\n", issue.Assignee)
	}
	if len(issue.Labels) > 0 {
		fmt.Fprintf(&b, "- **Labels:** %s\n", strings.Join(issue.Labels, ", "))
	}
	if issue.Parent != nil {
		fmt.Fprintf(&b, "- **Parent:** [%s] %s (%s)\n", issue.Parent.Key, issue.Parent.Title, issue.Parent.Status)
	}
	fmt.Fprintf(&b, "- **Branch:** %s\n", issue.Branch)
	fmt.Fprintf(&b, "- **Created:** %s | **Updated:** %s (UTC)\n", issue.CreatedAt.UTC().Format("2006-01-02 15:04"), issue.UpdatedAt.UTC().Format("2006-01-02 15:04"))

	if len(issue.DependsOn) > 0 {
		b.WriteString("\n## Blocked by\n\n")
		for _, ref := range issue.DependsOn {
			fmt.Fprintf(&b, "- [%s] %s (%s)\n", ref.Key, ref.Title, ref.Status)
		}
	}
	if len(issue.Blocks) > 0 {
		b.WriteString("\n## Blocks\n\n")
		for _, ref := range issue.Blocks {
			fmt.Fprintf(&b, "- [%s] %s (%s)\n", ref.Key, ref.Title, ref.Status)
		}
	}
	if len(issue.Subtasks) > 0 {
		fmt.Fprintf(&b, "\n## Subtasks (%d)\n\n", len(issue.Subtasks))
		for _, ref := range issue.Subtasks {
			fmt.Fprintf(&b, "- [%s] %s (%s)\n", ref.Key, ref.Title, ref.Status)
		}
	}

	if issue.Description != "" {
		fmt.Fprintf(&b, "\n## Description\n\n%s\n", issue.Description)
	}

	if len(commentRows) > 0 {
		fmt.Fprintf(&b, "\n## Comments (%d)\n", len(commentRows))
		for _, comment := range commentRows {
			fmt.Fprintf(&b, "\n**%s** (%s):\n%s\n", comment.Author, comment.CreatedAt.UTC().Format("2006-01-02 15:04"), comment.Body)
		}
	}

	// Attach pasted images as MCP image blocks so the model can actually
	// see them (the markdown above only carries their URLs).
	bodies := []string{issue.Description}
	for _, comment := range commentRows {
		bodies = append(bodies, comment.Body)
	}
	content := append(textContent(b.String()), s.imageBlocks(bodies)...)
	return content, false
}

// fileRefPattern matches attachment references in markdown: ![...](/files/x.png)
var fileRefPattern = regexp.MustCompile(`!\[[^\]]*\]\((/files/[^)\s]+)\)`)

// imageMimeTypes maps attachment extensions to MCP image block mime types.
// SVG is omitted — models can't view it as a raster image block.
var imageMimeTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
}

const (
	maxImageBlocks    = 6               // cap blocks per issue to keep responses sane
	maxImageBlockSize = 2 * 1024 * 1024 // skip files larger than 2MB
)

// imageBlocks finds /files/* image references in the given markdown bodies
// and returns them as base64 MCP image blocks.
func (s *Service) imageBlocks(bodies []string) []map[string]any {
	seen := map[string]bool{}
	blocks := []map[string]any{}

	for _, body := range bodies {
		for _, match := range fileRefPattern.FindAllStringSubmatch(body, -1) {
			url := match[1]
			if seen[url] || len(blocks) >= maxImageBlocks {
				continue
			}
			seen[url] = true

			mimeType, ok := imageMimeTypes[strings.ToLower(filepath.Ext(url))]
			if !ok {
				continue
			}

			// filepath.Base guards against path traversal in the URL.
			data, err := os.ReadFile(filepath.Join(s.filesDir, filepath.Base(url)))
			if err != nil || len(data) > maxImageBlockSize {
				continue // missing or oversized — the markdown link is still there
			}

			blocks = append(blocks, map[string]any{
				"type":     "image",
				"data":     base64.StdEncoding.EncodeToString(data),
				"mimeType": mimeType,
			})
		}
	}
	return blocks
}

func (s *Service) saveIssue(args map[string]any) (string, bool) {
	key := argString(args, "key")

	// Update path: key present.
	if key != "" {
		update := issues.IssueUpdate{}
		if v, ok := args["title"]; ok {
			str := fmt.Sprint(v)
			update.Title = &str
		}
		if v, ok := args["description"]; ok {
			str := fmt.Sprint(v)
			update.Description = &str
		}
		if v, ok := args["status"]; ok {
			str := fmt.Sprint(v)
			update.Status = &str
		}
		if v, ok := args["priority"]; ok {
			str := fmt.Sprint(v)
			update.Priority = &str
		}
		if v, ok := args["assignee"]; ok {
			str := fmt.Sprint(v)
			update.Assignee = &str
		}
		if v, ok := args["labels"]; ok {
			labelNames := argStringSlice(v)
			update.Labels = &labelNames
		}
		if v, ok := args["parent"]; ok {
			str := fmt.Sprint(v)
			update.Parent = &str
		}
		if v, ok := args["depends_on"]; ok {
			depKeys := argStringSlice(v)
			update.DependsOn = &depKeys
		}

		issue, err := s.issues.Update(key, update)
		if err != nil {
			return err.Error(), true
		}
		return "Updated:\n" + toJSON(issue), false
	}

	// Create path.
	create := issues.IssueCreate{
		Project:     argString(args, "project"),
		Title:       argString(args, "title"),
		Description: argString(args, "description"),
		Status:      argString(args, "status"),
		Priority:    argString(args, "priority"),
		Assignee:    argString(args, "assignee"),
		Parent:      argString(args, "parent"),
	}
	if v, ok := args["labels"]; ok {
		create.Labels = argStringSlice(v)
	}
	if v, ok := args["depends_on"]; ok {
		create.DependsOn = argStringSlice(v)
	}
	if create.Project == "" || create.Title == "" {
		return "'project' and 'title' are required to create an issue", true
	}

	issue, err := s.issues.Create(create)
	if err != nil {
		return err.Error(), true
	}
	return "Created:\n" + toJSON(issue), false
}

func (s *Service) deleteIssue(args map[string]any) (string, bool) {
	key := argString(args, "key")
	if key == "" {
		return "'key' is required", true
	}
	if err := s.issues.Delete(key); err != nil {
		return err.Error(), true
	}
	return fmt.Sprintf("Deleted %s.", key), false
}

func (s *Service) saveComment(args map[string]any) (string, bool) {
	issueKey := argString(args, "issue")
	body := argString(args, "body")
	if issueKey == "" || body == "" {
		return "'issue' and 'body' are required", true
	}

	author := argString(args, "author")
	if author == "" {
		author = "claude"
	}

	comment, err := s.comments.Create(issueKey, comments.CommentCreate{Author: author, Body: body})
	if err != nil {
		return err.Error(), true
	}
	return fmt.Sprintf("Comment added to %s (id %d).", issueKey, comment.ID), false
}

func (s *Service) getNextTask(args map[string]any) (string, bool) {
	assignee := argString(args, "assignee")
	if assignee == "" {
		assignee = "claude"
	}
	project := argString(args, "project")

	issue, err := s.issues.NextTask(assignee, project)
	if err != nil {
		if project != "" {
			return fmt.Sprintf("No open tasks for '%s' in project %s.", assignee, project), false
		}
		return fmt.Sprintf("No open tasks for '%s'.", assignee), false
	}
	return toJSON(issue), false
}

// --- argument helpers ---

func argString(args map[string]any, key string) string {
	if v, ok := args[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

func argInt(args map[string]any, key string, fallback int) int {
	if v, ok := args[key]; ok {
		// JSON numbers decode as float64 in Go.
		if f, ok := v.(float64); ok {
			return int(f)
		}
	}
	return fallback
}

func argStringSlice(v any) []string {
	out := []string{}
	if items, ok := v.([]any); ok {
		for _, item := range items {
			out = append(out, fmt.Sprint(item))
		}
	}
	return out
}

func toJSON(v any) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(data)
}
