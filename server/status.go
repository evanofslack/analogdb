package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) mountStatus() {

	s.router.Route("/ping", func(r chi.Router) { r.Get("/", s.ping) })
	s.router.Route("/healthz", func(r chi.Router) { r.Get("/", s.healthz) })
	s.router.Route("/readyz", func(r chi.Router) { r.Get("/", s.readyz) })
}

func (s *Server) ping(w http.ResponseWriter, r *http.Request) {
	if err := encodeResponse(w, r, "message: pong"); err != nil {
		writeError(w, r, err)
	}
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	if err := encodeResponse(w, r, "message: healthy"); err != nil {
		writeError(w, r, err)
	}
}

func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	err := s.HealthService.Readyz(r.Context())
	if err != nil {
		writeError(w, r, err)
	}
	if err := encodeResponse(w, r, "message: ready"); err != nil {
		writeError(w, r, err)
	}
}
