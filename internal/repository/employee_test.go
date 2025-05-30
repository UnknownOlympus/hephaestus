package repository_test

import (
	"context"
	"regexp"
	"testing"

	"github.com/Houeta/us-api-provider/internal/models"
	"github.com/Houeta/us-api-provider/internal/repository"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const saveEmployeeQuery = `
	INSERT INTO employees (id, fullname, position, email, phone)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (id) DO NOTHING;
`

const updateEmployeeQuery = `
	UPDATE employees
	SET fullname = $2, position = $3, email = $4, phone = $5, updated_at = CURRENT_TIMESTAMP
	WHERE id = $1;
`
const getEmployeeByIDQuery = `SELECT id, fullname, position, email, phone FROM employees WHERE id=$1`

func TestSaveEmployee_QueryError(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	expectedID := 123
	expectedFullname := "Test User"
	expectedPosition := "qa"
	expectedEmail := "test@test.com"
	expectedPhone := "123456789"

	mock.ExpectExec(regexp.QuoteMeta(saveEmployeeQuery)).
		WithArgs(expectedID, expectedFullname, expectedPosition, expectedEmail, expectedPhone).
		WillReturnError(assert.AnError)

	repo := repository.NewEmployeeRepository(mock)
	err = repo.SaveEmployee(
		context.Background(), expectedID, expectedFullname, expectedPosition, expectedEmail, expectedPhone)
	if err == nil {
		t.Error("Error was expected, but got nil.")
	}

	assert.Equal(t, err.Error(), "failed to save employee: "+assert.AnError.Error())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveEmployee_Success(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	expectedID := 123
	expectedFullname := "Test User"
	expectedPosition := "qa"
	expectedEmail := "test@test.com"
	expectedPhone := "123456789"

	mock.ExpectExec(regexp.QuoteMeta(saveEmployeeQuery)).
		WithArgs(expectedID, expectedFullname, expectedPosition, expectedEmail, expectedPhone).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	repo := repository.NewEmployeeRepository(mock)
	err = repo.SaveEmployee(
		context.Background(), expectedID, expectedFullname, expectedPosition, expectedEmail, expectedPhone)
	if err != nil {
		t.Errorf("Nil was expected, but got error: %s", err.Error())
	}

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateEmployee_QueryError(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	expectedID := 123
	expectedFullname := "Test User"
	expectedPosition := "qa"
	expectedEmail := "test@test.com"
	expectedPhone := "123456789"

	mock.ExpectExec(regexp.QuoteMeta(updateEmployeeQuery)).
		WithArgs(expectedID, expectedFullname, expectedPosition, expectedEmail, expectedPhone).
		WillReturnError(assert.AnError)

	repo := repository.NewEmployeeRepository(mock)
	err = repo.UpdateEmployee(
		context.Background(), expectedID, expectedFullname, expectedPosition, expectedEmail, expectedPhone)
	if err == nil {
		t.Error("Error was expected, but got nil.")
	}

	assert.Equal(t, err.Error(), "failed to update employee data: "+assert.AnError.Error())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateEmployee_Success(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	expectedID := 123
	expectedFullname := "Test User"
	expectedPosition := "qa"
	expectedEmail := "test@test.com"
	expectedPhone := "123456789"

	mock.ExpectExec(regexp.QuoteMeta(updateEmployeeQuery)).
		WithArgs(expectedID, expectedFullname, expectedPosition, expectedEmail, expectedPhone).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	repo := repository.NewEmployeeRepository(mock)
	err = repo.UpdateEmployee(
		context.Background(), expectedID, expectedFullname, expectedPosition, expectedEmail, expectedPhone)
	if err != nil {
		t.Errorf("Nil was expected, but got error: %s", err.Error())
	}

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetEmployeeByID_QueryError(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	expectedEmployee := models.Employee{
		ID: 123,
	}

	mock.ExpectQuery(regexp.QuoteMeta(getEmployeeByIDQuery)).
		WithArgs(expectedEmployee.ID).
		WillReturnError(assert.AnError)

	repo := repository.NewEmployeeRepository(mock)
	actualEmpployee, err := repo.GetEmployeeByID(context.Background(), expectedEmployee.ID)

	require.Error(t, err)
	require.EqualError(t, err, "failed to get employee by id: "+assert.AnError.Error())
	assert.IsType(t, models.Employee{}, actualEmpployee)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetEmployeeByID_Success(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	expEmployee := models.Employee{
		ID:       123,
		FullName: "test user",
		Position: "qa",
		Email:    "test@test.com",
		Phone:    "123456789",
	}
	expectedRows := pgxmock.NewRows([]string{"id", "fullname", "position", "email", "phone"}).
		AddRow(expEmployee.ID, expEmployee.FullName, expEmployee.Position, expEmployee.Email, expEmployee.Phone)

	mock.ExpectQuery(regexp.QuoteMeta(getEmployeeByIDQuery)).
		WithArgs(expEmployee.ID).
		WillReturnRows(expectedRows)

	repo := repository.NewEmployeeRepository(mock)
	actualEmpployee, err := repo.GetEmployeeByID(context.Background(), expEmployee.ID)

	require.NoError(t, err)
	assert.IsType(t, models.Employee{}, actualEmpployee)
	assert.Equal(t, expEmployee, actualEmpployee)
	require.NoError(t, mock.ExpectationsWereMet())
}
