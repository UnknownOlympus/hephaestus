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

	"github.com/UnknownOlympus/hephaestus/internal/metrics"
	"github.com/UnknownOlympus/hephaestus/internal/models"
	"github.com/UnknownOlympus/hephaestus/internal/repository"
	pb "github.com/UnknownOlympus/olympus-protos/gen/go/scraper/olympus"
	"github.com/tamathecxder/randomail"
)

type Staff struct {
	log           *slog.Logger
	repo          repository.EmployeeRepoIface
	metrics       *metrics.Metrics
	hermesClient  pb.ScraperServiceClient
	lastKnownHash string
}

func NewStaff(
	log *slog.Logger,
	repo repository.EmployeeRepoIface,
	metrics *metrics.Metrics,
	hermesClient pb.ScraperServiceClient,
) *Staff {
	return &Staff{log: log, repo: repo, metrics: metrics, hermesClient: hermesClient}
}

func (s *Staff) initLogger(opn string) *slog.Logger {
	return s.log.With(
		slog.String("op", opn),
		slog.String("division", "employee"),
	)
}

// Start executes the staff service logic by fetching employees, validating their email addresses,
// and either updating existing employees or saving new ones to the repository.
func (s *Staff) Start(ctx context.Context, interval time.Duration) error {
	const opn = "Employee.Start"
	log := s.initLogger(opn)

	var err error

	// 1. Catch-up mode
	log.InfoContext(ctx, "Starting initial data synchronization")
	if err = s.ProcessEmployee(ctx); err != nil {
		log.ErrorContext(ctx, "Initial run failed", "error", err)
		return fmt.Errorf("failed during catch-up process: %w", err)
	}

	// 2. Maintainance mode
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

func (s *Staff) ProcessEmployee(pctx context.Context) error {
	const opn = "Employee.ProcessEmployee"
	log := s.initLogger(opn)
	startTime := time.Now()

	contextTimeout := 10
	ctx, cancel := context.WithTimeout(pctx, time.Duration(contextTimeout)*time.Second)
	defer cancel()

	resp, err := s.hermesClient.GetEmployees(ctx, &pb.GetEmployeesRequest{
		KnownHash: s.lastKnownHash,
	})
	if err != nil {
		s.metrics.Runs.WithLabelValues("failure").Inc()
		s.metrics.RunDuration.WithLabelValues("employee").Observe(float64(time.Since(startTime).Seconds()))
		return fmt.Errorf("failed to get employees from Hermes: %w", err)
	}

	if len(resp.GetEmployees()) == 0 {
		log.InfoContext(ctx, "No new employee data. Hashes match.", "hash", resp.GetNewHash())
		s.lastKnownHash = resp.GetNewHash()
		return nil
	}

	log.InfoContext(ctx, "New data received from Hermes. Processing...", "employee_count", len(resp.GetEmployees()))

	employees := convertPbToModels(resp.GetEmployees())
	fixedEmployees := fixInvalidEmail(ctx, log, employees, s.metrics)

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

	s.lastKnownHash = resp.GetNewHash()
	s.metrics.Runs.WithLabelValues("success").Inc()
	s.metrics.RunDuration.WithLabelValues("employee").Observe(float64(time.Since(startTime).Seconds()))
	s.metrics.LastSuccessfulRun.WithLabelValues("employee").SetToCurrentTime()

	log.InfoContext(ctx, "Successfully processed and saved employee data.", "new_hash", s.lastKnownHash)
	return nil
}

func convertPbToModels(pbEmployees []*pb.Employee) []models.Employee {
	employees := make([]models.Employee, 0, len(pbEmployees))
	for _, pbe := range pbEmployees {
		emp := models.Employee{
			ID:        int(pbe.GetId()),
			FullName:  pbe.GetFullname(),
			ShortName: pbe.GetShortname(),
			Position:  pbe.GetPosition(),
			Email:     pbe.GetEmail(),
			Phone:     pbe.GetPhone(),
		}
		employees = append(employees, emp)
	}
	return employees
}

func fixInvalidEmail(
	ctx context.Context,
	log *slog.Logger,
	employees []models.Employee,
	metrics *metrics.Metrics,
) []models.Employee {
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
		metrics.EmailsFixed.Add(float64(invalidCounter))
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
