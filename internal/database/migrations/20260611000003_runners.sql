-- +goose Up
-- Runners: dispatcher processes register themselves as assignable users
-- (kind=runner). Assigning a ticket to a runner routes the work to the
-- terminal/path where that dispatcher runs. Online = recent last_seen.
ALTER TABLE users ADD COLUMN kind TEXT NOT NULL DEFAULT 'person';
ALTER TABLE users ADD COLUMN path TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN hostname TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN last_seen DATETIME;

-- +goose Down
ALTER TABLE users DROP COLUMN last_seen;
ALTER TABLE users DROP COLUMN hostname;
ALTER TABLE users DROP COLUMN path;
ALTER TABLE users DROP COLUMN kind;
