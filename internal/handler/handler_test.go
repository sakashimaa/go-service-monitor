package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sakashimaa/site-monitor/internal/domain"
	"github.com/sakashimaa/site-monitor/internal/repository"
)

type mockSiteService struct {
	getAllFunc     func(ctx context.Context) ([]domain.Site, error)
	createSiteFunc func(ctx context.Context, req *domain.CreateSiteRequest) (*domain.Site, error)
	deleteSiteFunc func(ctx context.Context, id string) error
	getStatusFunc  func(ctx context.Context, id string) (*domain.CheckHistory, error)
}

func (m *mockSiteService) GetAll(ctx context.Context) ([]domain.Site, error) {
	return m.getAllFunc(ctx)
}

func (m *mockSiteService) CreateSite(ctx context.Context, req *domain.CreateSiteRequest) (*domain.Site, error) {
	return m.createSiteFunc(ctx, req)
}

func (m *mockSiteService) DeleteSite(ctx context.Context, id string) error {
	return m.deleteSiteFunc(ctx, id)
}

func (m *mockSiteService) GetStatus(ctx context.Context, id string) (*domain.CheckHistory, error) {
	return m.getStatusFunc(ctx, id)
}

func (m *mockSiteService) PingDB(ctx context.Context) error {
	panic("PingDB must not be called from CRUD-handler tests")
}

func (m *mockSiteService) GetHistory(
	ctx context.Context,
	id string,
	limit int,
	cursor *time.Time,
) ([]domain.CheckHistory, error) {
	panic("GetHistory must not be called from CRUD-handler tests")
}

func TestSites(t *testing.T) {
	tests := []struct {
		name        string
		getAllFunc  func(ctx context.Context) ([]domain.Site, error)
		wantStatus  int
		wantBodyLen int
	}{
		{
			name: "succeeded list of sites",
			getAllFunc: func(ctx context.Context) ([]domain.Site, error) {
				return []domain.Site{
					{ID: "1", Name: "Google", URL: "https://google.com"},
					{ID: "2", Name: "GitHub", URL: "https://github.com"},
				}, nil
			},
			wantStatus:  http.StatusOK,
			wantBodyLen: 2,
		},
		{
			name: "service error -> 500",
			getAllFunc: func(ctx context.Context) ([]domain.Site, error) {
				return nil, fmt.Errorf("db connection lost")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockSiteService{getAllFunc: tt.getAllFunc}
			h := NewSiteHandler(svc, "test-version")

			req := httptest.NewRequest(http.MethodGet, "/sites", nil)
			rec := httptest.NewRecorder()

			h.Sites(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus != http.StatusOK {
				return
			}

			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}

			var sites []domain.Site
			if err := json.Unmarshal(rec.Body.Bytes(), &sites); err != nil {
				t.Fatalf("failed to unmarshal response body: %v", err)
			}
			if len(sites) != tt.wantBodyLen {
				t.Errorf("len(sites) = %d, want %d", len(sites), tt.wantBodyLen)
			}
		})
	}
}

