package repository

import (
	"context"
	"fmt"

	"github.com/Houeta/us-api-provider/internal/models"
)

// SaveEmployee saves an employee to the database. It inserts a new record with the provided details
// unless an employee with the same identifier already exists.
func (r *Repository) SaveEmployee(ctx context.Context, identifier int, fullname, position, email, phone string) error {
	query := `
		INSERT INTO employees (id, fullname, position, email, phone)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING;
	`

	_, err := r.db.Exec(ctx, query, identifier, fullname, position, email, phone)
	if err != nil {
		return fmt.Errorf("failed to save employee: %w", err)
	}

	return nil
}

// UpdateEmployee updates an employee's information in the database.
func (r *Repository) UpdateEmployee(ctx context.Context, identifier int, fullname, position, email, phone string,
) error {
	query := `
		UPDATE employees
		SET fullname = $2, position = $3, email = $4, phone = $5, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1;
	`

	_, err := r.db.Exec(ctx, query, identifier, fullname, position, email, phone)
	if err != nil {
		return fmt.Errorf("failed to update employee data: %w", err)
	}

	return nil
}

// GetEmployeeByID retrieves an employee from the database by their ID.
func (r *Repository) GetEmployeeByID(ctx context.Context, identifier int) (models.Employee, error) {
	var result models.Employee

	query := `SELECT id, fullname, position, email, phone FROM employees WHERE id=$1`

	err := r.db.QueryRow(ctx, query, identifier).Scan(
		&result.ID, &result.FullName, &result.Position, &result.Email, &result.Phone)
	if err != nil {
		return models.Employee{}, fmt.Errorf("failed to get employee by id: %w", err)
	}

	return result, nil
}
