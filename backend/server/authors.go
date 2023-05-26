package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type AuthorsResponse struct {
	Authors []string `json:"authors"`
}

const authorsPath = "/authors"

func (s *Server) mountAuthorHandlers() {
	s.router.Route(authorsPath, func(r chi.Router) {
		r.Get("/", s.getAuthors)
	})
}

func (s *Server) getAuthors(w http.ResponseWriter, r *http.Request) {
	authors, err := s.AuthorService.FindAuthors(r.Context())
	if err != nil {
		writeError(w, r, err)
	}
	authorsResponse := AuthorsResponse{
		Authors: authors,
	}
	if err := encodeResponse(w, r, http.StatusOK, authorsResponse); err != nil {
		writeError(w, r, err)
	}
}
