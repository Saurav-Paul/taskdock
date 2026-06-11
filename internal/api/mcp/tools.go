package mcp

// MCP tool definitions — name, description, and JSON Schema for arguments.
// Same shape as shelf's MCP_TOOL_DEFINITIONS.

import "github.com/Saurav-Paul/taskdock/internal/api/issues"

// statusEnum single-sources the status list from the issues domain, so a
// new status only needs adding in one place.
var statusEnum = issues.ValidStatuses

type toolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

var toolDefinitions = []toolDefinition{
	{
		Name:        "taskdock_list_projects",
		Description: "List all projects with their keys (e.g. TD), names, and descriptions.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
	{
		Name:        "taskdock_list_issues",
		Description: "List issues, optionally filtered by project, status, assignee, or a search query. Returns key, title, status, priority, assignee, and labels for each issue.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"project": map[string]any{
					"type":        "string",
					"description": "Filter by project key, e.g. 'TD'",
				},
				"status": map[string]any{
					"description": "Filter by status — a single value or an array (e.g. [\"in_progress\", \"in_review\"] for everything active)",
					"anyOf": []map[string]any{
						{"type": "string", "enum": statusEnum},
						{"type": "array", "items": map[string]any{"type": "string", "enum": statusEnum}},
					},
				},
				"assignee": map[string]any{
					"type":        "string",
					"description": "Filter by assignee handle, e.g. 'claude' or 'saurav'",
				},
				"query": map[string]any{
					"type":        "string",
					"description": "Substring search in title and description",
				},
				"limit": map[string]any{
					"type":        "integer",
					"default":     25,
					"description": "Maximum number of issues to return",
				},
			},
		},
	},
	{
		Name:        "taskdock_get_issue",
		Description: "Get the full details of one issue by key (e.g. TD-12), including its markdown description and all comments.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"key": map[string]any{
					"type":        "string",
					"description": "Issue key, e.g. 'TD-12'",
				},
			},
			"required": []string{"key"},
		},
	},
	{
		Name:        "taskdock_save_issue",
		Description: "Create or update an issue. If 'key' is provided, updates that issue (only the fields you pass change). Otherwise creates a new issue — 'project' and 'title' are required for creation. To embed an image in the description, first upload it outside MCP (curl -F 'file=@img.png;type=image/png' <server>/api/attachments → {\"url\": \"/files/...\"}), then reference it in the markdown as ![](/files/...).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"key": map[string]any{
					"type":        "string",
					"description": "Issue key to update, e.g. 'TD-12'. Omit to create a new issue.",
				},
				"project": map[string]any{
					"type":        "string",
					"description": "Project key for new issues, e.g. 'TD'",
				},
				"title": map[string]any{
					"type":        "string",
					"description": "Issue title",
				},
				"description": map[string]any{
					"type":        "string",
					"description": "Issue description in markdown",
				},
				"status": map[string]any{
					"type": "string",
					"enum": statusEnum,
				},
				"priority": map[string]any{
					"type": "string",
					"enum": []string{"none", "low", "medium", "high", "urgent"},
				},
				"assignee": map[string]any{
					"type":        "string",
					"description": "Assignee handle, e.g. 'claude'. Pass an empty string to unassign.",
				},
				"labels": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string"},
					"description": "Label names; replaces the issue's current labels. Unknown labels are created.",
				},
				"parent": map[string]any{
					"type":        "string",
					"description": "Parent issue key to make this a subtask, e.g. 'TD-3'. Pass an empty string to detach.",
				},
				"depends_on": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string"},
					"description": "Issue keys this issue is blocked by; replaces the current set. Pass [] to clear.",
				},
				"number": map[string]any{
					"type":        "integer",
					"description": "Explicit issue number when importing from another tracker (e.g. 951 → PRO-951). Omit for the next auto number. Creation only.",
				},
				"due_date": map[string]any{
					"type":        "string",
					"description": "Deadline as YYYY-MM-DD. Pass an empty string to clear. Issues due today or overdue jump the priority queue in get_next_task.",
				},
			},
		},
	},
	{
		Name:        "taskdock_delete_issue",
		Description: "Delete an issue by key (e.g. TD-12). This also deletes its comments. Irreversible.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"key": map[string]any{
					"type":        "string",
					"description": "Issue key, e.g. 'TD-12'",
				},
			},
			"required": []string{"key"},
		},
	},
	{
		Name:        "taskdock_save_comment",
		Description: "Add a markdown comment to an issue. Use this to log progress, decisions, or results while working on a ticket. Images work the same as in descriptions: upload via curl -F 'file=@img.png;type=image/png' <server>/api/attachments, then embed the returned url as ![](/files/...).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"issue": map[string]any{
					"type":        "string",
					"description": "Issue key, e.g. 'TD-12'",
				},
				"body": map[string]any{
					"type":        "string",
					"description": "Comment body in markdown",
				},
				"author": map[string]any{
					"type":        "string",
					"default":     "claude",
					"description": "Author handle; defaults to 'claude'",
				},
			},
			"required": []string{"issue", "body"},
		},
	},
	{
		Name:        "taskdock_save_link",
		Description: "Attach a URL to an issue — use this to link the PR you opened for a ticket, or any related doc. Links appear in get_issue output, so 'find the PR for this issue' is a get_issue call.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"issue": map[string]any{
					"type":        "string",
					"description": "Issue key, e.g. 'TD-12'",
				},
				"url": map[string]any{
					"type":        "string",
					"description": "The URL to attach (http/https)",
				},
				"title": map[string]any{
					"type":        "string",
					"description": "Optional display title, e.g. 'PR #42'",
				},
			},
			"required": []string{"issue", "url"},
		},
	},
	{
		Name:        "taskdock_get_next_task",
		Description: "Get the highest-priority unstarted issue (status backlog or todo) assigned to a user, skipping issues blocked by unfinished dependencies. Defaults to 'claude' — call this to find out what to work on next.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"assignee": map[string]any{
					"type":        "string",
					"default":     "claude",
					"description": "Assignee handle; defaults to 'claude'",
				},
				"project": map[string]any{
					"type":        "string",
					"description": "Optional project key to restrict the search, e.g. 'TD'",
				},
			},
		},
	},
}
