package server

import (
	"net/http"

	"github.com/evanofslack/analogdb"
	"github.com/go-chi/chi/v5"
)

const (
	keywordsPath        = "/keywords"
	defaultKeywordLimit = 50
)

type KeywordsResponse struct {
	Keywords []analogdb.KeywordSummary `json:"keywords"`
}

func (s *Server) mountKeywordHandlers() {
	s.router.Route(keywordsPath, func(r chi.Router) {
		r.Get("/summary", s.getSummary)
	})
}

func (s *Server) getSummary(w http.ResponseWriter, r *http.Request) {

	var limit = defaultLimit
	var err error

	if strLimit := r.URL.Query().Get("page_size"); strLimit != "" {
		limit, err = stringToInt(strLimit)
		if err != nil {
			s.writeError(w, r, err)
		}
	}

	keywords, err := s.KeywordService.GetKeywordSummary(r.Context(), limit)
	if err != nil {
		s.writeError(w, r, err)
	}
	response := KeywordsResponse{
		Keywords: *keywords,
	}
	if err := encodeResponse(w, r, http.StatusOK, response); err != nil {
		s.writeError(w, r, err)
	}
}
