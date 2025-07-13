package server

import (
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

const (
	swaggerRoute = "/swagger"
)

func (s *Server) mountSwaggerHandlers() {
	s.router.Route(swaggerRoute, func(r chi.Router) { r.Get("/*", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json"))) })
}
