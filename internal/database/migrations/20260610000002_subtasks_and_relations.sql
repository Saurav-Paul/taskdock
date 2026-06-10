-- +goose Up
-- Subtasks: an issue with a parent_id is a subtask of that issue (Linear's
-- sub-issue model). Deleting a parent cascades to its subtasks.
ALTER TABLE issues ADD COLUMN parent_id INTEGER REFERENCES issues(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS ix_issues_parent ON issues(parent_id);

-- Dependencies: "issue_id depends on (is blocked by) depends_on_id".
-- The reverse direction ("blocks") is derived by querying the other column.
CREATE TABLE IF NOT EXISTS issue_relations (
    issue_id      INTEGER NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    depends_on_id INTEGER NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    PRIMARY KEY (issue_id, depends_on_id)
);

CREATE INDEX IF NOT EXISTS ix_issue_relations_depends_on ON issue_relations(depends_on_id);

-- +goose Down
DROP TABLE IF EXISTS issue_relations;
DROP INDEX IF EXISTS ix_issues_parent;
ALTER TABLE issues DROP COLUMN parent_id;
