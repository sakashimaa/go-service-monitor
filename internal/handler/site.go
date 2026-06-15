package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/sakashimaa/site-monitor/internal/domain"
	"github.com/sakashimaa/site-monitor/internal/lib"
	"github.com/sakashimaa/site-monitor/internal/repository"
	"github.com/sakashimaa/site-monitor/internal/service"
)

type SiteHandler interface {
	Ping(w http.ResponseWriter, r *http.Request)
	Sites(w http.ResponseWriter, r *http.Request)
	CreateSite(w http.ResponseWriter, r *http.Request)
}

type HTTPHandler struct {
	service service.SiteService
}

func NewSiteHandler(service service.SiteService) SiteHandler {
	return &HTTPHandler{
		service: service,
	}
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
