package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

const staticPath = "./static"

func (s *Server) mountStaticHandlers() {
	s.router.Handle("/*", http.FileServer(http.Dir(staticPath)))
	s.router.Route("/favicon.ico", func(r chi.Router) { r.Get("/", faviconHandler) })
}

// http.HandleFunc
func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, fmt.Sprintf("%s/favicon.ico", staticPath))
}