func TestCreateSite(t *testing.T) {
	validBody := `{"name": "Google", "url": "https://google.com"}`

	tests := []struct {
		name           string
		body           string
		createSiteFunc func(ctx context.Context, req *domain.CreateSiteRequest) (*domain.Site, error)
		wantStatus     int
	}{
		{
			name: "successful creation",
			body: validBody,
			createSiteFunc: func(ctx context.Context, req *domain.CreateSiteRequest) (*domain.Site, error) {
				return &domain.Site{ID: uuid.NewString(), Name: req.Name, URL: req.URL}, nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid body json",
			body:       `{"name": "Google", "url":`,
			wantStatus: http.StatusBadRequest,
			// createSiteFunc не задан намеренно: Decode должен упасть раньше, чем дойдёт до сервиса
		},
		{
			name:       "empty url -> validation error",
			body:       `{"name": "Google", "url": ""}`,
			wantStatus: http.StatusBadRequest,
			// аналогично: Validate() должен отсечь запрос до вызова сервиса
		},
		{
			name: "url already exists -> 409",
			body: validBody,
			createSiteFunc: func(ctx context.Context, req *domain.CreateSiteRequest) (*domain.Site, error) {
				return nil, repository.ErrURLAlreadyExists
			},
			wantStatus: http.StatusConflict,
		},
		{
			name: "internal server error -> 500",
			body: validBody,
			createSiteFunc: func(ctx context.Context, req *domain.CreateSiteRequest) (*domain.Site, error) {
				return nil, fmt.Errorf("tx failed")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockSiteService{createSiteFunc: tt.createSiteFunc}
			h := NewSiteHandler(svc, "test-version")

			req := httptest.NewRequest(http.MethodPost, "/sites", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			h.CreateSite(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus != http.StatusCreated {
				return
			}

			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}

			var site domain.Site
			if err := json.Unmarshal(rec.Body.Bytes(), &site); err != nil {
				t.Fatalf("failed to unmarshal response body: %v", err)
			}
			if site.Name != "Google" || site.URL != "https://google.com" {
				t.Errorf("unexpected site in response: %+v", site)
			}
		})
	}
}

func TestDeleteSite(t *testing.T) {
	validID := uuid.NewString()

	tests := []struct {
		name           string
		id             string
		deleteSiteFunc func(ctx context.Context, id string) error
		wantStatus     int
	}{
		{
			name:       "invalid id format -> 400",
			id:         "not-a-uuid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "successful creation -> 204",
			id:   validID,
			deleteSiteFunc: func(ctx context.Context, id string) error {
				return nil
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name: "site not found -> 404",
			id:   validID,
			deleteSiteFunc: func(ctx context.Context, id string) error {
				return domain.ErrSiteNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "internal error -> 500",
			id:   validID,
			deleteSiteFunc: func(ctx context.Context, id string) error {
				return fmt.Errorf("db is down")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockSiteService{deleteSiteFunc: tt.deleteSiteFunc}
			h := NewSiteHandler(svc, "test-version")

			req := httptest.NewRequest(http.MethodDelete, "/sites/"+tt.id, nil)
			req.SetPathValue("id", tt.id) // мы вызываем хендлер напрямую, минуя роутеры,
			// поэтому path-параметр нужно проставить руками - обычно это делает роутер
			rec := httptest.NewRecorder()

			h.DeleteSite(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestSiteStatus(t *testing.T) {
	validID := uuid.NewString()

	tests := []struct {
		name          string
		id            string
		getStatusFunc func(ctx context.Context, id string) (*domain.CheckHistory, error)
		wantStatus    int
	}{
		{
			name:       "invalid id format -> 400",
			id:         "not-a-uuid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "history ok -> 200",
			id:   validID,
			getStatusFunc: func(ctx context.Context, id string) (*domain.CheckHistory, error) {
				return &domain.CheckHistory{
					ID:           uuid.NewString(),
					SiteID:       id,
					Status:       domain.StatusOK,
					ResponseCode: 200,
					ResponseTime: 42,
				}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "no history -> 404",
			id:   validID,
			getStatusFunc: func(ctx context.Context, id string) (*domain.CheckHistory, error) {
				return nil, domain.ErrSiteHistoryNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "internal error -> 500",
			id:   validID,
			getStatusFunc: func(ctx context.Context, id string) (*domain.CheckHistory, error) {
				return nil, fmt.Errorf("query failed")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockSiteService{getStatusFunc: tt.getStatusFunc}
			h := NewSiteHandler(svc, "test-version")

			req := httptest.NewRequest(http.MethodGet, "/sites/"+tt.id+"/status", nil)
			req.SetPathValue("id", tt.id)
			rec := httptest.NewRecorder()

			h.SiteStatus(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus != http.StatusOK {
				return
			}

			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}

			var history domain.CheckHistory
			if err := json.Unmarshal(rec.Body.Bytes(), &history); err != nil {
				t.Fatalf("failed to unmarshal response body: %v", err)
			}
			if history.SiteID != tt.id {
				t.Errorf("SiteID = %q, want %q", history.SiteID, tt.id)
			}
		})
	}
}
