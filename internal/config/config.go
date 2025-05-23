package config

import (
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Env      string         `env-default:"local" yaml:"env"`                          // Env is the current environment: local, dev, prod.
	Postgres PostgresConfig `                    yaml:"postgres" env-required:"true"` // Postgres holds the database configuration
	Userside UsersideConfig `                    yaml:"userside" env-required:"true"` // Userisde golds the site parser configuration
}

// PostgresConfig struct holds the configuration details for connecting to a PostgreSQL database.
type PostgresConfig struct {
	Host     string `yaml:"host"`                        // Host is the database server address.
	Port     string `yaml:"port"     env-default:"5432"` // Port is the database server port.
	User     string `yaml:"user"`                        // User is the database user.
	Password string `yaml:"password"`                    // Password is the database user's password.
	Dbname   string `yaml:"db_name"`                     // Dbname is the name of the database.
}

// UsersideConfig struct holds the configuration details for connection to http/s website.
type UsersideConfig struct {
	BaseURL  string        `yaml:"url"`                         // Base URL is the url of Userside in format `https://example.com/`
	LoginURL string        `yaml:"login_url"`                   // Login URL is the url of Userside to login
	Username string        `yaml:"username"`                    // Username is the created user in Userside to login
	Password string        `yaml:"password"`                    // Password is the created password in Userside to login
	Interval time.Duration `yaml:"interval"  env-default:"12h"` // Interal is the time after that parser will update info.
}

// MustLoad loads the configuration from a YAML file and returns a Config struct.
func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		panic("config path is empty")
	}

	// check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + configPath)
	}

	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		panic("config error: " + err.Error())
	}

	defUserInterval := 12

	viper.SetDefault("postgres.port", "5432")
	viper.SetDefault("userside.interval", time.Duration(defUserInterval*int(time.Hour)))

	return &Config{
		Env: viper.GetString("env"),
		Postgres: PostgresConfig{
			Host:     viper.GetString("postgres.host"),
			Port:     viper.GetString("postgres.port"),
			User:     viper.GetString("postgres.user"),
			Password: viper.GetString("postgres.password"),
			Dbname:   viper.GetString("postgres.db_name"),
		},
		Userside: UsersideConfig{
			BaseURL:  viper.GetString("userside.url"),
			LoginURL: viper.GetString("userside.login_url"),
			Username: viper.GetString("userside.username"),
			Password: viper.GetString("userside.password"),
			Interval: viper.GetDuration("userside.interval"),
		},
	}
}
