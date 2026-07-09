package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sakashimaa/site-monitor/internal/domain"
	"github.com/sakashimaa/site-monitor/internal/repository"
	"github.com/sakashimaa/site-monitor/internal/storage"
)

type SiteService interface {
	GetAll(ctx context.Context) ([]domain.Site, error)
	CreateSite(ctx context.Context, req *domain.CreateSiteRequest) (*domain.Site, error)
	DeleteSite(ctx context.Context, id string) error
	GetStatus(ctx context.Context, id string) (*domain.CheckHistory, error)
	PingDB(ctx context.Context) error
	GetHistory(ctx context.Context, id string, limit int, cursor *time.Time) ([]domain.CheckHistory, error)
}

type SiteServ struct {
	repo        repository.SiteRepository
	historyRepo repository.CheckHistoryRepository
	dbPool      *pgxpool.Pool
}

func NewSiteService(repo repository.SiteRepository, historyRepo repository.CheckHistoryRepository, dbPool *pgxpool.Pool) SiteService {
	return &SiteServ{
		repo:        repo,
		historyRepo: historyRepo,
		dbPool:      dbPool,
	}
}

func (s *SiteServ) GetHistory(ctx context.Context, id string, limit int, cursor *time.Time) ([]domain.CheckHistory, error) {
	if _, err := s.repo.GetById(ctx, id); err != nil {
		return nil, err
	}

	return s.historyRepo.GetHistory(ctx, id, limit, cursor)
}

func (s *SiteServ) PingDB(ctx context.Context) error {
	if s.dbPool == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	return s.dbPool.Ping(ctx)
}

func (s *SiteServ) GetStatus(ctx context.Context, id string) (*domain.CheckHistory, error) {
	return s.historyRepo.GetLatest(ctx, id)
}

func (s *SiteServ) DeleteSite(ctx context.Context, id string) error {
	return storage.WithTransaction(ctx, s.dbPool, func(tx pgx.Tx) error {
		if err := s.historyRepo.DeleteBySiteIdTx(ctx, tx, id); err != nil {
			return err
		}
		return s.repo.DeleteTx(ctx, tx, id)
	})
}

func (s *SiteServ) GetAll(ctx context.Context) ([]domain.Site, error) {
	return s.repo.GetAll(ctx)
}

func (s *SiteServ) CreateSite(ctx context.Context, req *domain.CreateSiteRequest) (*domain.Site, error) {
	site := &domain.Site{
		ID:   uuid.NewString(),
		Name: req.Name,
		URL:  req.URL,
	}

	err := storage.WithTransaction(ctx, s.dbPool, func(tx pgx.Tx) error {
		if err := s.repo.CreateTx(ctx, tx, site); err != nil {
			return err
		}

		h := &domain.CheckHistory{
			ID:     uuid.NewString(),
			SiteID: site.ID,
			Status: domain.StatusPending,
		}
		return s.historyRepo.CreateTx(ctx, tx, h)
	})

	if err != nil {
		return nil, err
	}
	return site, nil
}
