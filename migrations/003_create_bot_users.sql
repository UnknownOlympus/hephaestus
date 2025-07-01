-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS bot_users (
    telegram_id BIGINT PRIMARY KEY,
    employee_id INTEGER NOT NULL UNIQUE REFERENCES employees(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS bot_users;
-- +goose StatementEnd
