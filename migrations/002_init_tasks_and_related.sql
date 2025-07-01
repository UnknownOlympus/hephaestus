-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS scraper_status (
		id SERIAL PRIMARY KEY,
		last_processed_date DATE NOT NULL,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS task_types (
    type_id SERIAL PRIMARY KEY,
    type_name VARCHAR(50) UNIQUE NOT NULL
);

SET datestyle = dmy;

CREATE TABLE IF NOT EXISTS tasks (
    task_id INTEGER PRIMARY KEY,
    task_type_id INTEGER NOT NULL REFERENCES task_types(type_id),
    creation_date DATE NOT NULL,
    closing_date DATE,
    description TEXT,
    address TEXT,
    customer_name VARCHAR(255),
    customer_login VARCHAR(255), 
    comments TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS task_executors (
    task_id INTEGER NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    executor_id INTEGER NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, executor_id)
);

CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER set_timestamp
BEFORE UPDATE ON tasks
FOR EACH ROW
EXECUTE PROCEDURE update_timestamp();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_timestamp on tasks;
DROP FUNCTION IF EXISTS update_timestamp();
DROP TABLE IF EXISTS task_executors;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS request_types;
DROP TABLE IF EXISTS scraper_status;
-- +goose StatementEnd
