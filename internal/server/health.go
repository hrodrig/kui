package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/hrodrig/kui/internal/log"
)

type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database,omitempty"`
}

// getHealthz is a simple liveness probe — the server is up and serving.
func (s *Server) getHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(healthResponse{Status: "ok"})
}

// getReadyz is a readiness probe — verifies the database connection is alive.
func (s *Server) getReadyz(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	dbStatus := "ok"
	code := http.StatusOK

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.store.PingContext(ctx); err != nil {
		s.log.Warn("readyz: db ping failed: %v", err)
		status = "not ready"
		dbStatus = "error"
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(healthResponse{
		Status:   status,
		Database: dbStatus,
	})
}

// LogHealthCheck is a no-op logger wrapper to suppress health-check noise.
// Use as middleware when health checks would otherwise pollute access logs.
func LogHealthCheck(next http.Handler, log *log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Debug("health check: %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
