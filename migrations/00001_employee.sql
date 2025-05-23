-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS employees (
		id int PRIMARY KEY,
		fullname VARCHAR(255) NOT NULL,
		position VARCHAR(255),
        email VARCHAR(255) NOT NULL UNIQUE,
		phone VARCHAR(255),
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS employees;
-- +goose StatementEnd
