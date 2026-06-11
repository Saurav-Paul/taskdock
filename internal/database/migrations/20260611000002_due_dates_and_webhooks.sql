-- +goose Up
-- Due dates: date-only (YYYY-MM-DD), no time component — deadlines, not
-- appointments. Overdue issues jump the priority queue in get_next_task.
ALTER TABLE issues ADD COLUMN due_date TEXT;

-- Webhooks: per-project URL POSTed on issue/comment events. Empty = off.
-- The intended consumer: assign an issue to claude -> webhook fires -> a
-- script launches a Claude Code session that picks up the task.
ALTER TABLE projects ADD COLUMN webhook_url TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE issues DROP COLUMN due_date;
ALTER TABLE projects DROP COLUMN webhook_url;
