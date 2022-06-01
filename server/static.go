package http

import "net/http"

func (s *Server) MountStatic() {
	s.router.Handle("/*", http.FileServer(http.Dir("./static")))
}
