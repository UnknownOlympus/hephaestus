-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS tasks_lat_lon_idx ON tasks (latitude, longitude);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS tasks_lat_lon_idx
-- +goose StatementEnd
