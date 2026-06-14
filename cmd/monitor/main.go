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

	"github.com/sakashimaa/site-monitor/internal/api"
	"github.com/sakashimaa/site-monitor/internal/config"
	scheduler2 "github.com/sakashimaa/site-monitor/internal/scheduler"
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

	apiServer := api.NewServer(cfg)

	scheduler := scheduler2.NewScheduler(cfg)

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
