package server

import (
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"
)

type keywordsUpdatedResponse struct {
	Ids []int `json:"ids"`
}

type CaptionsMissingResponse struct {
	Ids []int `json:"ids" example:"1,2,3"`
}

type VectorsMissingResponse struct {
	Ids   []int `json:"ids" example:"1,2,3"`
	Extra int   `json:"extra" example:"0"`
}

const (
	scrapePath          = "/scrape"
	keywordsUpdatedPath = scrapePath + "/keywords/updated"
	captionsMissingPath = scrapePath + "/captions/missing"
	vectorsMissingPath  = scrapePath + "/vectors/missing"
)

func (s *Server) mountScrapeHandlers(r chi.Router) {
	r.Route(keywordsUpdatedPath, func(r chi.Router) {
		r.With(s.require(roleScraper)).Get("/", s.getKeywordUpdatedPosts)
	})
	r.Route(captionsMissingPath, func(r chi.Router) {
		r.With(s.require(roleScraper)).Get("/", s.getCaptionMissingPosts)
	})
	r.Route(vectorsMissingPath, func(r chi.Router) {
		r.With(s.require(roleScraper)).Get("/", s.getVectorMissingPosts)
	})
}

func (s *Server) getKeywordUpdatedPosts(w http.ResponseWriter, r *http.Request) {
	ids, err := s.ScrapeService.KeywordUpdatedPostIDs(r.Context())
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	response := keywordsUpdatedResponse{
		Ids: ids,
	}
	if err := encodeResponse(w, r, http.StatusOK, response); err != nil {
		s.writeError(w, r, err)
	}
}

// @Summary List posts missing a caption
// @Description List ids of posts with no caption, or with a caption of another version when version is set (requires authentication)
// @Tags scrape
// @Produce json
// @Param version query string false "Caption version the posts should have"
// @Success 200 {object} CaptionsMissingResponse
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BasicAuth
// @Router /scrape/captions/missing [get]
func (s *Server) getCaptionMissingPosts(w http.ResponseWriter, r *http.Request) {
	var version *string
	if v := r.URL.Query().Get("version"); v != "" {
		version = &v
	}
	ids, err := s.ScrapeService.CaptionMissingPostIDs(r.Context(), version)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	response := CaptionsMissingResponse{
		Ids: ids,
	}
	if err := encodeResponse(w, r, http.StatusOK, response); err != nil {
		s.writeError(w, r, err)
	}
}

// @Summary List posts missing a vector
// @Description List ids of posts with no object in the vector database, oldest first, and how many objects belong to deleted posts (requires authentication)
// @Tags scrape
// @Produce json
// @Success 200 {object} VectorsMissingResponse
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Failure 503 {object} ErrorResponse "Vector database unavailable"
// @Security BasicAuth
// @Router /scrape/vectors/missing [get]
func (s *Server) getVectorMissingPosts(w http.ResponseWriter, r *http.Request) {
	missing, extra, err := s.missingVectors(r.Context())
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	ids := slices.Clone(missing)
	slices.Reverse(ids)
	response := VectorsMissingResponse{
		Ids:   ids,
		Extra: extra,
	}
	if err := encodeResponse(w, r, http.StatusOK, response); err != nil {
		s.writeError(w, r, err)
	}
}
