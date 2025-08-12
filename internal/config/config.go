package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Env        string         `json:"env"`            // Env is the current environment: local, dev, prod.
	Postgres   PostgresConfig `json:"postgres"`       // Postgres holds the database configuration
	Interval   time.Duration  `json:"interval"`       // Interal is the time after that parser will update info.
	HermesAddr string         `json:"hermes_address"` //
}

// PostgresConfig struct holds the configuration details for connecting to a PostgreSQL database.
type PostgresConfig struct {
	Host     string `json:"host"`     // Host is the database server address.
	Port     string `json:"port"`     // Port is the database server port.
	User     string `json:"user"`     // User is the database user.
	Password string `json:"password"` // Password is the database user's password.
	Dbname   string `json:"db_name"`  // Dbname is the name of the database.
}

// MustLoad loads the configuration from a YAML file and returns a Config struct.
func MustLoad() *Config {
	_ = godotenv.Load()

	interval, err := time.ParseDuration(setDeafultEnv("HEPHAESTUS_INTERVAL", "10m"))
	if err != nil {
		panic("failed to parse interval from configuration")
	}

	return &Config{
		Env: setDeafultEnv("HEPHAESTUS_ENV", "production"),
		Postgres: PostgresConfig{
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			User:     os.Getenv("DB_USERNAME"),
			Password: os.Getenv("DB_PASSWORD"),
			Dbname:   os.Getenv("DB_NAME"),
		},
		Interval:   interval,
		HermesAddr: os.Getenv("HERMES_ADDRESS"),
	}
}

func setDeafultEnv(key, override string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = override
	}

	return value
}
