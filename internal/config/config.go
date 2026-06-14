package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Sites         []Site        `yaml:"sites"`
	CheckInterval time.Duration `yaml:"check_interval"`
	Timeout       time.Duration `yaml:"timeout"`
	Server        `yaml:"server"`
}

type Server struct {
	Port            int           `yaml:"port"`
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

	// ручная проверка (для текущих масштабов нормально)
	if cfg.Port == 0 {
		return nil, errors.New("port is required but not provided in config")
	}

	return &cfg, nil
}
