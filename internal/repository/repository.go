package repository

import (
	"context"
	"time"

	"github.com/Houeta/us-api-provider/internal/metrics"
	"github.com/Houeta/us-api-provider/internal/models"
)

type Repository struct {
	db      Database
	metrics *metrics.Metrics
}

type StatusRepoIface interface {
	SaveProcessedDate(ctx context.Context, date time.Time) error
	GetLastProcessedDate(ctx context.Context) (time.Time, error)
}

func NewStatusRepository(db Database, metrics *metrics.Metrics) StatusRepoIface {
	return &Repository{db: db, metrics: metrics}
}

// EmployeeRepoIface represents the interface for interacting with employee data in the repository.
type EmployeeRepoIface interface {
	SaveEmployee(ctx context.Context, identifier int, fullname, shortname, position, email, phone string) error
	UpdateEmployee(ctx context.Context, identifier int, fullname, shortname, position, email, phone string) error
	GetEmployeeByID(ctx context.Context, identifier int) (models.Employee, error)
}

func NewEmployeeRepository(db Database, metrics *metrics.Metrics) EmployeeRepoIface {
	return &Repository{db: db, metrics: metrics}
}

// TaskRepoIface represents the interface for interacting with task data in the repository.
type TaskRepoIface interface {
	GetOrCreateTaskTypeID(ctx context.Context, typeName string) (int, error)
	UpsertTask(ctx context.Context, task models.Task, typeID int) error
	UpdateTaskExecutors(ctx context.Context, taskID int, executors []string) error
	SaveTaskData(ctx context.Context, task models.Task) error
}

func NewTaskRepository(db Database, metrics *metrics.Metrics) TaskRepoIface {
	return &Repository{db: db, metrics: metrics}
}
