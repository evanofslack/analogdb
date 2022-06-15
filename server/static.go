package server

import "net/http"

func (s *Server) mountStatic() {
	s.router.Handle("/*", http.FileServer(http.Dir("../../static")))
}
