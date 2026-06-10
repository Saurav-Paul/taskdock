package mcp

// MCP tool definitions — name, description, and JSON Schema for arguments.
// Same shape as shelf's MCP_TOOL_DEFINITIONS.

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
					"type":        "string",
					"enum":        []string{"backlog", "todo", "in_progress", "done", "canceled"},
					"description": "Filter by status",
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
		Description: "Create or update an issue. If 'key' is provided, updates that issue (only the fields you pass change). Otherwise creates a new issue — 'project' and 'title' are required for creation.",
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
					"enum": []string{"backlog", "todo", "in_progress", "done", "canceled"},
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
		Description: "Add a markdown comment to an issue. Use this to log progress, decisions, or results while working on a ticket.",
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
		Name:        "taskdock_get_next_task",
		Description: "Get the highest-priority unstarted issue (status backlog or todo) assigned to a user. Defaults to 'claude' — call this to find out what to work on next.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"assignee": map[string]any{
					"type":        "string",
					"default":     "claude",
					"description": "Assignee handle; defaults to 'claude'",
				},
			},
		},
	},
}
