package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Sites         []Site        `yaml:"sites"`
	CheckInterval time.Duration `yaml:"check_interval" envconfig:"CHECK_INTERVAL"`
	Timeout       time.Duration `yaml:"timeout" envconfig:"HTTP_TIMEOUT"`
	LogLevel      string        `yaml:"log_level" envconfig:"LOG_LEVEL" default:"info"`
	DatabaseURL   string        `envconfig:"DATABASE_URL"`
	Pool          PoolConfig    `yaml:"pool"`
	Server        `yaml:"server"`
}

type PoolConfig struct {
	MaxConns        int32         `yaml:"max_conns"`
	MinConns        int32         `yaml:"min_conns"`
	MaxConnLifetime time.Duration `yaml:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `yaml:"max_conn_idle_time"`
}

type Server struct {
	Port            int           `yaml:"port" envconfig:"APP_PORT"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

type Site struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}

	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to process env variables: %w", err)
	}

	// ручная проверка (для текущих масштабов нормально)
	if cfg.Port == 0 {
		return nil, errors.New("port is required but not provided in config")
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return &cfg, nil
}
