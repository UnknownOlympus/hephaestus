package repository_test

import (
	"regexp"
	"testing"
	"time"

	"github.com/UnknownOlympus/hephaestus/internal/metrics"
	"github.com/UnknownOlympus/hephaestus/internal/repository"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var repoMetrics = metrics.NewMetrics(prometheus.DefaultRegisterer) //nolint:gochecknoglobals // for test case

func TestSaveLastProcessesDate_Success(t *testing.T) {
	t.Parallel()

	timeNow := time.Now()
	query := `
		INSERT INTO scraper_status (last_processed_date)
		VALUES ($1)
		ON CONFLICT (id) DO UPDATE SET last_processed_date = $1, updated_at = CURRENT_TIMESTAMP;`

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec(regexp.QuoteMeta(query)).WithArgs(timeNow).WillReturnResult(pgxmock.NewResult("INSERT", 1))

	repo := repository.NewStatusRepository(mock, repoMetrics)
	if err = repo.SaveProcessedDate(t.Context(), timeNow); err != nil {
		t.Errorf("error was not expected while inserting query: %v", err)
	}

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLastProcessedDate_QueryError(t *testing.T) {
	t.Parallel()

	timeNow := time.Now()
	query := `
		INSERT INTO scraper_status (last_processed_date)
		VALUES ($1)
		ON CONFLICT (id) DO UPDATE SET last_processed_date = $1, updated_at = CURRENT_TIMESTAMP;`

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec(regexp.QuoteMeta(query)).WithArgs(timeNow).WillReturnError(assert.AnError)

	repo := repository.NewStatusRepository(mock, repoMetrics)
	if err = repo.SaveProcessedDate(t.Context(), timeNow); err == nil {
		t.Errorf("error was expected, but received nil")
	}

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetLastProcessedDate_Success(t *testing.T) {
	t.Parallel()

	expectedTime := time.Now().AddDate(1, 3, 5)
	query := "SELECT last_processed_date FROM scraper_status ORDER BY updated_at DESC LIMIT 1"

	expectedRows := pgxmock.NewRows([]string{"last_processed_date"}).AddRow(expectedTime)

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnRows(expectedRows)

	repo := repository.NewStatusRepository(mock, repoMetrics)
	actualTime, err := repo.GetLastProcessedDate(t.Context())

	require.NoError(t, err)
	assert.Equal(t, expectedTime, actualTime)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetLastProcessedDate_QueryError(t *testing.T) {
	t.Parallel()

	query := "SELECT last_processed_date FROM scraper_status ORDER BY updated_at DESC LIMIT 1"

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnError(assert.AnError)

	repo := repository.NewStatusRepository(mock, repoMetrics)
	_, err = repo.GetLastProcessedDate(t.Context())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get last processed date from table last_processed_date")
	assert.Contains(t, err.Error(), assert.AnError.Error())
	require.NoError(t, mock.ExpectationsWereMet())
}
