package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeTestConfig(t *testing.T, content string) string {
	t.Helper() // говорит go что эта функция хелпер, при падении будет показываться номер строки вызывающего кода
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write yaml contents: %v", err)
	}

	return path
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		envVars     map[string]string
		wantErr     bool
		wantErrSub  string // подстрока которая обязана быть в тексте ошибки
	}{
		{
			name: "полный корректный конфиг",
			yamlContent: `
check_interval: 5s
timeout: 10s
sites:
  - name: "Google"
    url: "https://google.com"
server:
  port: 8080
  read_timeout: 5s
  write_timeout: 10s
  idle_timeout: 15s
  shutdown_timeout: 5s
pool:
  max_conns: 10
  min_conns: 2
`,
			envVars: map[string]string{
				"DATABASE_URL": "postgres://test:test@localhost:5432/test",
			},
			wantErr: false,
		},
		{
			name: "минимальный конфиг - только обязательные поля",
			yamlContent: `
check_interval: 5s
timeout: 10s
server:
  port: 8080
`,
			envVars: map[string]string{
				"DATABASE_URL": "postgres://test:test@localhost:5432/test",
			},
			wantErr: false,
		},
		{
			name: "нет port -> ошибка валидации",
			yamlContent: `
check_interval: 5s
sites:
  - name: "Google"
    url: "https://google.com"
`,
			envVars: map[string]string{
				"DATABASE_URL": "postgres://test:test@localhost:5432/test",
			},
			wantErr:    true,
			wantErrSub: "port is required",
		},
		{
			name: "нет DATABASE_URL -> ошибка валидации",
			yamlContent: `
check_interval: 5s
timeout: 10s
server:
  port: 8080
`,
			wantErr:    true,
			wantErrSub: "DATABASE_URL is required",
			envVars: map[string]string{
				"DATABASE_URL": "",
			},
		},
		{
			name:        "невалидный YAML",
			yamlContent: `sites: [broken: [`,
			envVars: map[string]string{
				"DATABASE_URL": "postgres://test:test@localhost:5432/test",
			},
			wantErr:    true,
			wantErrSub: "failed to parse yaml",
		},
		{
			name: "check_interval равен нулю (не указан)",
			yamlContent: `
timeout: 10s
server:
  port: 8080
`,
			envVars: map[string]string{
				"DATABASE_URL": "postgres://test:test@localhost:5432/test",
			},
			wantErr:    true,
			wantErrSub: "check_interval must be a positive duration",
		},
		{
			name: "timeout отрицательный",
			yamlContent: `
check_interval: 5s
timeout: -1s
server:
  port: 8080
`,
			envVars: map[string]string{
				"DATABASE_URL": "postgres://test:test@localhost:5432/test",
				"HTTP_TIMEOUT": "-1s",
			},
			wantErr:    true,
			wantErrSub: "timeout must be a positive duration",
		},
		{
			name: "url сайта без схемы (как 'Bad Protocol' в проде)",
			yamlContent: `
check_interval: 5s
timeout: 10s
sites:
  - name: "Bad Protocol"
    url: "jajahaha123123.qweqwe"
server:
  port: 8080
`,
			envVars: map[string]string{
				"DATABASE_URL": "postgres://test:test@localhost:5432/test",
			},
			wantErr:    true,
			wantErrSub: "must be an absolute http/https url",
		},
		{
			name: "url сайта с неподдерживаемой схемой",
			yamlContent: `
check_interval: 5s
timeout: 10s
sites:
  - name: "FTP Site"
    url: "ftp://example.com"
server:
  port: 8080
`,
			envVars: map[string]string{
				"DATABASE_URL": "postgres://test:test@localhost:5432/test",
			},
			wantErr:    true,
			wantErrSub: "unsupported url scheme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			path := writeTestConfig(t, tt.yamlContent)
			cfg, err := Load(path)

			if tt.wantErr {
				if err == nil {
					t.Fatal("awaited error, got nil")
				}

				if tt.wantErrSub != "" && !strings.Contains(err.Error(), tt.wantErrSub) {
					t.Errorf("error = %q, waited substring %q", err.Error(), tt.wantErrSub)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg == nil {
				t.Fatal("cfg = nil, even with no error")
			}
		})
	}
}

func Test_MissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exists.yaml")

	_, err := Load(path)
	if err == nil {
		t.Fatal("awaited error with not existent file, got nil")
	}

	if !strings.Contains(err.Error(), "failed to read config file") {
		t.Errorf("error = %v, awaited mention about reading file", err.Error())
	}
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	path := writeTestConfig(t, `
check_interval: 5s
timeout: 10s
server:
  port: 8080
`)

	t.Setenv("DATABASE_URL", "postgres://from-env@localhost:5432/db")
	t.Setenv("HTTP_TIMEOUT", "3s") // в файле timeout: 10s - ожидаем, что победит env

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	if cfg.Timeout != 3*time.Second {
		t.Errorf("Timeout = %v, want 3s (env должен переопределить файл)", cfg.Timeout)
	}
	if cfg.DatabaseURL != "postgres://from-env@localhost:5432/db" {
		t.Errorf("DatabaseURL = %q, want значение из env", cfg.DatabaseURL)
	}
}

func TestLoad_DefaultLogLevel(t *testing.T) {
	path := writeTestConfig(t, `
check_interval: 5s
timeout: 10s
server:
  port: 8080
`)

	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want default \"info\"", cfg.LogLevel)
	}
}
