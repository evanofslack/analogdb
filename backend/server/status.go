package server

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/go-chi/chi/v5"
)

const readyCheckTimeout = 1 * time.Second

const (
	readyOK       = "ok"
	readyDown     = "down"
	readyDisabled = "disabled"
)

const (
	pingRoute   = "/ping"
	healthRoute = "/healthz"
	readyRoute  = "/readyz"
)

func (s *Server) mountStatusHandlers() {
	s.router.Route(pingRoute, func(r chi.Router) { r.Get("/", s.ping) })
	s.router.Route(healthRoute, func(r chi.Router) { r.Get("/", s.healthz) })
	s.router.Route(readyRoute, func(r chi.Router) { r.Get("/", s.readyz) })
}

func (s *Server) ping(w http.ResponseWriter, r *http.Request) {
	if err := encodeResponse(w, r, http.StatusOK, "message: pong"); err != nil {
		s.writeError(w, r, err)
	}
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	if !s.healthy.Load() {
		err := &analogdb.Error{Code: analogdb.ERRUNAVAILABLE, Message: "service not available"}
		s.writeError(w, r, err)
		return
	}
	if err := encodeResponse(w, r, http.StatusOK, "message: healthy"); err != nil {
		s.writeError(w, r, err)
	}
}

func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	if !s.healthy.Load() {
		err := &analogdb.Error{Code: analogdb.ERRUNAVAILABLE, Message: "service shutting down"}
		s.writeError(w, r, err)
		return
	}

	checks := map[string]analogdb.ReadyService{
		"postgres": s.ReadyService,
		"redis":    s.CacheReadyService,
		"weaviate": s.VectorReadyService,
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	statuses := make(map[string]string, len(checks))
	for name, check := range checks {
		if check == nil {
			mu.Lock()
			statuses[name] = readyDisabled
			mu.Unlock()
			continue
		}
		wg.Add(1)
		go func(ctx context.Context, name string, check analogdb.ReadyService) {
			defer wg.Done()
			status := readyOK
			ctx, cancel := context.WithTimeout(ctx, readyCheckTimeout)
			defer cancel()
			if err := check.Readyz(ctx); err != nil {
				s.logger.WarnContext(ctx, "Readiness check failed", "dependency", name, "error", err)
				status = readyDown
			}
			mu.Lock()
			statuses[name] = status
			mu.Unlock()
		}(r.Context(), name, check)
	}
	wg.Wait()

	code := http.StatusOK
	if statuses["postgres"] != readyOK {
		code = http.StatusServiceUnavailable
	}
	if err := encodeResponse(w, r, code, statuses); err != nil {
		s.writeError(w, r, err)
	}
}
