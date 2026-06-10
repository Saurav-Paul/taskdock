-- +goose Up
-- Initial schema for taskdock: users, projects, issues, labels, comments.

-- Users: no auth in v1 — these are simple identity rows.
-- The "claude" user exists so issues can be assigned to Claude Code via MCP.
CREATE TABLE IF NOT EXISTS users (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    name         TEXT     NOT NULL UNIQUE,        -- short handle, e.g. "saurav", "claude"
    display_name TEXT     NOT NULL DEFAULT '',
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Projects: issues are grouped by project; "key" prefixes issue identifiers (e.g. TD-12).
CREATE TABLE IF NOT EXISTS projects (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT     NOT NULL,
    key         TEXT     NOT NULL UNIQUE,         -- short uppercase key, e.g. "TD"
    description TEXT     NOT NULL DEFAULT '',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Issues: "number" is a per-project sequence — the public identifier is <project.key>-<number>.
CREATE TABLE IF NOT EXISTS issues (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id  INTEGER  NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    number      INTEGER  NOT NULL,
    title       TEXT     NOT NULL,
    description TEXT     NOT NULL DEFAULT '',     -- markdown
    status      TEXT     NOT NULL DEFAULT 'todo'
                CHECK (status IN ('backlog', 'todo', 'in_progress', 'done', 'canceled')),
    priority    TEXT     NOT NULL DEFAULT 'none'
                CHECK (priority IN ('none', 'low', 'medium', 'high', 'urgent')),
    assignee_id INTEGER  REFERENCES users(id) ON DELETE SET NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (project_id, number)
);

CREATE INDEX IF NOT EXISTS ix_issues_project  ON issues(project_id);
CREATE INDEX IF NOT EXISTS ix_issues_status   ON issues(status);
CREATE INDEX IF NOT EXISTS ix_issues_assignee ON issues(assignee_id);

-- Labels: global (not per-project) in v1.
CREATE TABLE IF NOT EXISTS labels (
    id    INTEGER PRIMARY KEY AUTOINCREMENT,
    name  TEXT NOT NULL UNIQUE,
    color TEXT NOT NULL DEFAULT '#8b8b8b'
);

CREATE TABLE IF NOT EXISTS issue_labels (
    issue_id INTEGER NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    label_id INTEGER NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
    PRIMARY KEY (issue_id, label_id)
);

-- Comments: markdown thread per issue; Claude logs work progress here via MCP.
CREATE TABLE IF NOT EXISTS comments (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    issue_id   INTEGER  NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    author_id  INTEGER  REFERENCES users(id) ON DELETE SET NULL,
    body       TEXT     NOT NULL,                 -- markdown
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS ix_comments_issue ON comments(issue_id);

-- Seed the two standing users.
INSERT INTO users (name, display_name) VALUES
    ('saurav', 'Saurav Paul'),
    ('claude', 'Claude');

-- +goose Down
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS issue_labels;
DROP TABLE IF EXISTS labels;
DROP TABLE IF EXISTS issues;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS users;
