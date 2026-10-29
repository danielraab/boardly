package httpapi

import (
	"log/slog"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}

func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// logRequests writes a log line per request according to mode: "all",
// "errors" (status 500 and higher) or "none". A successful health check is
// never logged, so the container health check doesn't flood the log.
func logRequests(mode string, next http.Handler) http.Handler {
	if mode == "none" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		if mode == "errors" && rec.status < http.StatusInternalServerError {
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/api/health" && rec.status == http.StatusOK {
			return
		}
		slog.InfoContext(r.Context(), "request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration", time.Since(start),
		)
	})
}
