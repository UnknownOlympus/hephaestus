-- +goose Up
-- +goose StatementBegin
ALTER TABLE IF EXISTS tasks
ADD COLUMN is_closed BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_tasks_is_closed ON tasks (is_closed);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_tasks_is_closed;
-- +goose StatementEnd
