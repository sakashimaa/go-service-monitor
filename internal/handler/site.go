package handler

import (
	"log/slog"
	"net/http"

	"github.com/sakashimaa/site-monitor/internal/lib"
	"github.com/sakashimaa/site-monitor/internal/service"
)

type SiteHandler interface {
	Ping(w http.ResponseWriter, r *http.Request)
	Sites(w http.ResponseWriter, r *http.Request)
}

type HTTPHandler struct {
	service service.SiteService
}

func NewSiteHandler(service service.SiteService) SiteHandler {
	return &HTTPHandler{
		service: service,
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
