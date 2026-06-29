package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sakashimaa/site-monitor/internal/config"
	"github.com/sakashimaa/site-monitor/internal/handler"
	"github.com/sakashimaa/site-monitor/internal/middleware"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/sakashimaa/site-monitor/docs"
)

type Server struct {
	httpServer *http.Server
}

// i = 2 (Logging); h = Logging(v1Mux)
// i = 1 (RequestID); h = RequestID(Logging(v1Mux))
// i = 0 (Recovery); h = Recovery(RequestID(Logging(v1Mux)))
func chainMiddleware(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}

	return h
}

func NewServer(cfg *config.Config, siteHandler handler.SiteHandler) *Server {
	mainMux := http.NewServeMux()

	v1Mux := http.NewServeMux()

	v1Mux.HandleFunc("GET /sites/{id}/status", siteHandler.SiteStatus)
	v1Mux.HandleFunc("GET /ping", siteHandler.Ping)
	v1Mux.HandleFunc("GET /sites", siteHandler.Sites)
	v1Mux.HandleFunc("POST /sites", siteHandler.CreateSite)
	v1Mux.HandleFunc("DELETE /sites/{id}", siteHandler.DeleteSite)
	v1Mux.HandleFunc("GET /health", siteHandler.HealthCheck)
	v1Mux.HandleFunc("GET /sites/{id}/history", siteHandler.SiteHistory)

	v1Handler := chainMiddleware(
		v1Mux,
		middleware.RequestID,
		middleware.Recovery,
		middleware.Logging,
	)

	mainMux.Handle("/api/v1/", http.StripPrefix("/api/v1", v1Handler))

	mainMux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

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
