package server

import (
	"net/http"

	"github.com/arl/statsviz"
)

func (s *Server) mountStatsHandlers() {

	s.router.Get("/debug/statsviz/ws", statsviz.Ws)
	s.router.Get("/debug/statsviz", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/debug/statsviz/", 301)
	})
	s.router.With(auth).Handle("/debug/statsviz/*", statsviz.Index)
}
