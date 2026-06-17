package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sakashimaa/site-monitor/internal/config"
	"github.com/sakashimaa/site-monitor/internal/handler"
)

type Server struct {
	httpServer *http.Server
}

func NewServer(cfg *config.Config, siteHandler handler.SiteHandler) *Server {
	mainMux := http.NewServeMux()

	v1Mux := http.NewServeMux()

	v1Mux.HandleFunc("GET /sites/{id}/status", siteHandler.SiteStatus)
	v1Mux.HandleFunc("GET /ping", siteHandler.Ping)
	v1Mux.HandleFunc("GET /sites", siteHandler.Sites)
	v1Mux.HandleFunc("POST /sites", siteHandler.CreateSite)
	v1Mux.HandleFunc("DELETE /sites/{id}", siteHandler.DeleteSite)
	v1Mux.HandleFunc("GET /health", siteHandler.HealhCheck)

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
