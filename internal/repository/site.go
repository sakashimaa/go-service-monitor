package repository

import (
	"context"
	"sort"
	"sync"

	"github.com/sakashimaa/site-monitor/internal/domain"
)

type SiteRepository interface {
	GetAll(ctx context.Context) ([]domain.Site, error)
}

type InMemoryRepo struct {
	data map[int]domain.Site
	mu   sync.RWMutex
}

func NewSiteRepository(data map[int]domain.Site) SiteRepository {
	return &InMemoryRepo{
		data: data,
		mu:   sync.RWMutex{},
	}
}

func (r *InMemoryRepo) GetAll(ctx context.Context) ([]domain.Site, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]domain.Site, 0, len(r.data))
	for _, v := range r.data {
		res = append(res, v)
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].ID < res[j].ID
	})

	return res, nil
}
