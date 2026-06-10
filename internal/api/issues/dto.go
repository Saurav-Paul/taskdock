package issues

// Request/response shapes for the issues API.

import "time"

// IssueCreate is the JSON body for POST /api/issues.
type IssueCreate struct {
	Project     string   `json:"project"` // project key, e.g. "TD"
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`   // defaults to "todo"
	Priority    string   `json:"priority"` // defaults to "none"
	Assignee    string   `json:"assignee"` // user handle, e.g. "claude"; empty = unassigned
	Labels      []string `json:"labels"`
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
}

// ListFilters are the query parameters for GET /api/issues.
type ListFilters struct {
	Project  string // project key
	Status   string
	Assignee string // user handle
	Query    string // substring match on title/description
	Limit    int
}

// IssueResponse is the JSON shape returned for an issue.
type IssueResponse struct {
	ID          uint      `json:"id"`
	Key         string    `json:"key"` // e.g. "TD-12"
	Project     string    `json:"project"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	Assignee    string    `json:"assignee,omitempty"`
	Labels      []string  `json:"labels"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToResponse converts a loaded Issue model (with Project/Assignee/Labels
// preloaded) into the API response shape.
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
		CreatedAt:   i.CreatedAt,
		UpdatedAt:   i.UpdatedAt,
	}
	if i.Assignee != nil {
		resp.Assignee = i.Assignee.Name
	}
	for _, l := range i.Labels {
		resp.Labels = append(resp.Labels, l.Name)
	}
	return resp
}
