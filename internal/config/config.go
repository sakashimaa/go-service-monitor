package config

import (
	"errors"
	"fmt"
	"net/url"
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
	MaxConns        int32         `yaml:"max_conns" envconfig:"POOL_MAX_CONNS"`
	MinConns        int32         `yaml:"min_conns" envconfig:"POOL_MIN_CONNS"`
	MaxConnLifetime time.Duration `yaml:"max_conn_lifetime" envconfig:"POOL_MAX_CONN_LIFETIME"`
	MaxConnIdleTime time.Duration `yaml:"max_conn_idle_time" envconfig:"POOL_MAX_CONN_IDLE_TIME"`
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

	applyPoolDefaults(&cfg.Pool)

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	return &cfg, nil
}

func validate(cfg *Config) error {
	if cfg.Port == 0 {
		return errors.New("port is required but not provided in config")
	}

	if cfg.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required but not provided in config")
	}

	if cfg.CheckInterval <= 0 {
		return errors.New("check_interval must be a positive duration")
	}

	if cfg.Timeout <= 0 {
		return errors.New("timeout must be a positive duration")
	}

	if cfg.Pool.MinConns > cfg.Pool.MaxConns {
		return errors.New("pool.min_conns must not exceed pool.max_conns")
	}

	for _, site := range cfg.Sites {
		if err := validateSiteURL(site); err != nil {
			return err
		}
	}

	return nil
}

func validateSiteURL(site Site) error {
	u, err := url.Parse(site.URL)
	if err != nil {
		return fmt.Errorf("site %s: invalid url %q: %w", site.Name, site.URL, err)
	}

	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("site %s: %q must be an absolute http/https url", site.Name, site.URL)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("site %s: unsupported url scheme %q (expected http or https)", site.Name, u.Scheme)
	}

	return nil
}

func applyPoolDefaults(p *PoolConfig) {
	if p.MaxConns <= 0 {
		p.MaxConns = 10
	}
	if p.MinConns < 0 {
		p.MinConns = 0
	}
}
