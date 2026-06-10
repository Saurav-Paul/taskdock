-- +goose Up
-- Per-project issue counter: numbers come from here instead of MAX(number)+1,
-- so deleted issues never get their numbers reused, and imports can preserve
-- original keys (creating PRO-951 bumps the counter past 951).
ALTER TABLE projects ADD COLUMN next_number INTEGER NOT NULL DEFAULT 1;

UPDATE projects
SET next_number = COALESCE(
    (SELECT MAX(number) FROM issues WHERE issues.project_id = projects.id), 0
) + 1;

-- URL attachments (PR links, docs, dashboards). Agents attach the PR they
-- opened for a ticket so "find the PR for this issue" is a get_issue read.
CREATE TABLE IF NOT EXISTS links (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    issue_id   INTEGER  NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    url        TEXT     NOT NULL,
    title      TEXT     NOT NULL DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS ix_links_issue ON links(issue_id);

-- +goose Down
DROP TABLE IF EXISTS links;
ALTER TABLE projects DROP COLUMN next_number;
