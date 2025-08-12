package employees

import (
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/UnknownOlympus/hephaestus/internal/metrics"
	"github.com/UnknownOlympus/hephaestus/internal/models"
	mocks "github.com/UnknownOlympus/hephaestus/mock"
	pb "github.com/UnknownOlympus/olympus-protos/gen/go/scraper/olympus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestProcessEmployee(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	mockRepo := mocks.NewEmployeeRepoIface(t)
	mockHermes := mocks.NewScraperServiceClient(t)
	reg := prometheus.NewRegistry()
	testMetrics := metrics.NewMetrics(reg)
	staffService := NewStaff(logger, mockRepo, testMetrics, mockHermes)

	t.Run("should do nothing when hashes match", func(t *testing.T) {
		mockHermes.On("GetEmployees", mock.Anything, mock.Anything).Return(&pb.GetEmployeesResponse{
			NewHash:   "new_hash_123",
			Employees: []*pb.Employee{},
		}, nil).Once()

		err := staffService.ProcessEmployee(t.Context())

		require.NoError(t, err)
		mockRepo.AssertNotCalled(t, "GetEmployeeByID")
		mockHermes.AssertExpectations(t)
	})

	t.Run("should save a new employee", func(t *testing.T) {
		newEmployeePb := &pb.Employee{Id: 1, Fullname: "New Employee", Email: "new@example.com", Phone: "0961234567"}

		mockHermes.On("GetEmployees", mock.Anything, mock.Anything).Return(&pb.GetEmployeesResponse{
			NewHash:   "new_hash_456",
			Employees: []*pb.Employee{newEmployeePb},
		}, nil).Once()

		mockRepo.On("GetEmployeeByID", mock.Anything, 1).Return(models.Employee{}, sql.ErrNoRows).Once()

		mockRepo.On("SaveEmployee", mock.Anything, 1, "New Employee", "", "", "new@example.com", "0961234567").
			Return(nil).
			Once()

		err := staffService.ProcessEmployee(t.Context())

		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockHermes.AssertExpectations(t)
	})

	t.Run("should return error when failed to save employee", func(t *testing.T) {
		newEmployeePb := &pb.Employee{Id: 1, Fullname: "New Employee"}

		mockHermes.On("GetEmployees", mock.Anything, mock.Anything).Return(&pb.GetEmployeesResponse{
			NewHash:   "new_hash_456",
			Employees: []*pb.Employee{newEmployeePb},
		}, nil).Once()

		mockRepo.On("GetEmployeeByID", mock.Anything, 1).Return(models.Employee{}, sql.ErrNoRows).Once()

		mockRepo.On("SaveEmployee", mock.Anything, 1, "New Employee", "", "", mock.Anything, "").
			Return(assert.AnError).
			Once()

		err := staffService.ProcessEmployee(t.Context())

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to save new employee")
		mockRepo.AssertExpectations(t)
		mockHermes.AssertExpectations(t)
	})

	t.Run("should update an existing employee", func(t *testing.T) {
		updatedEmployeePb := &pb.Employee{Id: 2, Fullname: "Updated Name", Email: "updated@example.com"}
		existingEmployeeModel := models.Employee{ID: 2, FullName: "Old Name", Email: "old@example.com"}

		mockHermes.On("GetEmployees", mock.Anything, mock.Anything).Return(&pb.GetEmployeesResponse{
			NewHash:   "new_hash_789",
			Employees: []*pb.Employee{updatedEmployeePb},
		}, nil).Once()

		mockRepo.On("GetEmployeeByID", mock.Anything, 2).Return(existingEmployeeModel, nil).Once()

		mockRepo.On("UpdateEmployee", mock.Anything, 2, "Updated Name", "", "", "updated@example.com", "").
			Return(nil).
			Once()

		err := staffService.ProcessEmployee(t.Context())

		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockHermes.AssertExpectations(t)
	})

	t.Run("should return error when failed to update employee", func(t *testing.T) {
		updatedEmployeePb := &pb.Employee{Id: 2, Fullname: "Updated Name", Email: "12345"}
		existingEmployeeModel := models.Employee{ID: 2, FullName: "Old Name", Email: "old@example.com"}

		mockHermes.On("GetEmployees", mock.Anything, mock.Anything).Return(&pb.GetEmployeesResponse{
			NewHash:   "new_hash_789",
			Employees: []*pb.Employee{updatedEmployeePb},
		}, nil).Once()

		mockRepo.On("GetEmployeeByID", mock.Anything, 2).Return(existingEmployeeModel, nil).Once()

		mockRepo.On("UpdateEmployee", mock.Anything, 2, "Updated Name", "", "", mock.Anything, "").
			Return(assert.AnError).
			Once()

		err := staffService.ProcessEmployee(t.Context())

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to update employee")
		mockRepo.AssertExpectations(t)
		mockHermes.AssertExpectations(t)
	})

	t.Run("should skip an identical existing employee", func(t *testing.T) {
		identicalEmployeePb := &pb.Employee{Id: 3, Fullname: "Same Name", Email: "same@example.com"}
		identicalEmployeeModel := models.Employee{ID: 3, FullName: "Same Name", Email: "same@example.com"}

		mockHermes.On("GetEmployees", mock.Anything, mock.Anything).Return(&pb.GetEmployeesResponse{
			NewHash:   "new_hash_abc",
			Employees: []*pb.Employee{identicalEmployeePb},
		}, nil).Once()
		mockRepo.On("GetEmployeeByID", mock.Anything, 3).Return(identicalEmployeeModel, nil).Once()

		err := staffService.ProcessEmployee(t.Context())

		require.NoError(t, err)
		mockRepo.AssertNotCalled(t, "SaveEmployee")
		mockRepo.AssertNotCalled(t, "UpdateEmployee")
		mockRepo.AssertExpectations(t)
		mockHermes.AssertExpectations(t)
	})

	t.Run("should return an error if hermes fails", func(t *testing.T) {
		mockHermes.On("GetEmployees", mock.Anything, mock.Anything).Return(
			(*pb.GetEmployeesResponse)(nil), errors.New("gRPC connection failed"),
		).Once()

		err := staffService.ProcessEmployee(t.Context())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get employees from Hermes")
		mockRepo.AssertNotCalled(t, "GetEmployeeByID")
	})
}
