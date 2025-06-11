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

	"github.com/tamathecxder/randomail"

	"github.com/Houeta/us-api-provider/internal/auth"
	"github.com/Houeta/us-api-provider/internal/client"
	"github.com/Houeta/us-api-provider/internal/models"
	"github.com/Houeta/us-api-provider/internal/parser"
	"github.com/Houeta/us-api-provider/internal/repository"
)

type Staff struct {
	log    *slog.Logger
	repo   repository.EmployeeRepoIface
	parser parser.EmployeeParserIface
}

func NewStaff(log *slog.Logger, repo repository.EmployeeRepoIface) *Staff {
	return &Staff{log: log, repo: repo}
}

func (s *Staff) initLogger(opn string) *slog.Logger {
	return s.log.With(
		slog.String("op", opn),
		slog.String("division", "employee"),
	)
}

// Start executes the staff service logic by fetching employees, validating their email addresses,
// and either updating existing employees or saving new ones to the repository.
func (s *Staff) Start(ctx context.Context, loginURL, baseURL, username, password string, interval time.Duration) error {
	const opn = "Employee.Start"
	log := s.initLogger(opn)

	var err error

	httpClient := client.CreateHTTPClient(log)
	s.parser = parser.NewEmployeeParser(httpClient, baseURL)

	// 1. Login
	log.InfoContext(ctx, "Attempting login...")
	if err = auth.RetryLogin(ctx, log, httpClient, loginURL, baseURL, username, password); err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}
	log.InfoContext(ctx, "Login  successful.")

	// 2. Catch-up mode
	log.InfoContext(ctx, "Starting catch-up mode")
	if err = s.ProcessEmployee(ctx); err != nil {
		return fmt.Errorf("failed during catch-up process: %w", err)
	}

	// 3. Maintainance mode
	log.InfoContext(ctx, "Starting maintainance mode", "interval", interval.String())
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.InfoContext(ctx, "Periodic check triggered.")
			if err = s.ProcessEmployee(ctx); err != nil {
				log.ErrorContext(ctx, "Periodic run failed", "error", err)
			}
		case <-ctx.Done():
			log.InfoContext(ctx, "Service shutting down.")
			return nil
		}
	}
}

func (s *Staff) ProcessEmployee(ctx context.Context) error {
	return s.processEmployeeInternal(ctx, s.parser)
}

func (s *Staff) processEmployeeInternal(pctx context.Context, employeeParser parser.EmployeeParserIface) error {
	const opn = "Employee.ProcessEmployee"
	log := s.initLogger(opn)

	contextTimeout := 10
	ctx, cancel := context.WithTimeout(pctx, time.Duration(contextTimeout)*time.Second)
	defer cancel()

	employees, err := employeeParser.ParseEmployees(ctx)
	if err != nil {
		return fmt.Errorf("failed to parse employee from HTML: %w", err)
	}

	fixedEmployees := fixInvalidEmail(ctx, log, employees)

	for _, employee := range fixedEmployees {
		existed, existedEmployee := IsEmployeeExists(ctx, employee.ID, s.repo)
		if existed {
			if existedEmployee == employee {
				log.DebugContext(ctx, "employee is existed, skipped", "fullname", employee.FullName)
				continue
			}
			updateErr := s.repo.UpdateEmployee(ctx,
				employee.ID,
				employee.FullName,
				employee.ShortName,
				employee.Position,
				employee.Email,
				employee.Phone,
			)
			if updateErr != nil {
				return fmt.Errorf("failed to update employee: '%s': %w", employee.FullName, updateErr)
			}
		} else {
			saveErr := s.repo.SaveEmployee(ctx, employee.ID, employee.FullName, employee.ShortName,
				employee.Position, employee.Email, employee.Phone)
			if saveErr != nil {
				return fmt.Errorf("failed to save new employee %s: %w", employee.FullName, saveErr)
			}
		}
	}

	return nil
}

func fixInvalidEmail(ctx context.Context, log *slog.Logger, employees []models.Employee) []models.Employee {
	var invalidCounter int
	fixedEmployees := make([]models.Employee, 0, len(employees))

	for _, employee := range employees {
		if employee.Email == "" {
			log.DebugContext(ctx, "Email was not specified, generate random email", "employee", employee.FullName)
			employee.Email = randomail.GenerateRandomEmail()
			invalidCounter++
		}

		isEmail, _ := ValidateEmployee(employee.Email, employee.Phone)
		if !isEmail {
			log.InfoContext(ctx, "Employee has invalid email, it will be replaced with temporary random email.",
				"fullname", employee.FullName, "email", employee.Email,
			)
			employee.Email = randomail.GenerateRandomEmail()
			invalidCounter++
		}

		fixedEmployees = append(fixedEmployees, employee)
	}

	if invalidCounter != 0 {
		log.WarnContext(
			ctx, "Number of employees with no or invalid email addressess. For mode information, enable debug mode",
			"value", invalidCounter)
	}

	return fixedEmployees
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
