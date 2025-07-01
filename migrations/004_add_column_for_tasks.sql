-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks
ADD COLUMN is_closed BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX idx_tasks_is_closed ON tasks (is_closed);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_tasks_is_closed;
-- +goose StatementEnd
