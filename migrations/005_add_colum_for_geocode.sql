-- +goose Up
-- +goose StatementBegin
ALTER TABLE IF EXISTS tasks
ADD COLUMN latitude DECIMAL(10, 7),
ADD COLUMN longitude DECIMAL(10, 7),
ADD COLUMN geocoding_attempts INT DEFAULT 0,
ADD COLUMN geocoding_error TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
