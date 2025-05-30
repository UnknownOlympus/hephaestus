package repository

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Database interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// NewDatabase creates a new PostgreSQL database connection pool using the provided host, port, username, password, and database name.
func NewDatabase(host, port, username, password, dbName string) (*pgxpool.Pool, error) {
	var (
		ctxTimeout = 5 * time.Second
		idleTime   = 30 * time.Second
		hcPeriod   = 30 * time.Second
	)
	var err error

	dbHost := net.JoinHostPort(host, port)
	dbURL := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=disable",
		username,
		password,
		dbHost,
		dbName,
	)

	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	poolConfig.MinConns = 3
	poolConfig.MaxConnIdleTime = idleTime
	poolConfig.HealthCheckPeriod = hcPeriod

	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	dbpool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection to PostgreSQL: %w", err)
	}

	if err = dbpool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL DB: %w", err)
	}

	return dbpool, nil
}
