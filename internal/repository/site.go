package repository

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/sakashimaa/site-monitor/internal/domain"
)

var (
	ErrURLAlreadyExists = errors.New("URL already exists")
	ErrSiteNotFound     = errors.New("site not found")
)

type SiteRepository interface {
	GetAll(ctx context.Context) ([]domain.Site, error)
	Create(ctx context.Context, req *domain.Site) error
	Delete(ctx context.Context, id string) error
}

type InMemoryRepo struct {
	data map[string]domain.Site
	mu   sync.RWMutex
}

func NewSiteRepository(data map[string]domain.Site) SiteRepository {
	return &InMemoryRepo{
		data: data,
		mu:   sync.RWMutex{},
	}
}

func (r *InMemoryRepo) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.data[id]; !ok {
		return ErrSiteNotFound
	}

	delete(r.data, id)
	return nil
}

func (r *InMemoryRepo) Create(ctx context.Context, req *domain.Site) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, v := range r.data {
		if v.URL == req.URL {
			return ErrURLAlreadyExists
		}
	}

	r.data[req.ID] = *req
	return nil
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
