package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/sakashimaa/site-monitor/internal/api"
	"github.com/sakashimaa/site-monitor/internal/config"
	"github.com/sakashimaa/site-monitor/internal/domain"
	"github.com/sakashimaa/site-monitor/internal/handler"
	"github.com/sakashimaa/site-monitor/internal/repository"
	"github.com/sakashimaa/site-monitor/internal/scheduler"
	"github.com/sakashimaa/site-monitor/internal/service"
	"github.com/sakashimaa/site-monitor/internal/storage"
)

// может быть перезаписана при сборке
// go build -ldflags "-X main.buildVersion=v1.4.2"
var buildVersion = "v1.0.0-dev"

// @title 				Go Service Monitor API
// @version 			1.0.0
// @description 	API для мониторинга доступности сайтов
// @host					localhost:8080
// @BasePath			/api/v1
func main() {
	_ = godotenv.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error("application failed to run", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "configs/sites.yaml", "path to YAML configuration file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	slog.Info(
		"Configuration loaded",
		slog.Int("port", cfg.Port),
		slog.String("log_level", cfg.LogLevel),
		slog.String("check_interval", cfg.CheckInterval.String()),
		slog.String("http_timeout", cfg.Timeout.String()),
	)

	dbCtx, cancelDB := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelDB()

	dbPool, err := storage.NewPostgresPool(dbCtx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to create postgres pool: %w", err)
	}
	defer dbPool.Close()

	slog.Info("successfully connected to PostgreSQL")

	repo := repository.NewPostgresRepository(dbPool)
	srv := service.NewSiteService(repo, dbPool)
	hndl := handler.NewSiteHandler(srv, buildVersion)

	for _, site := range cfg.Sites {
		id := uuid.New().String()
		err = repo.Create(context.Background(), &domain.Site{
			ID:   id,
			Name: site.Name,
			URL:  site.URL,
		})
		if err != nil && !errors.Is(err, repository.ErrURLAlreadyExists) {
			slog.Error("failed to insert site from config in DB", slog.String("error", err.Error()))
			continue
		}
	}

	apiServer := api.NewServer(cfg, hndl)

	sched := scheduler.NewScheduler(cfg, repo)

	slog.Info(
		"Site Monitor started",
		slog.Int("sites_count", len(cfg.Sites)),
		slog.String("interval", cfg.CheckInterval.String()),
	)

	// оставил обычный println чтоб человеку за консолью было понятно
	fmt.Println("Press Ctrl+C to stop")

	serverErr := make(chan error, 1)

	go func() {
		slog.Info("Starting HTTP server", slog.Int("port", cfg.Port))

		err := apiServer.Start()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, os.Interrupt)
	defer cancel()

	go sched.Start(ctx)

	select {
	case <-ctx.Done():
		slog.Info("received shutdown signal...")
	case err := <-serverErr:
		slog.Error("HTTP Server crashed", slog.String("error", err.Error()))
		return fmt.Errorf("http server failed: %w", err)
	}

	slog.Info("Shutting down...")

	sched.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	if err := apiServer.Stop(shutdownCtx); err != nil {
		slog.Error("HTTP Server failed to shutdown", slog.String("error", err.Error()))
	}

	slog.Info("Site monitor stopped gracefully.")
	return nil
}
