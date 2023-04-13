package server

import "net/http"

func (s *Server) mountStaticHandlers() {
	s.router.Handle("/*", http.FileServer(http.Dir("./static")))
}
