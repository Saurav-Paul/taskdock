package issues

// Request/response shapes for the issues API.

import (
	"sort"
	"time"
)

// IssueCreate is the JSON body for POST /api/issues.
type IssueCreate struct {
	Project     string   `json:"project"` // project key, e.g. "TD"
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`   // defaults to "todo"
	Priority    string   `json:"priority"` // defaults to "none"
	Assignee    string   `json:"assignee"` // user handle, e.g. "claude"; empty = unassigned
	Labels      []string `json:"labels"`
	Parent      string   `json:"parent"`     // parent issue key — makes this a subtask
	DependsOn   []string `json:"depends_on"` // issue keys this issue is blocked by
}

// IssueUpdate is the JSON body for PATCH /api/issues/:key.
// Pointer fields distinguish "not sent" (nil) from "sent as empty"
// — sending "assignee": "" unassigns the issue.
type IssueUpdate struct {
	Title       *string   `json:"title,omitempty"`
	Description *string   `json:"description,omitempty"`
	Status      *string   `json:"status,omitempty"`
	Priority    *string   `json:"priority,omitempty"`
	Assignee    *string   `json:"assignee,omitempty"`
	Labels      *[]string `json:"labels,omitempty"`
	Parent      *string   `json:"parent,omitempty"`     // "" detaches from parent
	DependsOn   *[]string `json:"depends_on,omitempty"` // replaces the set
}

// ListFilters are the query parameters for GET /api/issues.
type ListFilters struct {
	Project  string // project key
	Status   string
	Assignee string // user handle
	Query    string // substring match on title/description
	Limit    int
}

// IssueRef is a lightweight reference to a related issue
// (parent, subtask, dependency).
type IssueRef struct {
	Key    string `json:"key"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// IssueResponse is the JSON shape returned for an issue.
type IssueResponse struct {
	ID          uint       `json:"id"`
	Key         string     `json:"key"` // e.g. "TD-12"
	Project     string     `json:"project"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	Assignee    string     `json:"assignee,omitempty"`
	Labels      []string   `json:"labels"`
	Parent      *IssueRef  `json:"parent,omitempty"`
	Subtasks    []IssueRef `json:"subtasks"`
	DependsOn   []IssueRef `json:"depends_on"`
	Blocks      []IssueRef `json:"blocks"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ToResponse converts a loaded Issue model (with Project/Assignee/Labels/
// Parent/Subtasks/DependsOn/Blocks preloaded) into the API response shape.
func ToResponse(i *Issue) IssueResponse {
	resp := IssueResponse{
		ID:          i.ID,
		Key:         i.Key(),
		Project:     i.Project.Key,
		Title:       i.Title,
		Description: i.Description,
		Status:      i.Status,
		Priority:    i.Priority,
		Labels:      make([]string, 0, len(i.Labels)),
		Subtasks:    toRefs(i.Subtasks),
		DependsOn:   toRefs(i.DependsOn),
		Blocks:      toRefs(i.Blocks),
		CreatedAt:   i.CreatedAt,
		UpdatedAt:   i.UpdatedAt,
	}
	if i.Assignee != nil {
		resp.Assignee = i.Assignee.Name
	}
	for _, l := range i.Labels {
		resp.Labels = append(resp.Labels, l.Name)
	}
	if i.Parent != nil {
		ref := toRef(i.Parent)
		resp.Parent = &ref
	}
	return resp
}

func toRef(i *Issue) IssueRef {
	return IssueRef{Key: i.Key(), Title: i.Title, Status: i.Status}
}

func toRefs(rows []Issue) []IssueRef {
	sort.Slice(rows, func(a, b int) bool { return rows[a].Number < rows[b].Number })
	refs := make([]IssueRef, 0, len(rows))
	for idx := range rows {
		refs = append(refs, toRef(&rows[idx]))
	}
	return refs
}
