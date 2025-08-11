package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/UnknownOlympus/hephaestus/internal/models"
)

// SaveEmployee saves an employee to the database. It inserts a new record with the provided details
// unless an employee with the same identifier already exists.
func (r *Repository) SaveEmployee(
	ctx context.Context,
	identifier int,
	fullname, shortname, position, email, phone string,
) error {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		r.metrics.DBQueryDuration.WithLabelValues("save_employee").Observe(duration)
	}()
	query := `
		INSERT INTO employees (id, fullname, shortname, position, email, phone)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO NOTHING;
	`

	_, err := r.db.Exec(ctx, query, identifier, fullname, shortname, position, email, phone)
	if err != nil {
		return fmt.Errorf("failed to save employee: %w", err)
	}

	return nil
}

// UpdateEmployee updates an employee's information in the database.
func (r *Repository) UpdateEmployee(
	ctx context.Context,
	identifier int,
	fullname, shortname, position, email, phone string,
) error {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		r.metrics.DBQueryDuration.WithLabelValues("update_employee").Observe(duration)
	}()
	query := `
		UPDATE employees
		SET fullname = $2, shortname = $3, position = $4, email = $5, phone = $6, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1;
	`

	_, err := r.db.Exec(ctx, query, identifier, fullname, shortname, position, email, phone)
	if err != nil {
		return fmt.Errorf("failed to update employee data: %w", err)
	}

	return nil
}

// GetEmployeeByID retrieves an employee from the database by their ID.
func (r *Repository) GetEmployeeByID(ctx context.Context, identifier int) (models.Employee, error) {
	var result models.Employee

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		r.metrics.DBQueryDuration.WithLabelValues("get_employee_by_id").Observe(duration)
	}()
	query := `SELECT id, fullname, shortname, position, email, phone FROM employees WHERE id=$1`

	err := r.db.QueryRow(ctx, query, identifier).Scan(
		&result.ID, &result.FullName, &result.ShortName, &result.Position, &result.Email, &result.Phone)
	if err != nil {
		return models.Employee{}, fmt.Errorf("failed to get employee by id: %w", err)
	}

	return result, nil
}
