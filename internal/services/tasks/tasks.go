package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Houeta/us-api-provider/internal/client"
	"github.com/Houeta/us-api-provider/internal/models"
	"github.com/Houeta/us-api-provider/internal/parser"
	"github.com/Houeta/us-api-provider/internal/repository"
)

type TaskService struct {
	log        *slog.Logger
	repo       repository.TaskRepoIface
	statusRepo repository.StatusRepoIface
}

func NewTaskService(log *slog.Logger,
	repo repository.TaskRepoIface,
	statusRepo repository.StatusRepoIface,
) *TaskService {
	return &TaskService{log: log, repo: repo, statusRepo: statusRepo}
}

func (ts *TaskService) initLogger(opn string) *slog.Logger {
	return ts.log.With(
		slog.String("op", opn),
		slog.String("division", "task"),
	)
}

func (ts *TaskService) Run(loginURL, baseURL, username, password string) ([]models.Task, error) {
	const opn = "Tasks.Run"

	var err error
	log := ts.initLogger(opn)
	ctxTimeout := 5
	httpClient := client.CreateHTTPClient(log)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(ctxTimeout*int(time.Second)))
	defer cancel()

	log.InfoContext(ctx, "Starting the service")
	log.InfoContext(ctx, "Attempting login", "url", baseURL)

	if err = ts.retryLogin(ctx, log, httpClient, loginURL, baseURL, username, password); err != nil {
		return nil, fmt.Errorf("failed to login by url '%s': %w", loginURL, err)
	}

	lastDate, err := ts.GetLastDate(ctx)
	if err != nil {
		log.ErrorContext(ctx, "Failed to get latest processed date", "error", err.Error())
		return nil, fmt.Errorf("failed to get latest processed date from repository: %w", err)
	}

	if err = ts.GetTaskTypes(ctx, httpClient, baseURL); err != nil {
		return nil, fmt.Errorf("failed to get task types on '%s': %w", baseURL, err)
	}

	if !lastDate.After(time.Now()) {
		log.DebugContext(ctx, "Scrapind data", "date", lastDate.Format("02.01.2006"))

		tasks, err := parser.ParseTasksByDate(ctx, httpClient, lastDate, baseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse from %s: %w", baseURL, err)
		}

		for _, task := range tasks {
			if err = ts.repo.SaveTaskData(ctx, task); err != nil {
				return nil, fmt.Errorf("failed to save task '%d' in repository: %w", task.ID, err)
			} else {
				log.DebugContext(ctx, "task processed successfully", "id", task.ID)
			}
		}

		// if err := ts.statusRepo.SaveProcessedDate(ctx, lastDate.AddDate(0, 0, 1)); err != nil {
		// 	return nil, fmt.Errorf("failed to save next processed date: %w", err)
		// }
		return tasks, nil
	}

	return nil, nil
}
