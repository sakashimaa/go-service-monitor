package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/sakashimaa/site-monitor/internal/checker"
	"github.com/sakashimaa/site-monitor/internal/config"
	"github.com/sakashimaa/site-monitor/internal/domain"
	"github.com/sakashimaa/site-monitor/internal/repository"
)

type Scheduler struct {
	Config *config.Config
	done   chan struct{}
	wg     sync.WaitGroup
	repo   repository.SiteRepository
}

func NewScheduler(config *config.Config, repo repository.SiteRepository) *Scheduler {
	return &Scheduler{
		Config: config,
		done:   make(chan struct{}),
		wg:     sync.WaitGroup{},
		repo:   repo,
	}
}

func (s *Scheduler) Start() {
	s.wg.Add(1)
	defer s.wg.Done()

	ticker := time.NewTicker(s.Config.CheckInterval)
	defer ticker.Stop()

	checkAll := func() {
		sites, err := s.repo.GetAll(context.Background())
		if err != nil {
			slog.Error("failed to retrieve sites from repo", slog.String("error", err.Error()))
			return
		}

		// в будущем на больших объемах надо будет переписать на конкуретную обработку при помощи горутин
		for _, site := range sites {
			res := checker.CheckSite(site.URL, s.Config)

			now := time.Now()
			code := res.ResponseCode
			durationMs := res.ResponseTime.Milliseconds()

			statusVal := domain.StatusOK
			var errStr *string

			if res.Error != nil || !res.AvailableStatus {
				statusVal = domain.StatusError
				if res.Error != nil {
					e := res.Error.Error()
					errStr = &e
				}
			}

			status := domain.SiteStatus{
				URL:           site.URL,
				Status:        statusVal,
				ResponseCode:  &code,
				LastCheckTime: &now,
				ResponseTime:  &durationMs,
				Error:         errStr,
			}

			err = s.repo.UpdateStatus(context.Background(), site.ID, status)
			if err != nil {
				slog.Error("failed to update status", slog.String("error", err.Error()))
				continue
			}

			if res.Error != nil {
				slog.Error(
					"site check failed",
					slog.String("url", site.URL),
					slog.String("error", res.Error.Error()),
				)
				continue
			}

			if res.AvailableStatus {
				slog.Info(
					"site ok",
					slog.String("url", site.URL),
					slog.Int("status_code", res.ResponseCode),
				)
			} else {
				slog.Warn(
					"site NOT ok",
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
