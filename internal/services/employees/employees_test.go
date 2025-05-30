package employees_test

import (
	"context"
	"database/sql"
	"log/slog"
	"testing"

	"github.com/Houeta/us-api-provider/internal/models"
	"github.com/Houeta/us-api-provider/internal/services/employees"
	mocks "github.com/Houeta/us-api-provider/mock"
	"github.com/stretchr/testify/assert"
)

func TestNewStaff(t *testing.T) {
	t.Parallel()

	logger := slog.Default()
	mockRepo := new(mocks.EmployeeRepoIface)

	s := employees.NewStaff(logger, mockRepo)

	assert.NotNil(t, s)
}

func TestIsEmployeeExists_ErrNoRows(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mock := new(mocks.EmployeeRepoIface)
	mock.On("GetEmployeeByID", ctx, 123).Return(models.Employee{}, sql.ErrNoRows)

	ok, employee := employees.IsEmployeeExists(ctx, 123, mock)

	assert.False(t, ok, "Expected false, but got true")
	assert.Equal(t, models.Employee{}, employee)
}

func TestIsEmployeeExists_Error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mock := new(mocks.EmployeeRepoIface)
	mock.On("GetEmployeeByID", ctx, 123).Return(models.Employee{}, assert.AnError)

	ok, employee := employees.IsEmployeeExists(ctx, 123, mock)

	assert.False(t, ok, "Expected false, but got true")
	assert.Equal(t, models.Employee{}, employee)
}

func TestIsEmployeeExists_Success(t *testing.T) {
	t.Parallel()

	expectedEmployee := models.Employee{
		ID:       123,
		FullName: "testuser",
		Position: "test",
	}
	ctx := context.Background()
	mock := new(mocks.EmployeeRepoIface)
	mock.On("GetEmployeeByID", ctx, 123).Return(expectedEmployee, nil)

	ok, employee := employees.IsEmployeeExists(ctx, 123, mock)

	assert.True(t, ok, "Expected true, but got false")
	assert.Equal(t, expectedEmployee, employee)
}

func TestValiidateEmployee_Success(t *testing.T) {
	t.Parallel()

	email := "testuser@example.com"
	phone := "+345678765432"

	isEmail, isPhone := employees.ValidateEmployee(email, phone)

	assert.True(t, isEmail)
	assert.True(t, isPhone)
}

func TestValiidateEmployee_Fail(t *testing.T) {
	t.Parallel()

	email := "testuser.com"
	phone := "+invalid_number"

	isEmail, isPhone := employees.ValidateEmployee(email, phone)

	assert.False(t, isEmail)
	assert.False(t, isPhone)
}
