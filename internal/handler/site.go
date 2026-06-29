package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
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
	HealthCheck(w http.ResponseWriter, r *http.Request)
	SiteHistory(w http.ResponseWriter, r *http.Request)
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

// SiteHistory godoc
// @Summary		Получить историю проверок сайта по id
// @Description Возвращает пагинированную страницу истории проверок конкретного сайта (курсорная пагинация)
// @Tags		system
// @Produce		json
// @Success		200 {object} domain.SiteHistoryResponse
// @Router		/sites/{id}/history [get]
func (h *HTTPHandler) SiteHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		http.Error(w, "id is invalid format: must be uuid", http.StatusBadRequest)
		return
	}

	limitNum := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limitNum = min(n, 20)
		}
	}

	var cursor *time.Time
	if c := r.URL.Query().Get("cursor"); c != "" {
		t, err := time.Parse(time.RFC3339, c)
		if err != nil {
			http.Error(w, "cursor is invalid format: must be RFC3339 date", http.StatusBadRequest)
			return
		}
		cursor = &t
	}

	history, err := h.service.GetHistory(r.Context(), id, limitNum, cursor)
	if err != nil {
		if errors.Is(err, domain.ErrSiteHistoryNotFound) {
			resp := &domain.SiteHistoryResponse{
				Data:    []domain.CheckHistory{},
				HasMore: false,
			}

			if err := lib.WriteJSON(w, http.StatusOK, resp); err != nil {
				slog.Error("encode resp failed", slog.String("error", err.Error()))
			}
			return
		}

		slog.Error("failed to get site history", slog.String("error", err.Error()), slog.String("site_id", id))
		http.Error(w, "failed to query history for this site", http.StatusInternalServerError)
		return
	}

	var nextCursor *time.Time
	hasMore := len(history) == limitNum
	if hasMore {
		t := history[len(history)-1].CreatedAt
		nextCursor = &t
	}

	resp := &domain.SiteHistoryResponse{
		Data:       history,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}

	if err := lib.WriteJSON(w, http.StatusOK, resp); err != nil {
		slog.Error("encode resp failed", slog.String("error", err.Error()))
	}
}

// HealthCheck godoc
// @Summary				Проверка доступности (здоровья) сервиса
// @Description		Возвращает текущий статус сервиса, аптайи, версию и статус зависимостей (пока пусто)
// @Tags					system
// @Produce				json
// @Success				200	{object}	domain.HealthResponse
// @Router				/health [get]
func (h *HTTPHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	httpCode := http.StatusOK
	deps := make(map[string]string)

	err := h.service.PingDB(r.Context())
	if err != nil {
		slog.Error("healthcheck: database is unreachable", slog.String("error", err.Error()))
		deps["database"] = "unhealthy"
		status = "unhealthy"
		httpCode = http.StatusServiceUnavailable
	} else {
		deps["database"] = "healthy"
	}

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

// SiteStatus GoDoc
// @Summary      Статус сайта
// @Description  Возвращает результаты последней проверки сайта
// @Tags         sites
// @Produce      json
// @Param        id   path      string  true  "id сайта"
// @Success      200  {object}  domain.SiteStatus
// @Failure      400  {string}  string "invalid id format"
// @Failure      404  {string}  string "site not found"
// @Router       /sites/{id}/status [get]
func (h *HTTPHandler) SiteStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if _, err := uuid.Parse(id); err != nil {
		http.Error(w, "invalid id format: must be uuid", http.StatusBadRequest)
		return
	}

	status, err := h.service.GetStatus(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrSiteHistoryNotFound) {
			http.Error(w, "history not found", http.StatusNotFound)
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

// DeleteSite godoc
// @Summary      Удалить сайт
// @Description  Удаляет сайт из мониторинга по его UUID
// @Tags         sites
// @Param        id   path      string  true  "UUID сайта"
// @Success      204  "No Content"
// @Failure      400  {string}  string "invalid id format"
// @Failure      404  {string}  string "site not found"
// @Router       /sites/{id} [delete]
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

// CreateSite godoc
// @Summary      Добавить сайт
// @Description  Добавляет новый сайт для проверки
// @Tags         sites
// @Accept       json
// @Produce      json
// @Param        request body domain.CreateSiteRequest true "Данные сайта"
// @Success      201  {object}  domain.Site
// @Failure      400  {string}  string "invalid json body или ошибка валидации"
// @Failure      409  {string}  string "url already exists"
// @Router       /sites [post]
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

// Ping godoc
// @Summary				Пинг-понг
// @Description 	Быстрая проверка доступности сервиса
// @Tags 					system
// @Produce				json
// @Success				200	{object}	domain.PingResponse
// @Router				/ping [get]
func (h *HTTPHandler) Ping(w http.ResponseWriter, r *http.Request) {
	resp := domain.PingResponse{
		Message: "pong",
	}

	if err := lib.WriteJSON(w, http.StatusOK, resp); err != nil {
		slog.Error("encode resp failed", slog.String("error", err.Error()), slog.String("handler", "Ping"))
	}
}

// Sites godoc
// @Summary      Список сайтов
// @Description  Получить список всех сайтов, находящихся на мониторинге
// @Tags         sites
// @Produce      json
// @Success      200  {array}   domain.Site
// @Router       /sites [get]
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
