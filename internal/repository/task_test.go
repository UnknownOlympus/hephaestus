package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Houeta/us-api-provider/internal/models"
	"github.com/Houeta/us-api-provider/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetOrCreateTaskTypeID checks the logic for getting or creating a task type ID.
func TestGetOrCreateTaskTypeID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("success - type exists", func(t *testing.T) {
		t.Parallel()
		// Creating a mock for the database
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock)

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

		repo := repository.NewTaskRepository(mock)

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

		repo := repository.NewTaskRepository(mock)
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
		repo := repository.NewTaskRepository(mock)

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
		repo := repository.NewTaskRepository(mock)

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
	ctx := context.Background()
	task := models.Task{ID: 101, Description: "Test Description"}
	typeID := 5

	t.Run("success - insert new task", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock)

		// 2. Waiting for INSERT
		mock.ExpectExec("INSERT INTO tasks").
			WithArgs(task.ID, typeID, task.CreatedAt, task.ClosedAt, task.Description, task.Address, task.CustomerName, task.CustomerLogin, task.Comments, false).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err = repo.UpsertTask(ctx, task, typeID)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failure - insert new task", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock)

		mock.ExpectExec("INSERT INTO tasks").
			WithArgs(task.ID, typeID, task.CreatedAt, task.ClosedAt, task.Description, task.Address, task.CustomerName, task.CustomerLogin, task.Comments, false).
			WillReturnError(assert.AnError)

		err = repo.UpsertTask(ctx, task, typeID)

		require.Error(t, err)
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestUpdateTaskExecutors checks for updates to task executors.
func TestUpdateTaskExecutors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	taskID := 101
	executors := []string{"Executor1", "Executor2"}

	t.Run("success - update executors", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock)

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

		err = repo.UpdateTaskExecutors(ctx, taskID, executors)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failure - on insert", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock)

		mock.ExpectExec("DELETE FROM task_executors WHERE task_id = \\$1").
			WithArgs(taskID).
			WillReturnResult(pgxmock.NewResult("DELETE", 2))

		mock.ExpectExec("INSERT INTO task_executors").
			WithArgs(taskID, executors[0]).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec("INSERT INTO task_executors").
			WithArgs(taskID, executors[1]).
			WillReturnError(assert.AnError)

		err = repo.UpdateTaskExecutors(ctx, taskID, executors)

		require.Error(t, err)
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failure - on delete", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock)
		dbError := errors.New("failed to delete")

		mock.ExpectExec("DELETE FROM task_executors").
			WithArgs(taskID).
			WillReturnError(dbError)

		err = repo.UpdateTaskExecutors(ctx, taskID, executors)

		require.Error(t, err)
		require.ErrorIs(t, err, dbError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestSaveTaskData checks the overall task save logic
// This test checks the correct orchestration of other method calls.
func TestSaveTaskData(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	task := models.Task{
		ID:        101,
		Type:      "NewType",
		Executors: []string{"Executor1"},
	}
	typeID := 10

	t.Run("success - full flow", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock)

		// Waiting for GetOrCreateTaskTypeID
		mock.ExpectQuery("SELECT type_id").WithArgs(task.Type).WillReturnError(pgx.ErrNoRows)
		mock.ExpectExec("INSERT INTO task_types").WithArgs(task.Type).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectQuery("SELECT type_id").
			WithArgs(task.Type).
			WillReturnRows(pgxmock.NewRows([]string{"type_id"}).AddRow(typeID))

		// Waiting for UpsertTask (assuming it's a new task)
		mock.ExpectExec("INSERT INTO tasks").
			WithArgs(task.ID, typeID, task.CreatedAt, task.ClosedAt, task.Description, task.Address, task.CustomerName,
				task.CustomerLogin, task.Comments, false).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		// Waiting for UpdateTaskExecutors
		mock.ExpectExec("DELETE FROM task_executors").WithArgs(task.ID).WillReturnResult(pgxmock.NewResult("DELETE", 0))
		mock.ExpectExec("INSERT INTO task_executors").
			WithArgs(task.ID, task.Executors[0]).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err = repo.SaveTaskData(ctx, task)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failure - on GetOrCreateTaskTypeID", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewTaskRepository(mock)
		dbError := errors.New("type select failed")

		// We simulate the error on the very first step
		mock.ExpectQuery("SELECT type_id").WithArgs(task.Type).WillReturnError(dbError)

		err = repo.SaveTaskData(ctx, task)

		require.Error(t, err)
		require.ErrorContains(t, err, "task type preparation error")
		require.ErrorIs(t, err, dbError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failure - on UpdateTaskExecutors", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT type_id").WithArgs(task.Type).WillReturnError(pgx.ErrNoRows)
		mock.ExpectExec("INSERT INTO task_types").WithArgs(task.Type).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectQuery("SELECT type_id").
			WithArgs(task.Type).
			WillReturnRows(pgxmock.NewRows([]string{"type_id"}).AddRow(typeID))
		mock.ExpectExec("INSERT INTO tasks").
			WithArgs(task.ID, typeID, task.CreatedAt, task.ClosedAt, task.Description, task.Address, task.CustomerName,
				task.CustomerLogin, task.Comments, false).
			WillReturnError(assert.AnError)

		repo := repository.NewTaskRepository(mock)
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

		mock.ExpectQuery("SELECT type_id").WithArgs(task.Type).WillReturnError(pgx.ErrNoRows)
		mock.ExpectExec("INSERT INTO task_types").WithArgs(task.Type).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectQuery("SELECT type_id").
			WithArgs(task.Type).
			WillReturnRows(pgxmock.NewRows([]string{"type_id"}).AddRow(typeID))

		mock.ExpectExec("INSERT INTO tasks").
			WithArgs(task.ID, typeID, task.CreatedAt, task.ClosedAt, task.Description, task.Address, task.CustomerName,
				task.CustomerLogin, task.Comments, false).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		mock.ExpectExec("DELETE FROM task_executors").WithArgs(task.ID).WillReturnError(assert.AnError)

		repo := repository.NewTaskRepository(mock)
		err = repo.SaveTaskData(ctx, task)

		require.Error(t, err)
		require.ErrorContains(t, err, "error updating executors")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
