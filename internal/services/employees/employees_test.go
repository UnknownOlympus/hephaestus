package employees

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/Houeta/us-api-provider/internal/models"
	mocks "github.com/Houeta/us-api-provider/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestProcessEmployeeInternal(t *testing.T) {
	// --- Test Cases Setup ---
	employeeToUpdate := models.Employee{ID: 101, FullName: "John Doe", Position: "Developer", Email: "john.doe@example.com", Phone: "+1234567890"}
	existingEmployeeInDB := models.Employee{ID: 101, FullName: "Johnny Doe", Position: "Old Position", Email: "old@email.com", Phone: "+111"}
	employeeToSkip := models.Employee{ID: 102, FullName: "Jane Smith", Position: "Manager", Email: "jane.smith@example.com", Phone: "+0987654321"}
	employeeToSave := models.Employee{ID: 103, FullName: "Sam Brown", Position: "Designer", Email: "sam.brown@example.com", Phone: "+555444333"}
	parsedEmployees := []models.Employee{employeeToUpdate, employeeToSkip, employeeToSave}

	t.Run("success - update, skip, and save", func(t *testing.T) {
		// --- Mocks Setup ---
		repoMock := new(mocks.EmployeeRepoIface)
		parserMock := new(mocks.EmployeeParserIface)
		logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
		staffService := NewStaff(logger, repoMock)
		ctx := context.Background()

		// --- Mock Expectations ---
		parserMock.On("ParseEmployees", mock.Anything).Return(parsedEmployees, nil)

		repoMock.On("GetEmployeeByID", mock.Anything, employeeToUpdate.ID).Return(existingEmployeeInDB, nil)
		repoMock.On("UpdateEmployee", mock.Anything, employeeToUpdate.ID, employeeToUpdate.FullName, employeeToUpdate.ShortName,
			employeeToUpdate.Position, employeeToUpdate.Email, employeeToUpdate.Phone).Return(nil)

		repoMock.On("GetEmployeeByID", mock.Anything, employeeToSkip.ID).Return(employeeToSkip, nil)

		repoMock.On("GetEmployeeByID", mock.Anything, employeeToSave.ID).Return(models.Employee{}, sql.ErrNoRows)
		repoMock.On("SaveEmployee", mock.Anything, employeeToSave.ID, employeeToSave.FullName, employeeToSave.ShortName,
			employeeToSave.Position, employeeToSave.Email, employeeToSave.Phone).Return(nil)

		// --- Execution ---
		err := staffService.processEmployeeInternal(ctx, parserMock)

		// --- Assertions ---
		require.NoError(t, err)
		parserMock.AssertExpectations(t)
		repoMock.AssertExpectations(t)
	})

	t.Run("failure on parse", func(t *testing.T) {
		repoMock := new(mocks.EmployeeRepoIface)
		parserMock := new(mocks.EmployeeParserIface)
		logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
		staffService := NewStaff(logger, repoMock)
		ctx := context.Background()
		parseError := errors.New("failed to parse")

		parserMock.On("ParseEmployees", mock.Anything).Return(nil, parseError)

		err := staffService.processEmployeeInternal(ctx, parserMock)

		require.Error(t, err)
		assert.ErrorIs(t, err, parseError)
		repoMock.AssertNotCalled(t, "GetEmployeeByID", mock.Anything, mock.Anything)
	})

	t.Run("failure on update", func(t *testing.T) {
		repoMock := new(mocks.EmployeeRepoIface)
		parserMock := new(mocks.EmployeeParserIface)
		logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
		staffService := NewStaff(logger, repoMock)
		ctx := context.Background()
		updateError := errors.New("failed to update")

		parserMock.On("ParseEmployees", mock.Anything).Return(parsedEmployees, nil)
		repoMock.On("GetEmployeeByID", mock.Anything, employeeToUpdate.ID).Return(existingEmployeeInDB, nil)
		repoMock.On("UpdateEmployee", mock.Anything, employeeToUpdate.ID, employeeToUpdate.FullName, employeeToUpdate.ShortName,
			employeeToUpdate.Position, employeeToUpdate.Email, employeeToUpdate.Phone).Return(updateError)

		err := staffService.processEmployeeInternal(ctx, parserMock)

		require.Error(t, err)
		assert.ErrorIs(t, err, updateError)
		parserMock.AssertExpectations(t)
		repoMock.AssertExpectations(t)
		repoMock.AssertNotCalled(t, "SaveEmployee")
	})

	t.Run("failure on save", func(t *testing.T) {
		repoMock := new(mocks.EmployeeRepoIface)
		parserMock := new(mocks.EmployeeParserIface)
		logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
		staffService := NewStaff(logger, repoMock)
		ctx := context.Background()
		saveError := errors.New("failed to save")

		parserMock.On("ParseEmployees", mock.Anything).Return(parsedEmployees, nil)
		repoMock.On("GetEmployeeByID", mock.Anything, employeeToUpdate.ID).Return(existingEmployeeInDB, nil)
		repoMock.On("UpdateEmployee", mock.Anything, employeeToUpdate.ID, employeeToUpdate.FullName, employeeToUpdate.ShortName,
			employeeToUpdate.Position, employeeToUpdate.Email, employeeToUpdate.Phone).Return(nil)
		repoMock.On("GetEmployeeByID", mock.Anything, employeeToSkip.ID).Return(employeeToSkip, nil)
		repoMock.On("GetEmployeeByID", mock.Anything, employeeToSave.ID).Return(models.Employee{}, sql.ErrNoRows)
		repoMock.On("SaveEmployee", mock.Anything, employeeToSave.ID, employeeToSave.FullName, employeeToSave.ShortName,
			employeeToSave.Position, employeeToSave.Email, employeeToSave.Phone).Return(saveError)

		err := staffService.processEmployeeInternal(ctx, parserMock)

		require.Error(t, err)
		assert.ErrorIs(t, err, saveError)
		parserMock.AssertExpectations(t)
		repoMock.AssertExpectations(t)
	})
}

