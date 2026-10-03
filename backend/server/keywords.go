package server

import (
	"net/http"

	"github.com/evanofslack/analogdb"
	"github.com/go-chi/chi/v5"
)

const (
	keywordsPath        = "/keywords"
	defaultKeywordLimit = 50
	maxKeywordLimit     = 500
	minKeywordDays      = 1
	maxKeywordDays      = 90
)

type KeywordsResponse struct {
	Keywords []analogdb.KeywordSummary `json:"keywords"`
}

func (s *Server) mountKeywordHandlers(r chi.Router) {
	r.Route(keywordsPath, func(r chi.Router) {
		r.Get("/summary", s.getSummary)
	})
}

// @Summary Get keyword summary
// @Description Most common keywords with their post counts, optionally only from posts created in the last days
// @Tags keywords
// @Produce json
// @Param page_size query int false "Number of keywords to return" default(50)
// @Param days query int false "Only count keywords of posts created in the last days, 1 to 90"
// @Success 200 {object} KeywordsResponse
// @Failure 400 {object} analogdb.Error "Invalid request"
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Router /keywords/summary [get]
func (s *Server) getSummary(w http.ResponseWriter, r *http.Request) {
	limit := defaultKeywordLimit
	var err error

	if strLimit := r.URL.Query().Get("page_size"); strLimit != "" {
		limit, err = stringToInt(strLimit)
		if err != nil {
			s.writeError(w, r, err)
			return
		}
		limit = clampLimit(limit, defaultKeywordLimit, 1, maxKeywordLimit)
	}

	filter := &analogdb.KeywordFilter{Limit: &limit}
	if strDays := r.URL.Query().Get("days"); strDays != "" {
		days, err := stringToInt(strDays)
		if err != nil {
			s.writeError(w, r, err)
			return
		}
		if days < minKeywordDays || days > maxKeywordDays {
			s.writeError(w, r, badRequest("days must be between %d and %d", minKeywordDays, maxKeywordDays))
			return
		}
		filter.Days = &days
	}

	keywords, err := s.KeywordService.GetKeywordSummary(r.Context(), filter)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	response := KeywordsResponse{
		Keywords: *keywords,
	}
	if err := encodeResponse(w, r, http.StatusOK, response); err != nil {
		s.writeError(w, r, err)
	}
}
