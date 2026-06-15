package service

import (
	"context"

	"github.com/sakashimaa/site-monitor/internal/domain"
	"github.com/sakashimaa/site-monitor/internal/repository"
)

type SiteService interface {
	GetAll(ctx context.Context) ([]domain.Site, error)
}

type SiteServ struct {
	repo repository.SiteRepository
}

func NewSiteService(repo repository.SiteRepository) SiteService {
	return &SiteServ{repo: repo}
}

func (s *SiteServ) GetAll(ctx context.Context) ([]domain.Site, error) {
	return s.repo.GetAll(ctx)
}
