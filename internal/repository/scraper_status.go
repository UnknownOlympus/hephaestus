package repository

import (
	"context"
	"fmt"
	"time"
)

// SaveLastProcessedDate saves last processed date.
func (r *Repository) SaveLastProcessedDate(ctx context.Context, date time.Time) error {
	query := `
		INSERT INTO scraper_status (last_processed_date)
		VALUES ($1)
		ON CONFLICT (id) DO UPDATE SET last_processed_date = $1, updated_at = CURRENT_TIMESTAMP;`

	_, err := r.db.Exec(ctx, query, date)
	if err != nil {
		return fmt.Errorf("failed to execute insert query: %w", err)
	}

	return nil
}

// GetLastProcessedDate returns last processed date.
func (r *Repository) GetLastProcessedDate(ctx context.Context) (time.Time, error) {
	query := "SELECT last_processed_date FROM scraper_status ORDER BY updated_at DESC LIMIT 1"

	var lastDate time.Time

	err := r.db.QueryRow(ctx, query).Scan(&lastDate)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get last processed date from table last_processed_date: %w", err)
	}

	return lastDate, nil
}
