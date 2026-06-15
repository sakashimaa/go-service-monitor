package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"github.com/sakashimaa/site-monitor/internal/api"
	"github.com/sakashimaa/site-monitor/internal/config"
	"github.com/sakashimaa/site-monitor/internal/domain"
	"github.com/sakashimaa/site-monitor/internal/handler"
	"github.com/sakashimaa/site-monitor/internal/repository"
	scheduler2 "github.com/sakashimaa/site-monitor/internal/scheduler"
	"github.com/sakashimaa/site-monitor/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	configPath := flag.String("config", "configs/sites.yaml", "path to YAML configuration file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("Failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	data := make(map[string]domain.Site, len(cfg.Sites))
	for _, site := range cfg.Sites {
		id := uuid.New().String()
		data[id] = domain.Site{
			ID:   id,
			Name: site.Name,
			URL:  site.URL,
		}
	}
	repo := repository.NewSiteRepository(data)
	service := service.NewSiteService(repo)
	handler := handler.NewSiteHandler(service)

	apiServer := api.NewServer(cfg, handler)

	scheduler := scheduler2.NewScheduler(cfg, repo)

	slog.Info(
		"Site Monitor started",
		slog.Int("sites_count", len(cfg.Sites)),
		slog.String("interval", cfg.CheckInterval.String()),
	)

	// оставил обычный println чтоб человеку за консолью было понятно
	fmt.Println("Press Ctrl+C to stop")

	go scheduler.Start()

	go func() {
		slog.Info("Starting HTTP server", slog.Int("port", cfg.Port))

		err := apiServer.Start()
		if err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP Server start failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, os.Interrupt)
	defer cancel()

	<-ctx.Done()

	slog.Info("Shutting down...")

	scheduler.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	if err := apiServer.Stop(shutdownCtx); err != nil {
		slog.Error("HTTP Server failed to shutdown", slog.String("error", err.Error()))
	}

	slog.Info("Site monitor stopped gracefully.")
}
