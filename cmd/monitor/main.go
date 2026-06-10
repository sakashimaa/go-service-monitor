package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sakashimaa/site-monitor/internal/config"
	scheduler2 "github.com/sakashimaa/site-monitor/internal/scheduler"
)

func main() {
	configPath := flag.String("config", "configs/sites.yaml", "path to YAML configuration file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Fatal error: %v", err)
	}

	scheduler := scheduler2.NewScheduler(cfg)
	go scheduler.Start()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, os.Interrupt)
	defer cancel()

	<-ctx.Done()

	fmt.Println("Shutting down gracefully...")
	scheduler.Stop()
}
