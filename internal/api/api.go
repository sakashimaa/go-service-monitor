package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sakashimaa/site-monitor/internal/config"
)

type Server struct {
	httpServer *http.Server
}

func NewServer(cfg *config.Config) *Server {
	mainMux := http.NewServeMux()

	v1Mux := http.NewServeMux()

	v1Mux.HandleFunc("GET /ping", pingHandler)
	mainMux.Handle("/api/v1/", http.StripPrefix("/api/v1", v1Mux))

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mainMux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return &Server{
		httpServer: srv,
	}
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp := map[string]any{
		"message": "pong",
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("payload encoding error",
			slog.String("handler", "pingHandler"),
			slog.String("error", err.Error()))

		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
