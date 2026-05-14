package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App      AppConfig      `yaml:"app"`
	Database DatabaseConfig `yaml:"database"`
	Runner   RunnerConfig   `yaml:"runner"`
}

type AppConfig struct {
	Port         string `yaml:"port"`
	LogLevel     string `yaml:"log_level"`
	MaxBodyBytes int64  `yaml:"max_body_bytes"`
}

type DatabaseConfig struct {
	Driver   string         `yaml:"driver"`
	SQLite   SQLiteConfig   `yaml:"sqlite"`
	Postgres PostgresConfig `yaml:"postgres"`
}

type SQLiteConfig struct {
	Path string `yaml:"path"`
}

type PostgresConfig struct {
	DSN string `yaml:"dsn"`
}

type RunnerConfig struct {
	TimeoutMs       int64    `yaml:"timeout_ms"`
	IgnoreJSONPaths []string `yaml:"ignore_json_paths"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.App.Port == "" {
		cfg.App.Port = "8080"
	}

	switch cfg.Database.Driver {
	case "sqlite", "postgres":
		return &cfg, nil
	default:
		return nil, errors.New("invalid database.driver, expected sqlite or postgres")
	}
}
