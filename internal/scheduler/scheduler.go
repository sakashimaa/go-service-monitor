package scheduler

import (
	"log/slog"
	"sync"
	"time"

	"github.com/sakashimaa/site-monitor/internal/checker"
	"github.com/sakashimaa/site-monitor/internal/config"
)

type Scheduler struct {
	Config *config.Config
	done   chan struct{}
	wg     sync.WaitGroup
}

func NewScheduler(config *config.Config) *Scheduler {
	return &Scheduler{
		Config: config,
		done:   make(chan struct{}),
		wg:     sync.WaitGroup{},
	}
}

func (s *Scheduler) Start() {
	s.wg.Add(1)
	defer s.wg.Done()

	ticker := time.NewTicker(s.Config.CheckInterval)
	defer ticker.Stop()

	checkAll := func() {
		for _, site := range s.Config.Sites {
			res := checker.CheckSite(site.URL, s.Config)
			if res.Error != nil {
				slog.Error("site check failed",
					slog.String("url", site.URL),
					slog.String("error", res.Error.Error()),
				)
				continue
			}

			if res.AvailableStatus {
				slog.Info("site ok",
					slog.String("url", site.URL),
					slog.Int("status_code", res.ResponseCode),
				)
			} else {
				slog.Warn("site NOT ok",
					slog.String("url", site.URL),
					slog.Int("status_code", res.ResponseCode),
				)
			}
		}
	}

	// Первая проверка сразу как по условию
	checkAll()

	// приоритизированный селект, для того чтобы избежать рандомного выбора (из-за чего приложение может не остановится)
	for {
		select {
		case <-s.done:
			return
		default:
		}

		select {
		case <-ticker.C:
			checkAll()
		case <-s.done:
			return
		}
	}
}

func (s *Scheduler) Stop() {
	close(s.done)
	s.wg.Wait()
}
