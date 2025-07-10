package tasks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Houeta/us-api-provider/internal/auth"
	"github.com/Houeta/us-api-provider/internal/client"
	"github.com/Houeta/us-api-provider/internal/metrics"
	"github.com/Houeta/us-api-provider/internal/parser"
	"github.com/Houeta/us-api-provider/internal/repository"
)

type TaskService struct {
	log        *slog.Logger
	repo       repository.TaskRepoIface
	statusRepo repository.StatusRepoIface
	parser     parser.TaskInterface
	metrics    *metrics.Metrics
}

func NewTaskService(log *slog.Logger,
	repo repository.TaskRepoIface,
	statusRepo repository.StatusRepoIface,
	metrics *metrics.Metrics,
) *TaskService {
	return &TaskService{log: log, repo: repo, statusRepo: statusRepo, metrics: metrics}
}

func (ts *TaskService) initLogger(opn string) *slog.Logger {
	return ts.log.With(
		slog.String("op", opn),
		slog.String("division", "task"),
	)
}

func (ts *TaskService) Start(
	ctx context.Context,
	loginURL, baseURL, username, password string,
	interval time.Duration,
) error {
	const opn = "Tasks.Start"
	log := ts.initLogger(opn)

	var err error

	httpClient := client.CreateHTTPClient(log)
	ts.parser = parser.NewTaskParser(httpClient, log, ts.metrics, baseURL)

	// 1. Login
	log.InfoContext(ctx, "Attempting login...")
	if err = auth.RetryLogin(ctx, log, httpClient, loginURL, baseURL, username, password); err != nil {
		ts.metrics.LoginAttempts.WithLabelValues("failure").Inc()
		return fmt.Errorf("failed to login: %w", err)
	}
	log.InfoContext(ctx, "Login successful.")
	ts.metrics.LoginAttempts.WithLabelValues("success").Inc()

	// 2. Update task types
	if err = ts.GetTaskTypes(ctx, httpClient, baseURL); err != nil {
		return fmt.Errorf("failed to get task types: %w", err)
	}

	// 3. Catch-up mode
	if err = ts.catchUpToNow(ctx, baseURL); err != nil {
		return fmt.Errorf("failed during catch-up process: %w", err)
	}

	// 4. Maintenance mode
	log.InfoContext(ctx, "Switching to maintenance mode.", "interval", interval.String())
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.InfoContext(ctx, "Periodic check triggered.")
			if err = ts.processDate(ctx, time.Now(), baseURL); err != nil {
				log.ErrorContext(ctx, "Periodic run failed", "error", err)
			}
		case <-ctx.Done():
			log.InfoContext(ctx, "Service shutting down.")
			return nil
		}
	}
}

func (ts *TaskService) catchUpToNow(ctx context.Context, baseURL string) error {
	const opn = "Tasks.catchUpToNow"
	log := ts.initLogger(opn)

	log.InfoContext(ctx, "Starting catch-up mode")

	for {
		lastDate, err := ts.GetLastDate(ctx)
		if err != nil {
			return fmt.Errorf("failed to get latest processed date: %w", err)
		}

		today := time.Now().UTC()
		lastDateUTC := lastDate.UTC()

		todayTruncated := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
		lastDateTruncated := time.Date(
			lastDateUTC.Year(),
			lastDateUTC.Month(),
			lastDateUTC.Day(),
			0,
			0,
			0,
			0,
			time.UTC,
		)

		if lastDateTruncated.After(todayTruncated) {
			log.InfoContext(
				ctx,
				"Catch-up complete. Last processed date is up-to-date.",
				"lastDate",
				lastDate.Format("2006-01-02"),
			)
			return nil
		}

		select {
		case <-ctx.Done():
			log.InfoContext(ctx, "Catch-up cancelled.")
			return fmt.Errorf("context initalize error: %w", ctx.Err())
		default:
		}

		if err = ts.processDate(ctx, lastDate, baseURL); err != nil {
			return fmt.Errorf("failed to process date %s during catch-up: %w", lastDate.Format("2006-01-02"), err)
		}
	}
}

func (ts *TaskService) processDate(ctx context.Context, dateToParse time.Time, baseURL string,
) error {
	const opn = "Tasks.processDate"
	log := ts.initLogger(opn)

	startTime := time.Now()
	defer func() {
		ts.metrics.RunDuration.WithLabelValues("task").Observe(time.Since(startTime).Seconds())
	}()

	normalizedDate := time.Date(
		dateToParse.Year(), dateToParse.Month(), dateToParse.Day(), 0, 0, 0, 0, time.UTC)

	log.DebugContext(ctx, "Scraping data", "date", normalizedDate.Format("02.01.2006"))

	tasks, err := ts.parser.ParseTasksByDate(ctx, normalizedDate)
	if err != nil {
		ts.metrics.Runs.WithLabelValues("failure").Inc()
		return fmt.Errorf(
			"failed to parse from '%s' for date '%s': %w",
			baseURL,
			normalizedDate.Format("2006-01-02"),
			err,
		)
	}

	if len(tasks) == 0 {
		log.DebugContext(ctx, "No tasks found for date", "date", normalizedDate.Format("2006-01-02"))
	}

	for _, task := range tasks {
		if err = ts.repo.SaveTaskData(ctx, task); err != nil {
			ts.metrics.Runs.WithLabelValues("failure").Inc()
			return fmt.Errorf("failed to save task '%d' in repository: %w", task.ID, err)
		}
	}

	nextDate := dateToParse.AddDate(0, 0, 1)
	if err = ts.statusRepo.SaveProcessedDate(ctx, nextDate); err != nil {
		ts.metrics.Runs.WithLabelValues("failure").Inc()
		return fmt.Errorf("failed to save next processed date '%s': %w", nextDate.Format("02.01.2006"), err)
	}

	log.InfoContext(ctx, "Successfully processed date", "date", dateToParse.Format("02.01.2006"))
	ts.metrics.Runs.WithLabelValues("success").Inc()
	ts.metrics.LastSuccessfulRun.WithLabelValues("task").SetToCurrentTime()

	return nil
}

func (ts *TaskService) GetLastDate(ctx context.Context) (time.Time, error) {
	lastDate, err := ts.statusRepo.GetLastProcessedDate(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			lastDate = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		} else {
			return time.Time{}, fmt.Errorf("failed to get latest processed date: %w", err)
		}
	}

	return lastDate, nil
}

func (ts *TaskService) GetTaskTypes(ctx context.Context, client *http.Client, destURL string) error {
	var err error

	taskNames, err := parser.ParseTaskTypes(ctx, client, destURL)
	if err != nil {
		return fmt.Errorf("failed to parse task types from '%s': %w", destURL, err)
	}

	for _, task := range taskNames {
		if _, err = ts.repo.GetOrCreateTaskTypeID(ctx, task); err != nil {
			return fmt.Errorf("failed to save task name '%s' in repository: %w", task, err)
		}
	}

	return nil
}
