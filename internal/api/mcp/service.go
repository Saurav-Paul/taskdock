package mcp

// Tool implementations — each tool delegates to the domain services and
// returns markdown/JSON text for the MCP client.

import (
	"encoding/json"
	"fmt"
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
}

// NewService creates the MCP tool service with its domain dependencies.
func NewService(p *projects.Service, i *issues.Service, c *comments.Service) *Service {
	return &Service{projects: p, issues: i, comments: c}
}

// CallTool executes a tool by name. Returns the text content and an isError flag.
func (s *Service) CallTool(name string, args map[string]any) (string, bool) {
	switch name {
	case "taskdock_list_projects":
		return s.listProjects()
	case "taskdock_list_issues":
		return s.listIssues(args)
	case "taskdock_get_issue":
		return s.getIssue(args)
	case "taskdock_save_issue":
		return s.saveIssue(args)
	case "taskdock_delete_issue":
		return s.deleteIssue(args)
	case "taskdock_save_comment":
		return s.saveComment(args)
	case "taskdock_get_next_task":
		return s.getNextTask(args)
	default:
		return fmt.Sprintf("Unknown tool: %s", name), true
	}
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

func (s *Service) getIssue(args map[string]any) (string, bool) {
	key := argString(args, "key")
	if key == "" {
		return "'key' is required", true
	}

	issue, err := s.issues.Get(key)
	if err != nil {
		return fmt.Sprintf("Issue not found: %s", key), true
	}

	commentRows, err := s.comments.ListForIssue(key)
	if err != nil {
		return err.Error(), true
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
	fmt.Fprintf(&b, "- **Created:** %s | **Updated:** %s\n", issue.CreatedAt.Format("2006-01-02 15:04"), issue.UpdatedAt.Format("2006-01-02 15:04"))

	if issue.Description != "" {
		fmt.Fprintf(&b, "\n## Description\n\n%s\n", issue.Description)
	}

	if len(commentRows) > 0 {
		fmt.Fprintf(&b, "\n## Comments (%d)\n", len(commentRows))
		for _, comment := range commentRows {
			fmt.Fprintf(&b, "\n**%s** (%s):\n%s\n", comment.Author, comment.CreatedAt.Format("2006-01-02 15:04"), comment.Body)
		}
	}

	return b.String(), false
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
	}
	if v, ok := args["labels"]; ok {
		create.Labels = argStringSlice(v)
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

	issue, err := s.issues.NextTask(assignee)
	if err != nil {
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
