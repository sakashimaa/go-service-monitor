package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/sakashimaa/site-monitor/internal/domain"
	"github.com/sakashimaa/site-monitor/internal/repository"
)

type SiteService interface {
	GetAll(ctx context.Context) ([]domain.Site, error)
	CreateSite(ctx context.Context, req *domain.CreateSiteRequest) (*domain.Site, error)
	DeleteSite(ctx context.Context, id string) error
}

type SiteServ struct {
	repo repository.SiteRepository
}

func NewSiteService(repo repository.SiteRepository) SiteService {
	return &SiteServ{repo: repo}
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
