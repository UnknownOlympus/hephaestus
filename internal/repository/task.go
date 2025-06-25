package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Houeta/us-api-provider/internal/models"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetOrCreateTaskTypeID(ctx context.Context, typeName string) (int, error) {
	var typeID int
	err := r.db.QueryRow(ctx, "SELECT type_id FROM task_types WHERE type_name = $1", typeName).Scan(&typeID)
	if err == nil {
		return typeID, nil // type is found, return id
	}

	insertQuery := `
		INSERT INTO task_types (type_name)
		VALUES ($1)
		ON CONFLICT (type_name) DO NOTHING;
	`
	if errors.Is(err, pgx.ErrNoRows) {
		// type not found, insert it
		_, err = r.db.Exec(ctx, insertQuery, typeName)
		if err != nil {
			return 0, fmt.Errorf("error inserting new task type '%s': %w", typeName, err)
		}
		// now, we get the ID again (this covers the case if another transaction inserted it between our queries)
		err = r.db.QueryRow(ctx, "SELECT type_id FROM task_types WHERE type_name = $1", typeName).Scan(&typeID)
		if err != nil {
			return 0, fmt.Errorf("error inserting ID for newly inserted task type '%s': %w", typeName, err)
		}
		return typeID, nil
	}

	return 0, fmt.Errorf("request error to `task_types`: %w", err)
}

func (r *Repository) SaveTaskData(ctx context.Context, task models.Task) error {
	// 1. Get ID for task type
	typeID, err := r.GetOrCreateTaskTypeID(ctx, task.Type)
	if err != nil {
		return fmt.Errorf("task type preparation error: %w", err)
	}

	// 2. Insert or update task
	err = r.UpsertTask(ctx, task, typeID)
	if err != nil {
		return fmt.Errorf("task insert/update error: %w", err)
	}

	// 3. Update executors for the task
	err = r.UpdateTaskExecutors(ctx, task.ID, task.Executors)
	if err != nil {
		return fmt.Errorf("error updating executors: %w", err)
	}

	return nil
}

func (r *Repository) UpsertTask(ctx context.Context, task models.Task, typeID int) error {
	isClosed := !task.ClosedAt.IsZero()

	query := `
		INSERT INTO tasks (
			task_id, task_type_id, creation_date, closing_date, description,
			address, customer_name, customer_login, comments, is_closed
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (task_id) DO UPDATE SET
			task_type_id = EXCLUDED.task_type_id,
			closing_date = EXCLUDED.closing_date,
			description = EXCLUDED.description,
			address = EXCLUDED.address,
			customer_name = EXCLUDED.customer_name,
			customer_login = EXCLUDED.customer_login,
			comments = EXCLUDED.comments,
			is_closed = EXCLUDED.is_closed,
			updated_at = CURRENT_TIMESTAMP;
	`
	_, err := r.db.Exec(ctx, query,
		task.ID, typeID, task.CreatedAt, task.ClosedAt, task.Description,
		task.Address, task.CustomerName, task.CustomerLogin, task.Comments, isClosed,
	)
	if err != nil {
		return fmt.Errorf("upsert task error for task '%d': %w", task.ID, err)
	}

	return nil
}

func (r *Repository) UpdateTaskExecutors(ctx context.Context, taskID int, executors []string) error {
	// 1. Delete all executors for this task
	_, err := r.db.Exec(ctx, "DELETE FROM task_executors WHERE task_id = $1", taskID)
	if err != nil {
		return fmt.Errorf("failed to delete existing executors for the task '%d': %w", taskID, err)
	}

	query := `
		INSERT INTO task_executors (task_id, executor_id)
		VALUES ($1, (SELECT id FROM employees WHERE shortname = $2));
	`

	// 2. Insert new executors
	for _, executorName := range executors {
		_, err = r.db.Exec(ctx, query, taskID, executorName)
		if err != nil {
			return fmt.Errorf("failed to save link between task '%d' and employee '%s': %w", taskID, executorName, err)
		}
	}

	return nil
}
