package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sakashimaa/site-monitor/internal/domain"
	"github.com/sakashimaa/site-monitor/internal/repository"
)

type SiteService interface {
	GetAll(ctx context.Context) ([]domain.Site, error)
	CreateSite(ctx context.Context, req *domain.CreateSiteRequest) (*domain.Site, error)
	DeleteSite(ctx context.Context, id string) error
	GetStatus(ctx context.Context, id string) (domain.SiteStatus, error)
	UpdateStatus(ctx context.Context, id string, data domain.SiteStatus) error
	PingDB(ctx context.Context) error
}

type SiteServ struct {
	repo   repository.SiteRepository
	dbPool *pgxpool.Pool
}

func NewSiteService(repo repository.SiteRepository, dbPool *pgxpool.Pool) SiteService {
	return &SiteServ{
		repo:   repo,
		dbPool: dbPool,
	}
}

func (s *SiteServ) PingDB(ctx context.Context) error {
	if s.dbPool == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	return s.dbPool.Ping(ctx)
}

func (s *SiteServ) GetStatus(ctx context.Context, id string) (domain.SiteStatus, error) {
	return s.repo.GetStatus(ctx, id)
}

func (s *SiteServ) UpdateStatus(ctx context.Context, id string, data domain.SiteStatus) error {
	return s.repo.UpdateStatus(ctx, id, data)
}

func (s *SiteServ) DeleteSite(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *SiteServ) GetAll(ctx context.Context) ([]domain.Site, error) {
	return s.repo.GetAll(ctx)
}

func (s *SiteServ) CreateSite(ctx context.Context, req *domain.CreateSiteRequest) (*domain.Site, error) {
	id := uuid.New().String()
	site := &domain.Site{
		ID:   id,
		Name: req.Name,
		URL:  req.URL,
	}

	err := s.repo.Create(ctx, site)
	if err != nil {
		return nil, err
	}

	return site, nil
}
