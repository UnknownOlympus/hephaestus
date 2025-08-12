package tasks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/UnknownOlympus/hephaestus/internal/metrics"
	"github.com/UnknownOlympus/hephaestus/internal/models"
	"github.com/UnknownOlympus/hephaestus/internal/repository"
	pb "github.com/UnknownOlympus/olympus-protos/gen/go/scraper/olympus"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type TaskService struct {
	log           *slog.Logger
	repo          repository.TaskRepoIface
	statusRepo    repository.StatusRepoIface
	hermesClient  pb.ScraperServiceClient
	metrics       *metrics.Metrics
	lastKnownHash string
}

func NewTaskService(log *slog.Logger,
	repo repository.TaskRepoIface,
	statusRepo repository.StatusRepoIface,
	metrics *metrics.Metrics,
	hermesClient pb.ScraperServiceClient,
) *TaskService {
	return &TaskService{log: log, repo: repo, statusRepo: statusRepo, metrics: metrics, hermesClient: hermesClient}
}

func (ts *TaskService) initLogger(opn string) *slog.Logger {
	return ts.log.With(
		slog.String("op", opn),
		slog.String("division", "task"),
	)
}

func (ts *TaskService) Start(ctx context.Context, interval time.Duration) error {
	const opn = "Tasks.Start"
	log := ts.initLogger(opn)

	var err error

	// 2. Update task types
	if err = ts.updateTaskTypes(ctx); err != nil {
		log.ErrorContext(ctx, "failed to update task types on startup", "error", err)
		return fmt.Errorf("failed to get task types: %w", err)
	}

	// 3. Catch-up mode
	if err = ts.catchUpToNow(ctx); err != nil {
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
			if err = ts.processDate(ctx, time.Now()); err != nil {
				log.ErrorContext(ctx, "Periodic run failed", "error", err)
			}
		case <-ctx.Done():
			log.InfoContext(ctx, "Service shutting down.")
			return nil
		}
	}
}

func (ts *TaskService) catchUpToNow(ctx context.Context) error {
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

		if err = ts.processDate(ctx, lastDate); err != nil {
			return fmt.Errorf("failed to process date %s during catch-up: %w", lastDate.Format("2006-01-02"), err)
		}
	}
}

func (ts *TaskService) processDate(ctx context.Context, dateToParse time.Time,
) error {
	const opn = "Tasks.processDate"
	log := ts.initLogger(opn)
	startTime := time.Now()

	normalizedDate := time.Date(
		dateToParse.Year(), dateToParse.Month(), dateToParse.Day(), 0, 0, 0, 0, time.UTC)

	dateKey := normalizedDate.Format("2006-01-02")
	log.DebugContext(ctx, "Scraping data", "date", dateKey)

	req := &pb.GetDailyTasksRequest{
		KnownHash: ts.lastKnownHash,
		Date:      wrapperspb.String(dateKey),
	}
	resp, err := ts.hermesClient.GetDailyTasks(ctx, req)
	if err != nil {
		ts.metrics.Runs.WithLabelValues("failure").Inc()
		return fmt.Errorf("failed to get tasks for date '%s' from Hermes: %w", dateKey, err)
	}

	if len(resp.GetTasks()) == 0 || ts.lastKnownHash == resp.GetNewHash() {
		log.DebugContext(ctx, "No new tasks found for date", "date", dateKey)
	} else {
		log.InfoContext(ctx, "New data received from Hermes", "date", dateKey, "count", len(resp.GetTasks()))
		tasks := convertPbTasksToModels(resp.GetTasks())
		for _, task := range tasks {
			if err = ts.repo.SaveTaskData(ctx, task); err != nil {
				ts.metrics.Runs.WithLabelValues("failure").Inc()
				return fmt.Errorf("failed to save task '%d': %w", task.ID, err)
			}
		}
	}

	ts.lastKnownHash = resp.GetNewHash()
	nextDate := dateToParse.AddDate(0, 0, 1)
	if err = ts.statusRepo.SaveProcessedDate(ctx, nextDate); err != nil {
		ts.metrics.Runs.WithLabelValues("failure").Inc()
		return fmt.Errorf("failed to save next processed date '%s': %w", nextDate.Format("02.01.2006"), err)
	}

	log.InfoContext(ctx, "Successfully processed date", "date", dateToParse.Format("02.01.2006"))
	ts.metrics.Runs.WithLabelValues("success").Inc()
	ts.metrics.LastSuccessfulRun.WithLabelValues("task").SetToCurrentTime()
	ts.metrics.RunDuration.WithLabelValues("task").Observe(time.Since(startTime).Seconds())
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

func (ts *TaskService) updateTaskTypes(ctx context.Context) error {
	resp, err := ts.hermesClient.GetTaskTypes(ctx, &pb.GetTaskTypesRequest{})
	if err != nil {
		return fmt.Errorf("failed to get task types from Hermes: %w", err)
	}

	for _, taskName := range resp.GetTypes() {
		if _, err = ts.repo.GetOrCreateTaskTypeID(ctx, taskName); err != nil {
			ts.log.ErrorContext(ctx, "failed to save task type", "name", taskName, "error", err)
			return fmt.Errorf("failed to save task name '%s' in repository: %w", taskName, err)
		}
	}

	return nil
}

func convertPbTasksToModels(pbTasks []*pb.Task) []models.Task {
	tasks := make([]models.Task, 0, len(pbTasks))
	for _, pbt := range pbTasks {
		task := models.Task{
			ID:            int(pbt.GetId()),
			Type:          pbt.GetType(),
			CreatedAt:     pbt.GetCreationDate().AsTime(),
			ClosedAt:      pbt.GetClosingDate().AsTime(),
			Description:   pbt.GetDescription(),
			Address:       pbt.GetAddress(),
			CustomerName:  pbt.GetCustomerName(),
			CustomerLogin: pbt.GetCustomerLogin(),
			Comments:      pbt.GetComments(),
		}
		tasks = append(tasks, task)
	}
	return tasks
}
