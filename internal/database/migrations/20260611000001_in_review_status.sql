-- +goose NO TRANSACTION
-- +goose Up
-- SQLite can't alter a CHECK constraint, so adding the 'in_review' status
-- means rebuilding the issues table (the documented 12-step recipe).
-- Also adds started_at/completed_at, stamped on the first transition into
-- in_progress and done/canceled respectively. FKs from other tables survive
-- because ids are preserved and the table name is restored by the rename.
PRAGMA foreign_keys = OFF;

CREATE TABLE issues_new (
    id           INTEGER  PRIMARY KEY AUTOINCREMENT,
    project_id   INTEGER  NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    number       INTEGER  NOT NULL,
    title        TEXT     NOT NULL,
    description  TEXT     NOT NULL DEFAULT '',
    status       TEXT     NOT NULL DEFAULT 'todo'
                 CHECK (status IN ('backlog', 'todo', 'in_progress', 'in_review', 'done', 'canceled')),
    priority     TEXT     NOT NULL DEFAULT 'none'
                 CHECK (priority IN ('none', 'low', 'medium', 'high', 'urgent')),
    assignee_id  INTEGER  REFERENCES users(id) ON DELETE SET NULL,
    parent_id    INTEGER  REFERENCES issues(id) ON DELETE CASCADE,
    started_at   DATETIME,
    completed_at DATETIME,
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (project_id, number)
);

INSERT INTO issues_new
    (id, project_id, number, title, description, status, priority,
     assignee_id, parent_id, created_at, updated_at)
SELECT id, project_id, number, title, description, status, priority,
       assignee_id, parent_id, created_at, updated_at
FROM issues;

DROP TABLE issues;
ALTER TABLE issues_new RENAME TO issues;

CREATE INDEX IF NOT EXISTS ix_issues_project  ON issues(project_id);
CREATE INDEX IF NOT EXISTS ix_issues_status   ON issues(status);
CREATE INDEX IF NOT EXISTS ix_issues_assignee ON issues(assignee_id);
CREATE INDEX IF NOT EXISTS ix_issues_parent   ON issues(parent_id);

PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;

CREATE TABLE issues_old (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    project_id  INTEGER  NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    number      INTEGER  NOT NULL,
    title       TEXT     NOT NULL,
    description TEXT     NOT NULL DEFAULT '',
    status      TEXT     NOT NULL DEFAULT 'todo'
                CHECK (status IN ('backlog', 'todo', 'in_progress', 'done', 'canceled')),
    priority    TEXT     NOT NULL DEFAULT 'none'
                CHECK (priority IN ('none', 'low', 'medium', 'high', 'urgent')),
    assignee_id INTEGER  REFERENCES users(id) ON DELETE SET NULL,
    parent_id   INTEGER  REFERENCES issues(id) ON DELETE CASCADE,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (project_id, number)
);

INSERT INTO issues_old
    (id, project_id, number, title, description, status, priority,
     assignee_id, parent_id, created_at, updated_at)
SELECT id, project_id, number, title, description,
       CASE WHEN status = 'in_review' THEN 'in_progress' ELSE status END,
       priority, assignee_id, parent_id, created_at, updated_at
FROM issues;

DROP TABLE issues;
ALTER TABLE issues_old RENAME TO issues;

CREATE INDEX IF NOT EXISTS ix_issues_project  ON issues(project_id);
CREATE INDEX IF NOT EXISTS ix_issues_status   ON issues(status);
CREATE INDEX IF NOT EXISTS ix_issues_assignee ON issues(assignee_id);
CREATE INDEX IF NOT EXISTS ix_issues_parent   ON issues(parent_id);

PRAGMA foreign_keys = ON;