func TestFixInvalidEmail(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	ctx := context.Background()

	t.Run("all emails are valid", func(t *testing.T) {
		employeesList := []models.Employee{
			{FullName: "Valid User", Email: "valid@example.com"},
		}
		fixed := fixInvalidEmail(ctx, logger, employeesList)
		assert.Equal(t, "valid@example.com", fixed[0].Email)
	})

	t.Run("email is empty", func(t *testing.T) {
		employeesList := []models.Employee{
			{FullName: "Empty Email User", Email: ""},
		}
		fixed := fixInvalidEmail(ctx, logger, employeesList)
		assert.NotEmpty(t, fixed[0].Email)
		isEmail, _ := ValidateEmployee(fixed[0].Email, "")
		assert.True(t, isEmail)
	})

	t.Run("invalid email", func(t *testing.T) {
		employeeList := []models.Employee{
			{FullName: "Invalid User", Email: "invalid.email@com"},
		}
		fixed := fixInvalidEmail(ctx, logger, employeeList)
		assert.NotEmpty(t, fixed[0].Email)
		isEmail, _ := ValidateEmployee(fixed[0].Email, "")
		assert.True(t, isEmail)
	})
}

func TestNewStaff(t *testing.T) {
	t.Parallel()
	logger := slog.Default()
	mockRepo := new(mocks.EmployeeRepoIface)
	s := NewStaff(logger, mockRepo)
	assert.NotNil(t, s)
}

func TestIsEmployeeExists(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("exists", func(t *testing.T) {
		expectedEmployee := models.Employee{ID: 123, FullName: "testuser"}
		mockRepo := new(mocks.EmployeeRepoIface)
		mockRepo.On("GetEmployeeByID", ctx, 123).Return(expectedEmployee, nil)
		ok, employee := IsEmployeeExists(ctx, 123, mockRepo)
		assert.True(t, ok)
		assert.Equal(t, expectedEmployee, employee)
		mockRepo.AssertExpectations(t)
	})

	t.Run("does not exist", func(t *testing.T) {
		mockRepo := new(mocks.EmployeeRepoIface)
		mockRepo.On("GetEmployeeByID", ctx, 123).Return(models.Employee{}, sql.ErrNoRows)
		ok, employee := IsEmployeeExists(ctx, 123, mockRepo)
		assert.False(t, ok)
		assert.Equal(t, models.Employee{}, employee)
		mockRepo.AssertExpectations(t)
	})

	t.Run("sql error", func(t *testing.T) {
		mockRepo := new(mocks.EmployeeRepoIface)
		mockRepo.On("GetEmployeeByID", ctx, 123).Return(models.Employee{}, sql.ErrConnDone)
		ok, employee := IsEmployeeExists(ctx, 123, mockRepo)
		assert.False(t, ok)
		assert.Equal(t, models.Employee{}, employee)
		mockRepo.AssertExpectations(t)
	})
}

func TestValidateEmployee(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name        string
		email       string
		phone       string
		expectEmail bool
		expectPhone bool
	}{
		{"valid email and phone", "test@example.com", "+1234567890", true, true},
		{"invalid email, valid phone", "test.com", "123-456-7890", false, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isEmail, isPhone := ValidateEmployee(tc.email, tc.phone)
			assert.Equal(t, tc.expectEmail, isEmail)
			assert.Equal(t, tc.expectPhone, isPhone)
		})
	}
}
