package repository_test

import (
	"errors"
	"regexp"
	"testing"

	"github.com/UnknownOlympus/hephaestus/internal/models"
	"github.com/UnknownOlympus/hephaestus/internal/repository"
	"github.com/UnknownOlympus/olympus-protos/gen/go/scraper/olympus"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetOrCreateTaskTypeID checks the logic for getting or creating a task type ID.
func TestGetOrCreateTaskTypeID(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	t.Run("success - type exists", func(t *testing.T) {
		t.Parallel()
		// Creating a mock for the database
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock, repoMetrics)

		typeName := "Existing Type"
		expectedID := 1

		// We are waiting for a SELECT query that will find an existing type
		mock.ExpectQuery("SELECT type_id FROM task_types WHERE type_name = \\$1").
			WithArgs(typeName).
			WillReturnRows(pgxmock.NewRows([]string{"type_id"}).AddRow(expectedID))

		// Call function
		id, err := repo.GetOrCreateTaskTypeID(ctx, typeName)

		// Check result
		require.NoError(t, err)
		assert.Equal(t, expectedID, id)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success - type does not exist", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock, repoMetrics)

		typeName := "New Type"
		expectedID := 2

		// 1. We are expecting a SELECT that will return a "no rows" error.
		mock.ExpectQuery("SELECT type_id FROM task_types WHERE type_name = \\$1").
			WithArgs(typeName).
			WillReturnError(pgx.ErrNoRows)

		// 2. Waiting for INSERT to create a new type
		mock.ExpectExec("INSERT INTO task_types").
			WithArgs(typeName).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		// 3. We are waiting for the second SELECT, which will now find the created ID
		mock.ExpectQuery("SELECT type_id FROM task_types WHERE type_name = \\$1").
			WithArgs(typeName).
			WillReturnRows(pgxmock.NewRows([]string{"type_id"}).AddRow(expectedID))

		id, err := repo.GetOrCreateTaskTypeID(ctx, typeName)

		require.NoError(t, err)
		assert.Equal(t, expectedID, id)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failure - db error on select", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock, repoMetrics)
		dbError := errors.New("DB error")

		mock.ExpectQuery("SELECT type_id FROM task_types WHERE type_name = \\$1").
			WithArgs("any type").
			WillReturnError(dbError)

		_, err = repo.GetOrCreateTaskTypeID(ctx, "any type")

		require.Error(t, err)
		require.ErrorIs(t, err, dbError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failure - insert error", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		typeName := "New Type"
		repo := repository.NewTaskRepository(mock, repoMetrics)

		// 1. We are expecting a SELECT that will return a "no rows" error.
		mock.ExpectQuery("SELECT type_id FROM task_types WHERE type_name = \\$1").
			WithArgs(typeName).
			WillReturnError(pgx.ErrNoRows)

		// 2. Waiting for INSERT to create a new type
		mock.ExpectExec("INSERT INTO task_types").
			WithArgs(typeName).
			WillReturnError(assert.AnError)

		_, err = repo.GetOrCreateTaskTypeID(ctx, typeName)

		require.Error(t, err)
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failure - 2nd select error", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		typeName := "New Type"
		repo := repository.NewTaskRepository(mock, repoMetrics)

		// 1. We are expecting a SELECT that will return a "no rows" error.
		mock.ExpectQuery("SELECT type_id FROM task_types WHERE type_name = \\$1").
			WithArgs(typeName).
			WillReturnError(pgx.ErrNoRows)

		// 2. Waiting for INSERT to create a new type
		mock.ExpectExec("INSERT INTO task_types").
			WithArgs(typeName).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		// 3. We are waiting for the second SELECT, which will now find the created ID
		mock.ExpectQuery("SELECT type_id FROM task_types WHERE type_name = \\$1").
			WithArgs(typeName).
			WillReturnError(assert.AnError)

		_, err = repo.GetOrCreateTaskTypeID(ctx, typeName)

		require.Error(t, err)
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestUpsertTask checks the logic of the insert or update task.
func TestUpsertTask(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	task := models.Task{ID: 101, Description: "Test Description"}
	typeID := 5

	t.Run("success - insert new task", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock, repoMetrics)

		// 2. Waiting for INSERT
		mock.ExpectExec("INSERT INTO tasks").
			WithArgs(task.ID, typeID, task.CreatedAt, task.ClosedAt, task.Description, task.Address, task.Comments, false).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err = repo.UpsertTask(ctx, mock, task, typeID)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failure - insert new task", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock, repoMetrics)

		mock.ExpectExec("INSERT INTO tasks").
			WithArgs(task.ID, typeID, task.CreatedAt, task.ClosedAt, task.Description, task.Address, task.Comments, false).
			WillReturnError(assert.AnError)

		err = repo.UpsertTask(ctx, mock, task, typeID)

		require.Error(t, err)
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestUpdateTaskExecutors checks for updates to task executors.
func TestUpdateTaskExecutors(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	taskID := 101
	executors := []string{"Executor1", "Executor2"}

	t.Run("success - update executors", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock, repoMetrics)

		// 1. Waiting for old artists to be removed
		mock.ExpectExec("DELETE FROM task_executors WHERE task_id = \\$1").
			WithArgs(taskID).
			WillReturnResult(pgxmock.NewResult("DELETE", 2))

		// 2. We are waiting for the inclusion of new artists in the cycle
		mock.ExpectExec("INSERT INTO task_executors").
			WithArgs(taskID, executors[0]).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec("INSERT INTO task_executors").
			WithArgs(taskID, executors[1]).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err = repo.UpdateTaskExecutors(ctx, mock, taskID, executors)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failure - on insert", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock, repoMetrics)

		mock.ExpectExec("DELETE FROM task_executors WHERE task_id = \\$1").
			WithArgs(taskID).
			WillReturnResult(pgxmock.NewResult("DELETE", 2))

		mock.ExpectExec("INSERT INTO task_executors").
			WithArgs(taskID, executors[0]).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec("INSERT INTO task_executors").
			WithArgs(taskID, executors[1]).
			WillReturnError(assert.AnError)

		err = repo.UpdateTaskExecutors(ctx, mock, taskID, executors)

		require.Error(t, err)
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failure - on delete", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock, repoMetrics)
		dbError := errors.New("failed to delete")

		mock.ExpectExec("DELETE FROM task_executors").
			WithArgs(taskID).
			WillReturnError(dbError)

		err = repo.UpdateTaskExecutors(ctx, mock, taskID, executors)

		require.Error(t, err)
		require.ErrorIs(t, err, dbError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUpdateTaskCustomers(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	taskID := 101
	customers := []*olympus.Customer{
		{
			Id:    1,
			Name:  "John Doe",
			Login: "johnd",
		},
		{
			Id:    0,
			Name:  "New John",
			Login: "n/a",
		},
	}

	t.Run("success - update customers", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock, repoMetrics)

		// 1. Waiting for customers will added
		mock.ExpectQuery("INSERT INTO customers").
			WithArgs(customers[0].GetId(), customers[0].GetName(), customers[0].GetLogin()).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(0))
		mock.ExpectQuery("SELECT id FROM customers WHERE name = \\$1 AND external_id IS NULL").
			WithArgs(customers[1].GetName()).
			WillReturnError(pgx.ErrNoRows)
		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO customers (name, login) VALUES ($1, $2) RETURNING id")).
			WithArgs(customers[1].GetName(), customers[1].GetLogin()).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(1))

		// 2. Waiting for delete all chains with task
		mock.ExpectExec("DELETE FROM task_customers WHERE task_id = \\$1").
			WithArgs(taskID).
			WillReturnResult(pgxmock.NewResult("DELETE", 2))

		// 2. We are waiting for the inclusion of new customer in the cycle
		mock.ExpectCopyFrom(pgx.Identifier{"task_customers"}, []string{"task_id", "customer_id"}).WillReturnResult(1)

		err = repo.UpdateTaskCustomers(ctx, mock, taskID, customers)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failure - on update customer", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock, repoMetrics)

		mock.ExpectQuery("INSERT INTO customers").
			WithArgs(customers[0].GetId(), customers[0].GetName(), customers[0].GetLogin()).
			WillReturnError(assert.AnError)

		err = repo.UpdateTaskCustomers(ctx, mock, taskID, customers)

		require.Error(t, err)
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failure - on add customer", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock, repoMetrics)

		mock.ExpectQuery("INSERT INTO customers").
			WithArgs(customers[0].GetId(), customers[0].GetName(), customers[0].GetLogin()).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(0))
		mock.ExpectQuery("SELECT id FROM customers WHERE name = \\$1 AND external_id IS NULL").
			WithArgs(customers[1].GetName()).
			WillReturnError(assert.AnError)

		err = repo.UpdateTaskCustomers(ctx, mock, taskID, customers)

		require.Error(t, err)
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failure - on delete", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock, repoMetrics)

		// 1. Waiting for customers will added
		mock.ExpectQuery("INSERT INTO customers").
			WithArgs(customers[0].GetId(), customers[0].GetName(), customers[0].GetLogin()).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(0))
		mock.ExpectQuery("SELECT id FROM customers WHERE name = \\$1 AND external_id IS NULL").
			WithArgs(customers[1].GetName()).
			WillReturnError(pgx.ErrNoRows)
		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO customers (name, login) VALUES ($1, $2) RETURNING id")).
			WithArgs(customers[1].GetName(), customers[1].GetLogin()).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(1))

		// 2. Waiting for delete all chains with task
		mock.ExpectExec("DELETE FROM task_customers WHERE task_id = \\$1").
			WithArgs(taskID).
			WillReturnError(assert.AnError)

		err = repo.UpdateTaskCustomers(ctx, mock, taskID, customers)

		require.Error(t, err)
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("dailure - on copy from", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock, repoMetrics)

		// 1. Waiting for customers will added
		mock.ExpectQuery("INSERT INTO customers").
			WithArgs(customers[0].GetId(), customers[0].GetName(), customers[0].GetLogin()).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(0))
		mock.ExpectQuery("SELECT id FROM customers WHERE name = \\$1 AND external_id IS NULL").
			WithArgs(customers[1].GetName()).
			WillReturnError(pgx.ErrNoRows)
		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO customers (name, login) VALUES ($1, $2) RETURNING id")).
			WithArgs(customers[1].GetName(), customers[1].GetLogin()).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(1))

		// 2. Waiting for delete all chains with task
		mock.ExpectExec("DELETE FROM task_customers WHERE task_id = \\$1").
			WithArgs(taskID).
			WillReturnResult(pgxmock.NewResult("DELETE", 2))

		// 2. We are waiting for the inclusion of new customer in the cycle
		mock.ExpectCopyFrom(pgx.Identifier{"task_customers"}, []string{"task_id", "customer_id"}).
			WillReturnError(assert.AnError)

		err = repo.UpdateTaskCustomers(ctx, mock, taskID, customers)

		require.Error(t, err)
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestSaveTaskData checks the overall task save logic
// This test checks the correct orchestration of other method calls.
func TestSaveTaskData(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	task := models.Task{
		ID:        101,
		Type:      "NewType",
		Executors: []string{"Executor1"},
		Customers: []*olympus.Customer{
			{
				Id:    1,
				Name:  "John Doe",
				Login: "johnd",
			},
			{
				Id:    0,
				Name:  "New John",
				Login: "n/a",
			},
		},
	}
	typeID := 10

	t.Run("success - full flow", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock, repoMetrics)

		// Expect to Begin transaction
		mock.ExpectBegin()

		// Waiting for GetOrCreateTaskTypeID
		mock.ExpectQuery("SELECT type_id").WithArgs(task.Type).WillReturnError(pgx.ErrNoRows)
		mock.ExpectExec("INSERT INTO task_types").WithArgs(task.Type).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectQuery("SELECT type_id").
			WithArgs(task.Type).
			WillReturnRows(pgxmock.NewRows([]string{"type_id"}).AddRow(typeID))

		// Waiting for UpsertTask (assuming it's a new task)
		mock.ExpectExec("INSERT INTO tasks").
			WithArgs(task.ID, typeID, task.CreatedAt, task.ClosedAt, task.Description, task.Address, task.Comments, false).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		// Waiting for UpdateTaskExecutors
		mock.ExpectExec("DELETE FROM task_executors").WithArgs(task.ID).WillReturnResult(pgxmock.NewResult("DELETE", 0))
		mock.ExpectExec("INSERT INTO task_executors").
			WithArgs(task.ID, task.Executors[0]).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		// Waiting for UpdateTaskCustomers
		mock.ExpectQuery("INSERT INTO customers").
			WithArgs(task.Customers[0].GetId(), task.Customers[0].GetName(), task.Customers[0].GetLogin()).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(0))
		mock.ExpectQuery("SELECT id FROM customers WHERE name = \\$1 AND external_id IS NULL").
			WithArgs(task.Customers[1].GetName()).
			WillReturnError(pgx.ErrNoRows)
		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO customers (name, login) VALUES ($1, $2) RETURNING id")).
			WithArgs(task.Customers[1].GetName(), task.Customers[1].GetLogin()).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectExec("DELETE FROM task_customers WHERE task_id = \\$1").
			WithArgs(task.ID).
			WillReturnResult(pgxmock.NewResult("DELETE", 2))
		mock.ExpectCopyFrom(pgx.Identifier{"task_customers"}, []string{"task_id", "customer_id"}).WillReturnResult(1)

		// expect commit
		mock.ExpectCommit()

		err = repo.SaveTaskData(ctx, task)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failure - on Begin", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock, repoMetrics)

		// We simulate the error on the very first step
		mock.ExpectBegin().WillReturnError(assert.AnError)

		err = repo.SaveTaskData(ctx, task)

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to begin transaction")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failure - on GetOrCreateTaskTypeID", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock, repoMetrics)
		dbError := errors.New("type select failed")

		// We simulate the error on the very first step
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT type_id").WithArgs(task.Type).WillReturnError(dbError)
		mock.ExpectRollback()

		err = repo.SaveTaskData(ctx, task)

		require.Error(t, err)
		require.ErrorContains(t, err, "task type preparation error")
		require.ErrorIs(t, err, dbError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failure - on UpsertTask", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT type_id").WithArgs(task.Type).WillReturnError(pgx.ErrNoRows)
		mock.ExpectExec("INSERT INTO task_types").WithArgs(task.Type).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectQuery("SELECT type_id").
			WithArgs(task.Type).
			WillReturnRows(pgxmock.NewRows([]string{"type_id"}).AddRow(typeID))
		mock.ExpectExec("INSERT INTO tasks").
			WithArgs(task.ID, typeID, task.CreatedAt, task.ClosedAt, task.Description, task.Address, task.Comments, false).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		repo := repository.NewTaskRepository(mock, repoMetrics)
		err = repo.SaveTaskData(ctx, task)

		require.Error(t, err)
		require.ErrorContains(t, err, "task insert/update error")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("failure - on UpdateTaskExecutors", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT type_id").WithArgs(task.Type).WillReturnError(pgx.ErrNoRows)
		mock.ExpectExec("INSERT INTO task_types").WithArgs(task.Type).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectQuery("SELECT type_id").
			WithArgs(task.Type).
			WillReturnRows(pgxmock.NewRows([]string{"type_id"}).AddRow(typeID))

		mock.ExpectExec("INSERT INTO tasks").
			WithArgs(task.ID, typeID, task.CreatedAt, task.ClosedAt, task.Description, task.Address, task.Comments, false).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		mock.ExpectExec("DELETE FROM task_executors").WithArgs(task.ID).WillReturnError(assert.AnError)
		mock.ExpectRollback()

		repo := repository.NewTaskRepository(mock, repoMetrics)
		err = repo.SaveTaskData(ctx, task)

		require.Error(t, err)
		require.ErrorContains(t, err, "error updating executors")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failure - on UpdateTaskCustomers", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		// Expect to Begin transaction
		mock.ExpectBegin()

		// Waiting for GetOrCreateTaskTypeID
		mock.ExpectQuery("SELECT type_id").WithArgs(task.Type).WillReturnError(pgx.ErrNoRows)
		mock.ExpectExec("INSERT INTO task_types").WithArgs(task.Type).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectQuery("SELECT type_id").
			WithArgs(task.Type).
			WillReturnRows(pgxmock.NewRows([]string{"type_id"}).AddRow(typeID))

		// Waiting for UpsertTask (assuming it's a new task)
		mock.ExpectExec("INSERT INTO tasks").
			WithArgs(task.ID, typeID, task.CreatedAt, task.ClosedAt, task.Description, task.Address, task.Comments, false).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		// Waiting for UpdateTaskExecutors
		mock.ExpectExec("DELETE FROM task_executors").WithArgs(task.ID).WillReturnResult(pgxmock.NewResult("DELETE", 0))
		mock.ExpectExec("INSERT INTO task_executors").
			WithArgs(task.ID, task.Executors[0]).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		// Waiting for UpdateTaskCustomers
		mock.ExpectQuery("INSERT INTO customers").
			WithArgs(task.Customers[0].GetId(), task.Customers[0].GetName(), task.Customers[0].GetLogin()).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		repo := repository.NewTaskRepository(mock, repoMetrics)
		err = repo.SaveTaskData(ctx, task)

		require.Error(t, err)
		require.ErrorContains(t, err, "error updating customers")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failure - on Commit", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		// Expect to Begin transaction
		mock.ExpectBegin()

		// Waiting for GetOrCreateTaskTypeID
		mock.ExpectQuery("SELECT type_id").WithArgs(task.Type).WillReturnError(pgx.ErrNoRows)
		mock.ExpectExec("INSERT INTO task_types").WithArgs(task.Type).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectQuery("SELECT type_id").
			WithArgs(task.Type).
			WillReturnRows(pgxmock.NewRows([]string{"type_id"}).AddRow(typeID))

		// Waiting for UpsertTask (assuming it's a new task)
		mock.ExpectExec("INSERT INTO tasks").
			WithArgs(task.ID, typeID, task.CreatedAt, task.ClosedAt, task.Description, task.Address, task.Comments, false).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		// Waiting for UpdateTaskExecutors
		mock.ExpectExec("DELETE FROM task_executors").WithArgs(task.ID).WillReturnResult(pgxmock.NewResult("DELETE", 0))
		mock.ExpectExec("INSERT INTO task_executors").
			WithArgs(task.ID, task.Executors[0]).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		// Waiting for UpdateTaskCustomers
		mock.ExpectQuery("INSERT INTO customers").
			WithArgs(task.Customers[0].GetId(), task.Customers[0].GetName(), task.Customers[0].GetLogin()).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(0))
		mock.ExpectQuery("SELECT id FROM customers WHERE name = \\$1 AND external_id IS NULL").
			WithArgs(task.Customers[1].GetName()).
			WillReturnError(pgx.ErrNoRows)
		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO customers (name, login) VALUES ($1, $2) RETURNING id")).
			WithArgs(task.Customers[1].GetName(), task.Customers[1].GetLogin()).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectExec("DELETE FROM task_customers WHERE task_id = \\$1").
			WithArgs(task.ID).
			WillReturnResult(pgxmock.NewResult("DELETE", 2))
		mock.ExpectCopyFrom(pgx.Identifier{"task_customers"}, []string{"task_id", "customer_id"}).WillReturnResult(1)

		// expect commit
		mock.ExpectCommit().WillReturnError(assert.AnError)
		mock.ExpectRollback()

		repo := repository.NewTaskRepository(mock, repoMetrics)
		err = repo.SaveTaskData(ctx, task)

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to commit transaction")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
