package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sakashimaa/site-monitor/internal/domain"
	"github.com/sakashimaa/site-monitor/internal/lib"
	"github.com/sakashimaa/site-monitor/internal/repository"
	"github.com/sakashimaa/site-monitor/internal/service"
)

type SiteHandler interface {
	Ping(w http.ResponseWriter, r *http.Request)
	Sites(w http.ResponseWriter, r *http.Request)
	CreateSite(w http.ResponseWriter, r *http.Request)
	DeleteSite(w http.ResponseWriter, r *http.Request)
	SiteStatus(w http.ResponseWriter, r *http.Request)
	HealhCheck(w http.ResponseWriter, r *http.Request)
}

type HTTPHandler struct {
	service   service.SiteService
	startTime time.Time
	version   string
}

func NewSiteHandler(service service.SiteService, version string) SiteHandler {
	return &HTTPHandler{
		service:   service,
		startTime: time.Now(),
		version:   version,
	}
}

func (h *HTTPHandler) HealhCheck(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	httpCode := http.StatusOK
	deps := make(map[string]string)

	// будущая проверка зависимостей
	// пример:
	//
	// err := h.service.PingDB(r.Context())
	// if err != nil {
	// 		deps["database"] = "unhealthy"
	// 		status = "unhealthy"
	// 		httpCode = http.StatusServiceUnavailable
	// } else {
	// 		deps["database"] = "healhy"
	// }

	resp := domain.HealthResponse{
		Status:       status,
		Version:      h.version,
		Uptime:       time.Since(h.startTime).String(),
		Timestamp:    time.Now(),
		Dependencies: deps,
	}

	if err := lib.WriteJSON(w, httpCode, resp); err != nil {
		slog.Error("failed to write health response", slog.String("error", err.Error()))
	}
}

func (h *HTTPHandler) SiteStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if _, err := uuid.Parse(id); err != nil {
		http.Error(w, "invalid id format: must be uuid", http.StatusBadRequest)
		return
	}

	status, err := h.service.GetStatus(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrSiteNotFound) {
			http.Error(w, "site not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to get site status", slog.String("error", err.Error()))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := lib.WriteJSON(w, http.StatusOK, status); err != nil {
		slog.Error("encode resp failed", slog.String("error", err.Error()), slog.String("handler", "SiteStatus"))
	}
}

func (h *HTTPHandler) DeleteSite(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if _, err := uuid.Parse(id); err != nil {
		http.Error(w, "invalid id format: must be uuid", http.StatusBadRequest)
		return
	}

	err := h.service.DeleteSite(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrSiteNotFound) {
			http.Error(w, "site not found", http.StatusNotFound)
			return
		}

		slog.Error("failed to delete site", slog.String("error", err.Error()))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandler) CreateSite(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateSiteRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	res, err := h.service.CreateSite(r.Context(), &req)
	if err != nil {
		if errors.Is(err, repository.ErrURLAlreadyExists) {
			http.Error(w, "url already exists", http.StatusConflict)
			return
		}

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := lib.WriteJSON(w, http.StatusCreated, res); err != nil {
		slog.Error("encode resp failed", slog.String("error", err.Error()), slog.String("handler", "CreateSite"))
	}
}

func (h *HTTPHandler) Ping(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{
		"message": "pong",
	}

	if err := lib.WriteJSON(w, http.StatusOK, resp); err != nil {
		slog.Error("encode resp failed", slog.String("error", err.Error()), slog.String("handler", "Ping"))
	}
}

func (h *HTTPHandler) Sites(w http.ResponseWriter, r *http.Request) {
	res, err := h.service.GetAll(r.Context())
	if err != nil {
		// добавить маппинг ошибок позже
		slog.Error("get all error",
			slog.String("error", err.Error()))

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := lib.WriteJSON(w, http.StatusOK, res); err != nil {
		slog.Error("encode resp failed", slog.String("error", err.Error()), slog.String("handler", "Sites"))
	}
}
