package employees

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/Houeta/us-api-provider/internal/auth"
	"github.com/Houeta/us-api-provider/internal/client"
	"github.com/Houeta/us-api-provider/internal/models"
	"github.com/Houeta/us-api-provider/internal/parser"
	"github.com/Houeta/us-api-provider/internal/repository"
)

type Staff struct {
	log  *slog.Logger
	repo repository.EmployeeRepoIface
}

func NewStaff(log *slog.Logger, repo repository.EmployeeRepoIface) *Staff {
	return &Staff{log: log, repo: repo}
}

// Run executes the staff service logic by fetching employees, validating their email addresses,
// and either updating existing employees or saving new ones to the repository.
func (s *Staff) Run(loginURL, baseURL, username, password string) error {
	const opn = "Staff.Run"

	var missCounter int
	ctxTimeout := 5

	log := s.log.With(
		slog.String("op", opn),
		slog.String("division", "employee"),
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(ctxTimeout*int(time.Second)))
	defer cancel()

	log.InfoContext(ctx, "Starting the service")

	employees, err := s.GetEmployees(ctx, loginURL, baseURL, username, password)
	if err != nil {
		return fmt.Errorf("failed to get employees: %w", err)
	}

	for _, employee := range employees {
		if employee.Email == "" {
			log.InfoContext(ctx, "Email was not specified, skip", "employee", employee.FullName)
			missCounter++
			continue
		}

		isEmail, _ := ValidateEmployee(employee.Email, employee.Phone)
		if !isEmail {
			log.InfoContext(ctx, "Employee has invalid email, he will be skipped.",
				"fullname", employee.FullName, "email", employee.Email)
			continue
		}
		existed, existedEmployee := IsEmployeeExists(ctx, employee.ID, s.repo)
		if existed {
			if existedEmployee == employee {
				log.DebugContext(ctx, "employee is existed, skipping", "fullname", employee.FullName)
				continue
			}
			updateErr := s.repo.UpdateEmployee(
				ctx,
				employee.ID,
				employee.FullName,
				employee.Position,
				employee.Email,
				employee.Phone,
			)
			if updateErr != nil {
				return fmt.Errorf("failed to update employee %s: %w", employee.FullName, updateErr)
			}
		} else {
			saveErr := s.repo.SaveEmployee(ctx, employee.ID, employee.FullName, employee.Position, employee.Email, employee.Phone)
			if saveErr != nil {
				return fmt.Errorf("failed to save new employee %s: %w", employee.FullName, saveErr)
			}
		}
	}

	log.WarnContext(
		ctx,
		"Number of employees with no or invalid email addresses. For more information, enable debug mode.",
		"value",
		missCounter,
	)

	return nil
}

// GetEmployees retrieves a list of employees from the specified base URL using the provided login credentials.
func (s *Staff) GetEmployees(ctx context.Context, loginURL, baseURL, username, password string,
) ([]models.Employee, error) {
	const op = "Staff.GetEmployees"
	const retryTimeout = 5 * time.Second

	log := s.log.With(
		slog.String("op", op),
		slog.String("division", "employee"),
	)

	log.InfoContext(ctx, "Logging in", "url", baseURL)

	httpClient := client.CreateHTTPClient(s.log)
	err := auth.Login(ctx, httpClient, loginURL, baseURL, username, password)
	if err != nil {
		log.ErrorContext(ctx, "Failed to login", "error", err.Error())
		time.Sleep(retryTimeout)
	}

	employees, err := parser.ParseEmployees(ctx, httpClient, baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse from %s: %w", baseURL, err)
	}

	return employees, nil
}

// ValidateEmployee validates the email and phone number of an employee.
func ValidateEmployee(email, phone string) (bool, bool) {
	var isEmail bool
	var isPhone bool

	if isValidEmail(email) {
		isEmail = true
	}

	if isValidPhoneNumber(phone) {
		isPhone = true
	}

	return isEmail, isPhone
}

// IsEmployeeExists checks if an employee with the given ID exists in the repository.
func IsEmployeeExists(ctx context.Context, employeeID int, repo repository.EmployeeRepoIface) (bool, models.Employee) {
	employee, err := repo.GetEmployeeByID(ctx, employeeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, models.Employee{}
		}
		return false, models.Employee{}
	}

	return true, employee
}

// isValidEmail checks if the given email address is valid.
func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// isValidPhoneNumber checks if a phone number is valid according to the E.164 format.
func isValidPhoneNumber(phone string) bool {
	e164Regex := `^\+?[0-9]\d{1,14}$`
	re := regexp.MustCompile(e164Regex)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")

	return re.Find([]byte(phone)) != nil
}
