package server

import (
	"net/http"

	"github.com/evanofslack/analogdb"
	"github.com/go-chi/chi/v5"
)

func (s *Server) mountStatusHandlers() {

	s.router.Route("/ping", func(r chi.Router) { r.Get("/", s.ping) })
	s.router.Route("/healthz", func(r chi.Router) { r.Get("/", s.healthz) })
	s.router.Route("/readyz", func(r chi.Router) { r.Get("/", s.readyz) })
}

func (s *Server) ping(w http.ResponseWriter, r *http.Request) {
	if err := encodeResponse(w, r, http.StatusOK, "message: pong"); err != nil {
		s.writeError(w, r, err)
	}
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	if !s.healthy {
		err := &analogdb.Error{Code: analogdb.ERRUNAVAILABLE, Message: "Service not available"}
		s.writeError(w, r, err)
	}
	if err := encodeResponse(w, r, http.StatusOK, "message: healthy"); err != nil {
		s.writeError(w, r, err)
	}
}

func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	err := s.ReadyService.Readyz(r.Context())
	if err != nil {
		s.writeError(w, r, err)
	}
	if err := encodeResponse(w, r, http.StatusOK, "message: ready"); err != nil {
		s.writeError(w, r, err)
	}
}
