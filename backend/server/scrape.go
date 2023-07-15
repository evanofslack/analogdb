package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type keywordsUpdatedResponse struct {
	Ids []int `json:"ids"`
}

const (
	scrapePath          = "/scrape"
	keywordsUpdatedPath = scrapePath + "/keywords/updated"
)

func (s *Server) mountScrapeHandlers() {
	s.router.Route(keywordsUpdatedPath, func(r chi.Router) {
		r.With(auth).Get("/", s.getKeywordUpdatedPosts)
	})
}

func (s *Server) getKeywordUpdatedPosts(w http.ResponseWriter, r *http.Request) {
	ids, err := s.ScrapeService.KeywordUpdatedPostIDs(r.Context())
	if err != nil {
		s.writeError(w, r, err)
	}
	response := keywordsUpdatedResponse{
		Ids: ids,
	}
	if err := encodeResponse(w, r, http.StatusOK, response); err != nil {
		s.writeError(w, r, err)
	}
}
