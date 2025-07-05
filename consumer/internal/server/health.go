package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// AddHealthChecker registers a health checker for a named service
func (s *Server) AddHealthChecker(name string, checker HealthChecker) {
	s.checkers[name] = checker
	s.logger.Debug("health checker registered", "service", name)
}

type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services,omitempty"`
	Error     string            `json:"error,omitempty"`
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	status := HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(status); err != nil {
		s.logger.Error("failed to encode health response", "error", err)
	}
}

func (s *Server) readinessHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	status := HealthStatus{
		Status:    "ready",
		Timestamp: time.Now(),
		Services:  make(map[string]string),
	}
	allReady := true
	for name, checker := range s.checkers {
		if err := checker.HealthCheck(ctx); err != nil {
			status.Services[name] = fmt.Sprintf("unhealthy: %v", err)
			allReady = false
			s.logger.Warn("service health check failed", "service", name, "error", err)
		} else {
			status.Services[name] = "healthy"
		}
	}

	w.Header().Set("Content-Type", "application/json")

	if !allReady {
		status.Status = "not ready"
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	if err := json.NewEncoder(w).Encode(status); err != nil {
		s.logger.Error("failed to encode readiness response", "error", err)
	}
}
