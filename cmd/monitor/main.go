package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/sakashimaa/site-monitor/internal/checker"
	"github.com/sakashimaa/site-monitor/internal/config"
)

func main() {
	configPath := flag.String("config", "configs/sites.yaml", "path to YAML configuration file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Fatal error: %v", err)
	}

	for _, site := range cfg.Sites {
		res := checker.CheckSite(site.URL, cfg)
		if !res.AvailableStatus {
			fmt.Printf("Site %s (%s) is NOT ok\n", site.Name, site.URL)
			continue
		}

		fmt.Printf("Site %s (%s) ok\n", site.Name, site.URL)
	}
}
