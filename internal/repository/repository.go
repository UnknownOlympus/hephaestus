package repository

import (
	"context"
	"time"

	"github.com/UnknownOlympus/hephaestus/internal/metrics"
	"github.com/UnknownOlympus/hephaestus/internal/models"
	pb "github.com/UnknownOlympus/olympus-protos/gen/go/scraper/olympus"
	"github.com/jackc/pgx/v5"
)

type Repository struct {
	db      Database
	metrics *metrics.Metrics
}

// StatusRepoIface defines the interface for a status repository.
// It provides methods to save and retrieve the last processed date.
type StatusRepoIface interface {
	// SaveProcessedDate saves the given date as the last processed date.
	// It returns an error if the operation fails.
	SaveProcessedDate(ctx context.Context, date time.Time) error

	// GetLastProcessedDate retrieves the last processed date.
	// It returns the date and an error if the operation fails.
	GetLastProcessedDate(ctx context.Context) (time.Time, error)
}

func NewStatusRepository(db Database, metrics *metrics.Metrics) StatusRepoIface {
	return &Repository{db: db, metrics: metrics}
}

// EmployeeRepoIface represents the interface for interacting with employee data in the repository.
type EmployeeRepoIface interface {
	SaveEmployee(
		ctx context.Context,
		identifier int,
		fullname, shortname, position, email, phone string,
		isAdmin bool,
	) error
	UpdateEmployee(
		ctx context.Context,
		identifier int,
		fullname, shortname, position, email, phone string,
		isAdmin bool,
	) error
	GetEmployeeByID(ctx context.Context, identifier int) (models.Employee, error)
}

func NewEmployeeRepository(db Database, metrics *metrics.Metrics) EmployeeRepoIface {
	return &Repository{db: db, metrics: metrics}
}

// TaskRepoIface represents the interface for interacting with task data in the repository.
type TaskRepoIface interface {
	GetOrCreateTaskTypeID(ctx context.Context, typeName string) (int, error)
	UpsertTask(ctx context.Context, tx pgx.Tx, task models.Task, typeID int) error
	UpdateTaskExecutors(ctx context.Context, tx pgx.Tx, taskID int, executors []string) error
	UpdateTaskCustomers(ctx context.Context, tx pgx.Tx, taskID int, customers []*pb.Customer) error
	SaveTaskData(ctx context.Context, task models.Task) error
}

func NewTaskRepository(db Database, metrics *metrics.Metrics) TaskRepoIface {
	return &Repository{db: db, metrics: metrics}
}
