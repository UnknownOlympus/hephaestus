package main

import (
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Houeta/us-api-provider/internal/config"
	"github.com/Houeta/us-api-provider/internal/repository"
	"github.com/Houeta/us-api-provider/internal/services/employees"
	"github.com/Houeta/us-api-provider/internal/services/tasks"
)

const (
	envLocal = "local"
	envDev   = "development"
	envProd  = "production"
)

// main is the entry point of the application.
func main() {
	var err error

	cfg := config.MustLoad()

	logger := setupLogger(cfg.Env)

	dtb, err := repository.NewDatabase(
		cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.Dbname)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer dtb.Close()

	employeeRepo := repository.NewEmployeeRepository(dtb)
	taskRepo := repository.NewTaskRepository(dtb)
	statRepo := repository.NewStatusRepository(dtb)

	staff := employees.NewStaff(logger, employeeRepo)
	if err = staff.Run(cfg.Userside.LoginURL, cfg.Userside.BaseURL, cfg.Userside.Username, cfg.Userside.Password); err != nil {
		logger.Error("Failed to run employee parser", "op", "main.main", "division", "employee", "error", err)
	}

	taskService := tasks.NewTaskService(logger, taskRepo, statRepo)
	if _, err = taskService.Run(cfg.Userside.LoginURL, cfg.Userside.BaseURL, cfg.Userside.Username, cfg.Userside.Password); err != nil {
		logger.Error("Failed to run task parser", "op", "main.main", "division", "employee", "error", err)
	}

	waitForShutdown(logger)
	logger.Info("Shutting down gracefully...")
}

// setupLogger initializes and returns a logger based on the environment provided.
func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level:     slog.LevelDebug,
				AddSource: false,
				ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
					return a
				},
			}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level:     slog.LevelInfo,
				AddSource: false,
				ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
					return a
				},
			}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level:     slog.LevelWarn,
				AddSource: false,
				ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
					if a.Key == slog.TimeKey {
						return slog.Attr{Key: "", Value: slog.Value{}}
					}
					return a
				},
			}),
		)
	default:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level:     slog.LevelError,
				AddSource: false,
				ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
					if a.Key == slog.TimeKey {
						return slog.Attr{Key: "", Value: slog.Value{}}
					}
					return a
				},
			}),
		)

		log.Error(
			"The env parameter was not specified, or was invalid. Logging will be minimal, by default." +
				" Please specify the value of `env`: local, development, production")
	}

	return log
}

// waitForShutdown blocks the program execution until a termination signal is received from the operating system.
func waitForShutdown(log *slog.Logger) {
	// Create a channel to receive OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Block until a signal is received
	<-stop
	log.Info("Received SIGTERM signal")
}
