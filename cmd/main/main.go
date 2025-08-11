package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/UnknownOlympus/hephaestus/internal/client/hermes"
	"github.com/UnknownOlympus/hephaestus/internal/config"
	"github.com/UnknownOlympus/hephaestus/internal/metrics"
	"github.com/UnknownOlympus/hephaestus/internal/repository"
	"github.com/UnknownOlympus/hephaestus/internal/server"
	"github.com/UnknownOlympus/hephaestus/internal/services/employees"
	"github.com/UnknownOlympus/hephaestus/internal/services/tasks"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

const (
	envLocal = "local"
	envDev   = "development"
	envProd  = "production"
)

// main is the entry point of the application.
func main() {
	var err error
	var wgr sync.WaitGroup
	delta := 3
	serviceDealyInSeconds := 3

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	cfg := config.MustLoad()

	logger := setupLogger(cfg.Env)

	// Create a separate registry for metrics with exemplar
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	appMetrics := metrics.NewMetrics(reg)

	dtb, err := repository.NewDatabase(
		cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.Dbname)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	hermesClient, hermesConn, err := hermes.NewClient(cfg.HermesAddr)
	if err != nil {
		log.Fatalf("Failed to connect to Hermes service: %v", err)
	}
	defer stop()
	defer dtb.Close()

	employeeRepo := repository.NewEmployeeRepository(dtb, appMetrics)
	taskRepo := repository.NewTaskRepository(dtb, appMetrics)
	statRepo := repository.NewStatusRepository(dtb, appMetrics)
	staff := employees.NewStaff(logger, employeeRepo, appMetrics, hermesClient)
	taskService := tasks.NewTaskService(logger, taskRepo, statRepo, appMetrics, hermesClient)

	wgr.Add(delta)

	go func() {
		defer wgr.Done()
		serverPort := 8080
		server.StartMonitoringServer(ctx, logger, reg, dtb, serverPort, hermesConn)
	}()

	go func() {
		defer wgr.Done()
		logger.InfoContext(ctx, "Starting Employee Service")
		if err = staff.Start(ctx, cfg.Interval); err != nil {
			logger.ErrorContext(ctx, "Employee Service failed", "error", err)
		}
		logger.InfoContext(ctx, "Employee Service stopped.")
	}()

	time.Sleep(time.Duration(serviceDealyInSeconds) * time.Second)

	go func() {
		defer wgr.Done()
		logger.InfoContext(ctx, "Starting Task Service")
		if err = taskService.Start(ctx, cfg.Interval); err != nil {
			logger.ErrorContext(ctx, "Task Service failed", "error", err)
		}
		logger.InfoContext(ctx, "Task Service stopped.")
	}()

	logger.InfoContext(ctx, "Application started. Press Ctrl+C to stop.")

	wgr.Wait()

	logger.InfoContext(ctx, "Application stopped gracefully...")
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
