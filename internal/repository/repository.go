package repository

import (
	"context"
	"time"

	"github.com/Houeta/us-api-provider/internal/models"
)

type Repository struct {
	db Database
}

type StatusRepoIface interface {
	SaveLastProcessedDate(ctx context.Context, date time.Time) error
	GetLastProcessedDate(ctx context.Context) (time.Time, error)
}

func NewStatusRepository(db Database) StatusRepoIface {
	return &Repository{db: db}
}

// EmployeeRepoIface represents the interface for interacting with employee data in the repository.
type EmployeeRepoIface interface {
	SaveEmployee(ctx context.Context, identifier int, fullname, position, email, phone string) error
	UpdateEmployee(ctx context.Context, identifier int, fullname, position, email, phone string) error
	GetEmployeeByID(ctx context.Context, identifier int) (models.Employee, error)
}

func NewEmployeeRepository(db Database) EmployeeRepoIface {
	return &Repository{db: db}
}
