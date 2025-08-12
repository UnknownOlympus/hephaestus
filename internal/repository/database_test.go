package repository_test

import (
	"strings"
	"testing"
	"time"

	"github.com/UnknownOlympus/hephaestus/internal/repository"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestNewDatabase_Success(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode.")
	}

	var err error

	ctx := t.Context()
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpassword"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	defer func() {
		if err = pgContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate postgres container: %v", err)
		}
	}()

	host, err := pgContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get host: %v", err)
	}

	port, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get mapped port: %v", err)
	}

	dbpool, err := repository.NewDatabase(host, port.Port(), "testuser", "testpassword", "testdb")
	if err != nil {
		t.Fatalf("NewDatabase failed: %v", err)
	}
	defer dbpool.Close()

	if dbpool == nil {
		t.Fatalf("Expected non-nil dbpool, got nil")
	}

	if err = dbpool.Ping(ctx); err != nil {
		t.Fatalf("Failed to ping database after connection: %v", err)
	}
	t.Log("Successfully connected and pinged database")
}

func TestNewDatabase_ParseConfigError(t *testing.T) {
	t.Parallel()
	dbpool, err := repository.NewDatabase("localhost", "invalid-port", "user", "pass", "db")

	require.Error(t, err, "Expected an error for invalid database URL, but got nil")
	require.Nil(t, dbpool, "Expected nil dbpool, got: %v", dbpool)

	expectedErr := "failed to parse database config"
	require.ErrorContains(t, err, expectedErr)
	require.ErrorContainsf(t, err, "invalid port", "Expected error to mention 'invalid port', got: %v", err)
}

func TestNewDatabase_ConnectionError(t *testing.T) {
	t.Parallel()
	dbpool, err := repository.NewDatabase("nonexistent-host", "5432", "user", "pass", "db")

	require.Error(t, err, "Expected an error for connection failure, but got nil")
	if dbpool != nil {
		dbpool.Close()
		t.Errorf("Expected nil dbpool, got: %v", err)
	}

	expectedErr := "unable to create connection to PostgreSQL" // Error from NewWithConfig
	expectedErr2 := "failed to ping PostgreSQL DB"             // Error from Ping
	expectedErr3 := "no such host"                             // DNS error

	if !strings.Contains(err.Error(), expectedErr) &&
		!strings.Contains(err.Error(), expectedErr2) &&
		!strings.Contains(err.Error(), expectedErr3) {
		t.Errorf(
			"Expected error to contain '%s' or '%s' or '%s', got: %v",
			expectedErr,
			expectedErr2,
			expectedErr3,
			err,
		)
	}
}
