package config_test

import (
	"testing"
	"time"

	"github.com/UnknownOlympus/hephaestus/internal/config"

	"github.com/stretchr/testify/assert"
)

func Test_MustLoadFromFile(t *testing.T) {
	t.Setenv("HEPHAESTUS_ENV", "local")
	t.Setenv("DB_HOST", "testHost")
	t.Setenv("DB_PORT", "12345")
	t.Setenv("DB_USERNAME", "admin")
	t.Setenv("DB_PASSWORD", "adminpass")
	t.Setenv("DB_NAME", "testName")
	t.Setenv("HERMES_ADDRESS", "testAddr")

	cfg := config.MustLoad()

	assert.Equal(t, "local", cfg.Env)
	assert.Equal(t, "testHost", cfg.Postgres.Host)
	assert.Equal(t, "12345", cfg.Postgres.Port)
	assert.Equal(t, "admin", cfg.Postgres.User)
	assert.Equal(t, "adminpass", cfg.Postgres.Password)
	assert.Equal(t, "testName", cfg.Postgres.Dbname)
	assert.Equal(t, 10*time.Minute, cfg.Interval)
	assert.Equal(t, "testAddr", cfg.HermesAddr)
}

func TestMustLoad_IntervalError(t *testing.T) {
	t.Setenv("HEPHAESTUS_INTERVAL", "error_value")

	assert.PanicsWithValue(t, "failed to parse interval from configuration", func() {
		config.MustLoad()
	})
}
