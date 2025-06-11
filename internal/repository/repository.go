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
	SaveProcessedDate(ctx context.Context, date time.Time) error
	GetLastProcessedDate(ctx context.Context) (time.Time, error)
}

func NewStatusRepository(db Database) StatusRepoIface {
	return &Repository{db: db}
}

// EmployeeRepoIface represents the interface for interacting with employee data in the repository.
type EmployeeRepoIface interface {
	SaveEmployee(ctx context.Context, identifier int, fullname, shortname, position, email, phone string) error
	UpdateEmployee(ctx context.Context, identifier int, fullname, shortname, position, email, phone string) error
	GetEmployeeByID(ctx context.Context, identifier int) (models.Employee, error)
}

func NewEmployeeRepository(db Database) EmployeeRepoIface {
	return &Repository{db: db}
}

// TaskRepoIface represents the interface for interacting with task data in the repository.
type TaskRepoIface interface {
	GetOrCreateTaskTypeID(ctx context.Context, typeName string) (int, error)
	UpsertTask(ctx context.Context, task models.Task, typeID int) error
	UpdateTaskExecutors(ctx context.Context, taskID int, executors []string) error
	SaveTaskData(ctx context.Context, task models.Task) error
}

func NewTaskRepository(db Database) TaskRepoIface {
	return &Repository{db: db}
}
