package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sakashimaa/site-monitor/internal/checker"
	"github.com/sakashimaa/site-monitor/internal/config"
	"github.com/sakashimaa/site-monitor/internal/domain"
	"github.com/sakashimaa/site-monitor/internal/messaging"
	"github.com/sakashimaa/site-monitor/internal/repository"
)

type Scheduler struct {
	config      *config.Config
	done        chan struct{}
	wg          sync.WaitGroup
	repo        repository.SiteRepository
	historyRepo repository.CheckHistoryRepository
	publisher   messaging.EventPublisher
}

func NewScheduler(
	config *config.Config,
	repo repository.SiteRepository,
	historyRepo repository.CheckHistoryRepository,
	publisher messaging.EventPublisher,
) *Scheduler {
	return &Scheduler{
		config:      config,
		done:        make(chan struct{}),
		wg:          sync.WaitGroup{},
		repo:        repo,
		historyRepo: historyRepo,
		publisher:   publisher,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.wg.Add(1)
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.CheckInterval)
	defer ticker.Stop()

	checkAll := func(c context.Context) {
		sites, err := s.repo.GetAll(c)
		if err != nil {
			slog.Error("failed to retrieve sites from repo", slog.String("error", err.Error()))
			return
		}

		// в будущем на больших объемах надо будет переписать на конкуретную обработку при помощи горутин
		for _, site := range sites {
			select {
			case <-c.Done():
				return
			default:
			}

			res := checker.CheckSite(site.URL, s.config)

			statusVal := domain.StatusOK
			var errStr *string

			if res.Error != nil || !res.AvailableStatus {
				statusVal = domain.StatusError
				if res.Error != nil {
					e := res.Error.Error()
					errStr = &e
				}
			}

			h := &domain.CheckHistory{
				ID:           uuid.NewString(),
				SiteID:       site.ID,
				Status:       statusVal,
				ResponseCode: res.ResponseCode,
				ResponseTime: res.ResponseTime.Milliseconds(),
				Error:        errStr,
			}
			if err := s.historyRepo.Create(c, h); err != nil {
				slog.Error("failed to create history record", slog.String("error", err.Error()))
			}

			event := messaging.SiteCheckEvent{
				SiteID:       site.ID,
				URL:          site.URL,
				StatusCode:   res.ResponseCode,
				IsAvailable:  res.AvailableStatus,
				ResponseTime: res.ResponseTime.Milliseconds(),
				CheckedAt:    time.Now(),
				ErrorMessage: errStr,
			}
			if err := s.publisher.Publish(c, event); err != nil {
				slog.Error("failed to publish event", slog.String("site_id", site.ID), slog.String("error", err.Error()))
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
	checkAll(ctx)

	// приоритизированный селект, для того чтобы избежать рандомного выбора (из-за чего приложение может не остановится)
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.done:
			return
		case <-ticker.C:
			checkAll(ctx)
		}
	}
}

func (s *Scheduler) Stop() {
	close(s.done)
	s.wg.Wait()
}
