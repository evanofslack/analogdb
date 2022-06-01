package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) MountStatus() {

	s.router.Route("/ping", func(r chi.Router) { r.Get("/", s.ping) })
}

func (s *Server) ping(w http.ResponseWriter, r *http.Request) {
	if err := encodeResponse(w, r, "message: pong"); err != nil {
		writeError(w, r, err)
	}
}
