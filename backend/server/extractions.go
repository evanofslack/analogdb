package server

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/evanofslack/analogdb"
	"github.com/go-chi/chi/v5"
)

// Extraction routes sit under /admin with the other internal routes, but are in
// the OpenAPI spec because Dagster uses them through the generated client.

const (
	maxExtractionsBatch     = 500
	maxExtractionsBodyBytes = 8 << 20
	defaultExtractionsLimit = 100
	maxExtractionsLimit     = 500
)

type ExtractionsRequest struct {
	Extractions []*analogdb.PostExtraction `json:"extractions"`
}

type UpsertExtractionsResponse struct {
	Written int   `json:"written" example:"499"`
	Skipped []int `json:"skipped" example:"1234"`
}

type ExtractionsResponse struct {
	Extractions  []*analogdb.PostExtraction `json:"extractions"`
	NextBeforeID *int                       `json:"next_before_id" example:"1200"`
}

func (s *Server) mountExtractionHandlers(r chi.Router) {
	r.Post("/extractions", s.upsertExtractions)
	r.Get("/extractions", s.getExtractions)
}

// @Summary Store post metadata extractions
// @Description Create or replace the stored metadata extraction of each post (requires authentication). Posts that don't exist are skipped.
// @Tags extractions
// @Accept json
// @Produce json
// @Param extractions body ExtractionsRequest true "extractions to store, at most 500"
// @Success 200 {object} UpsertExtractionsResponse
// @Failure 400 {object} analogdb.Error "Invalid extractions"
// @Failure 401 {object} analogdb.Error "Unauthorized"
// @Failure 403 {object} analogdb.Error "Forbidden"
// @Failure 422 {object} analogdb.Error "Unprocessable entity"
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Security BasicAuth
// @Router /admin/extractions [post]
func (s *Server) upsertExtractions(w http.ResponseWriter, r *http.Request) {
	var body ExtractionsRequest
	if err := s.decodeBodyLimit(w, r, &body, "parse extractions from request body", maxExtractionsBodyBytes); err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := validateExtractions(body.Extractions); err != nil {
		s.writeError(w, r, err)
		return
	}

	written, skipped, err := s.ExtractionService.UpsertExtractions(r.Context(), body.Extractions)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	response := UpsertExtractionsResponse{Written: written, Skipped: skipped}
	if err := encodeResponse(w, r, http.StatusOK, response); err != nil {
		s.writeError(w, r, err)
	}
}

func validateExtractions(extractions []*analogdb.PostExtraction) error {
	if len(extractions) == 0 {
		return badRequest("no extractions")
	}
	if len(extractions) > maxExtractionsBatch {
		return badRequest("too many extractions, max %d per request", maxExtractionsBatch)
	}
	for i, e := range extractions {
		if e == nil || e.PostID <= 0 {
			return badRequest("extraction %d: invalid post_id", i)
		}
		if e.ExtractorVersion == "" || e.Model == "" || e.InputHash == "" {
			return badRequest("extraction %d: extractor_version, model and input_hash are required", i)
		}
		if !jsonStartsWith(e.Raw, '{') {
			return badRequest("extraction %d: raw must be a JSON object", i)
		}
		if len(e.Unmatched) > 0 && !jsonStartsWith(e.Unmatched, '[') {
			return badRequest("extraction %d: unmatched must be a JSON array", i)
		}
	}
	return nil
}

func jsonStartsWith(raw json.RawMessage, first byte) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && trimmed[0] == first
}

// @Summary List post metadata extractions
// @Description List stored metadata extractions, newest post first (requires authentication)
// @Tags extractions
// @Produce json
// @Param has_unmatched query bool false "Only extractions with or without unmatched mentions"
// @Param kind query string false "Only extractions with an unmatched mention of this kind" Enums(camera, film)
// @Param key query string false "Only extractions with an unmatched mention with this normalized key"
// @Param full query bool false "Include the input text and raw LLM output"
// @Param before_id query int false "Only post ids below this, for paging"
// @Param limit query int false "Number of results, at most 500" default(100)
// @Success 200 {object} ExtractionsResponse
// @Failure 400 {object} analogdb.Error "Invalid query parameters"
// @Failure 401 {object} analogdb.Error "Unauthorized"
// @Failure 403 {object} analogdb.Error "Forbidden"
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Security BasicAuth
// @Router /admin/extractions [get]
func (s *Server) getExtractions(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	filter := &analogdb.ExtractionFilter{Limit: defaultExtractionsLimit}
	if str := query.Get("limit"); str != "" {
		val, err := stringToInt(str)
		if err != nil {
			s.writeError(w, r, err)
			return
		}
		filter.Limit = clampLimit(val, defaultExtractionsLimit, 1, maxExtractionsLimit)
	}
	if str := query.Get("before_id"); str != "" {
		val, err := stringToInt(str)
		if err != nil {
			s.writeError(w, r, err)
			return
		}
		filter.BeforeID = &val
	}
	if str := query.Get("has_unmatched"); str != "" {
		val, err := stringToBool(str)
		if err != nil {
			s.writeError(w, r, err)
			return
		}
		filter.HasUnmatched = &val
	}
	if str := query.Get("full"); str != "" {
		val, err := stringToBool(str)
		if err != nil {
			s.writeError(w, r, err)
			return
		}
		filter.Full = val
	}
	if kind := query.Get("kind"); kind != "" {
		if kind != "camera" && kind != "film" {
			s.writeError(w, r, badRequest("invalid kind %q, want camera or film", kind))
			return
		}
		filter.Kind = &kind
	}
	if key := query.Get("key"); key != "" {
		filter.Key = &key
	}

	extractions, err := s.ExtractionService.FindExtractions(r.Context(), filter)
	if err != nil {
		s.writeError(w, r, err)
		return
	}

	response := ExtractionsResponse{Extractions: extractions}
	if len(extractions) == filter.Limit {
		next := extractions[len(extractions)-1].PostID
		response.NextBeforeID = &next
	}
	if err := encodeResponse(w, r, http.StatusOK, response); err != nil {
		s.writeError(w, r, err)
	}
}
