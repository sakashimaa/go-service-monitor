package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &responseWriter{ResponseWriter: w}
		reqID := GetRequestID(r.Context())

		slog.InfoContext(r.Context(), "request started", // задел на будущее, чтоб доставать что-то из логгера
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("request_id", reqID),
		)

		next.ServeHTTP(wrapped, r)

		slog.InfoContext(r.Context(), "request completed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("request_id", reqID),
			slog.Int("status", wrapped.status),
			slog.Duration("duration", time.Since(start)),
		)
	})
}
