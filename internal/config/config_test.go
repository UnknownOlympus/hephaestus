package config_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Flaque/filet"
	"github.com/Houeta/us-api-provider/internal/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMustLoad_EmptyPath(t *testing.T) {
	t.Parallel()

	assert.PanicsWithValue(t, "config path is empty", func() {
		config.MustLoad()
	})
}

func TestMustLoad_FileNotExist(t *testing.T) {
	t.Setenv("CONFIG_PATH", "./invalid/path")
	assert.PanicsWithValue(t, "config file does not exist: ./invalid/path", func() {
		config.MustLoad()
	})
}

func TestMustLoad_ReadError(t *testing.T) {
	tmpFile := filet.TmpFile(t, "", "::::bad_yaml")
	defer filet.CleanUp(t)

	t.Setenv("CONFIG_PATH", tmpFile.Name())

	viper.SetConfigFile(tmpFile.Name())
	err := viper.ReadInConfig()
	require.Error(t, err)

	assert.PanicsWithValue(t, fmt.Sprintf("config error: %v", err), func() {
		config.MustLoad()
	})
}

func TestMustLoad_Success(t *testing.T) {
	configContent := `
---
env: "local"
postgres:
  host: "localhost"
  user: "pgUser"
  password: "pgPassword"
  db_name: "pgDatabase"
userside:
  url: "http://example.test"
  login_url: "http://example.test/login"
  username: "testUser"
  password: "testPassword"
`
	filet.File(t, "conf.yaml", configContent)
	defer filet.CleanUp(t)

	t.Setenv("CONFIG_PATH", "conf.yaml")

	cfg := config.MustLoad()

	assert.Equal(t, "local", cfg.Env)
	assert.Equal(t, "localhost", cfg.Postgres.Host)
	assert.Equal(t, "5432", cfg.Postgres.Port)
	assert.Equal(t, "pgUser", cfg.Postgres.User)
	assert.Equal(t, "pgPassword", cfg.Postgres.Password)
	assert.Equal(t, "pgDatabase", cfg.Postgres.Dbname)
	assert.Equal(t, "http://example.test", cfg.Userside.BaseURL)
	assert.Equal(t, "http://example.test/login", cfg.Userside.LoginURL)
	assert.Equal(t, "testUser", cfg.Userside.Username)
	assert.Equal(t, "testPassword", cfg.Userside.Password)
	assert.Equal(t, 12*time.Hour, cfg.Userside.Interval)
}
