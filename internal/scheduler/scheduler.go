package scheduler

import (
	"fmt"
	"time"

	"github.com/sakashimaa/site-monitor/internal/checker"
	"github.com/sakashimaa/site-monitor/internal/config"
)

type Scheduler struct {
	Config *config.Config
	done   chan struct{}
}

func NewScheduler(config *config.Config) *Scheduler {
	return &Scheduler{
		Config: config,
		done:   make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	ticker := time.NewTicker(s.Config.CheckInterval)
	defer ticker.Stop()

	checkAll := func() {
		for _, site := range s.Config.Sites {
			res := checker.CheckSite(site.URL, s.Config)

			timeStr := time.Now().Format("2006-01-02 15:04:05")

			if res.AvailableStatus {
				fmt.Printf("[%s] Site %s ok\n", timeStr, site.URL)
			} else {
				fmt.Printf("[%s] Site %s NOT ok\n", timeStr, site.URL)
			}
		}
	}

	// Первая проверка сразу как по условию
	checkAll()

	for {
		select {
		case <-ticker.C:
			checkAll()
		case <-s.done:
			fmt.Println("Scheduler stopped")
			return
		}
	}
}

func (s *Scheduler) Stop() {
	close(s.done)
}
