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
	"github.com/Houeta/us-api-provider/internal/parser"
)

func (ts *TaskService) retryLogin(
	ctx context.Context,
	log *slog.Logger,
	httpClient *http.Client,
	loginURL, baseURL, username, password string,
) error {
	var err error

	const retryTimeout = 5 * time.Second
	const retries = 3

	for i := 0; i < retries; i++ {
		err := auth.Login(ctx, httpClient, loginURL, baseURL, username, password)
		if err == nil {
			log.InfoContext(ctx, "Successfuly logged in")
			return nil
		}

		log.WarnContext(ctx, "Failed to login, retrying...", "attempt", i+1, "of", retries, "error", err.Error())
		time.Sleep(retryTimeout)
	}

	finalError := errors.New("failed to login after multiple retries")
	log.ErrorContext(ctx, finalError.Error(), "last_error", err)
	return finalError
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
