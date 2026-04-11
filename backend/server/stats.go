package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/go-chi/chi/v5"
)

type StatsOverviewResponse struct {
	Data        analogdb.StatsOverview `json:"data"`
	GeneratedAt time.Time              `json:"generated_at"`
}

type StatsPeriodsResponse struct {
	Data []analogdb.StatsPeriod `json:"data"`
	Meta analogdb.StatsMeta     `json:"meta"`
}

type StatsFilmsResponse struct {
	Data []analogdb.StatsFilm `json:"data"`
	Meta analogdb.StatsMeta   `json:"meta"`
}

type StatsCamerasResponse struct {
	Data []analogdb.StatsCamera `json:"data"`
	Meta analogdb.StatsMeta     `json:"meta"`
}

type StatsColorsResponse struct {
	Data []analogdb.StatsColor `json:"data"`
	Meta analogdb.StatsMeta    `json:"meta"`
}

const (
	statsPath         = "/stats"
	defaultStatsLimit = 20
	maxStatsLimit     = 100
)

func (s *Server) mountStatsHandlers(r chi.Router) {
	r.Route(statsPath, func(r chi.Router) {
		r.Get("/overview", s.getStatsOverview)
		r.Get("/posts/over-time", s.getStatsPostsOverTime)
		r.Get("/films", s.getStatsFilms)
		r.Get("/cameras", s.getStatsCameras)
		r.Get("/colors", s.getStatsColors)
	})
}

// @Summary Get global statistics overview
// @Description Returns aggregate counts and score averages across all posts
// @Tags stats
// @Accept json
// @Produce json
// @Param start query int false "Filter by start time (unix timestamp)"
// @Param end   query int false "Filter by end time (unix timestamp)"
// @Success 200 {object} StatsOverviewResponse
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Router /stats/overview [get]
func (s *Server) getStatsOverview(w http.ResponseWriter, r *http.Request) {
	filter, err := parseToStatsFilter(r)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	overview, err := s.StatsService.GetOverview(r.Context(), filter)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	resp := StatsOverviewResponse{
		Data:        *overview,
		GeneratedAt: time.Now().UTC(),
	}
	if err := encodeResponse(w, r, http.StatusOK, resp); err != nil {
		s.writeError(w, r, err)
	}
}

// @Summary Get post counts and scores over time
// @Description Returns post counts and average scores grouped by time period
// @Tags stats
// @Accept json
// @Produce json
// @Param granularity query string false "Time grouping" Enums(month, week, year) default(month)
// @Param start       query int    false "Filter by start time (unix timestamp)"
// @Param end         query int    false "Filter by end time (unix timestamp)"
// @Success 200 {object} StatsPeriodsResponse
// @Failure 400 {object} analogdb.Error "Invalid query parameters"
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Router /stats/posts/over-time [get]
func (s *Server) getStatsPostsOverTime(w http.ResponseWriter, r *http.Request) {
	filter, err := parseToStatsFilter(r)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	periods, err := s.StatsService.GetPostsOverTime(r.Context(), filter)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	resp := StatsPeriodsResponse{
		Meta: analogdb.StatsMeta{
			Total:       len(periods),
			GeneratedAt: time.Now().UTC(),
		},
	}
	for _, p := range periods {
		resp.Data = append(resp.Data, *p)
	}
	if err := encodeResponse(w, r, http.StatusOK, resp); err != nil {
		s.writeError(w, r, err)
	}
}

// @Summary Get film stocks ranked by metric
// @Description Returns film stocks ranked by post count or average score
// @Tags stats
// @Accept json
// @Produce json
// @Param metric query string false "Ranking metric" Enums(score, count) default(count)
// @Param limit  query int    false "Max results (max 100)" default(20)
// @Param start  query int    false "Filter by start time (unix timestamp)"
// @Param end    query int    false "Filter by end time (unix timestamp)"
// @Success 200 {object} StatsFilmsResponse
// @Failure 400 {object} analogdb.Error "Invalid query parameters"
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Router /stats/films [get]
func (s *Server) getStatsFilms(w http.ResponseWriter, r *http.Request) {
	filter, err := parseToStatsFilter(r)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	films, err := s.StatsService.GetFilmStats(r.Context(), filter)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	resp := StatsFilmsResponse{
		Meta: analogdb.StatsMeta{
			Total:       len(films),
			GeneratedAt: time.Now().UTC(),
		},
	}
	for _, f := range films {
		resp.Data = append(resp.Data, *f)
	}
	if err := encodeResponse(w, r, http.StatusOK, resp); err != nil {
		s.writeError(w, r, err)
	}
}

// @Summary Get cameras ranked by metric
// @Description Returns cameras ranked by post count or average score
// @Tags stats
// @Accept json
// @Produce json
// @Param metric query string false "Ranking metric" Enums(score, count) default(count)
// @Param limit  query int    false "Max results (max 100)" default(20)
// @Param start  query int    false "Filter by start time (unix timestamp)"
// @Param end    query int    false "Filter by end time (unix timestamp)"
// @Success 200 {object} StatsCamerasResponse
// @Failure 400 {object} analogdb.Error "Invalid query parameters"
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Router /stats/cameras [get]
func (s *Server) getStatsCameras(w http.ResponseWriter, r *http.Request) {
	filter, err := parseToStatsFilter(r)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	cameras, err := s.StatsService.GetCameraStats(r.Context(), filter)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	resp := StatsCamerasResponse{
		Meta: analogdb.StatsMeta{
			Total:       len(cameras),
			GeneratedAt: time.Now().UTC(),
		},
	}
	for _, c := range cameras {
		resp.Data = append(resp.Data, *c)
	}
	if err := encodeResponse(w, r, http.StatusOK, resp); err != nil {
		s.writeError(w, r, err)
	}
}

// @Summary Get dominant colors ranked by occurrence
// @Description Returns the most frequently dominant colors across all posts
// @Tags stats
// @Accept json
// @Produce json
// @Param limit query int false "Max results (max 100)" default(20)
// @Param start query int false "Filter by start time (unix timestamp)"
// @Param end   query int false "Filter by end time (unix timestamp)"
// @Success 200 {object} StatsColorsResponse
// @Failure 400 {object} analogdb.Error "Invalid query parameters"
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Router /stats/colors [get]
func (s *Server) getStatsColors(w http.ResponseWriter, r *http.Request) {
	filter, err := parseToStatsFilter(r)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	colors, err := s.StatsService.GetColorStats(r.Context(), filter)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	resp := StatsColorsResponse{
		Meta: analogdb.StatsMeta{
			Total:       len(colors),
			GeneratedAt: time.Now().UTC(),
		},
	}
	for _, c := range colors {
		resp.Data = append(resp.Data, *c)
	}
	if err := encodeResponse(w, r, http.StatusOK, resp); err != nil {
		s.writeError(w, r, err)
	}
}

func parseToStatsFilter(r *http.Request) (*analogdb.StatsFilter, error) {
	filter := &analogdb.StatsFilter{}
	values := r.URL.Query()

	n := defaultStatsLimit
	if raw := values.Get("limit"); raw != "" {
		parsed, err := stringToInt(raw)
		if err != nil {
			return nil, err
		}
		if parsed > maxStatsLimit {
			parsed = maxStatsLimit
		}
		n = parsed
	}
	filter.Limit = &n

	if raw := values.Get("start"); raw != "" {
		n, err := stringToInt(raw)
		if err != nil {
			return nil, err
		}
		filter.Start = &n
	}

	if raw := values.Get("end"); raw != "" {
		n, err := stringToInt(raw)
		if err != nil {
			return nil, err
		}
		filter.End = &n
	}

	if raw := values.Get("granularity"); raw != "" {
		valid := map[string]bool{"month": true, "week": true, "year": true}
		if !valid[raw] {
			return nil, fmt.Errorf("invalid granularity %q, valid options: month, week, year", raw)
		}
		filter.Granularity = &raw
	}

	if raw := values.Get("metric"); raw != "" {
		valid := map[string]bool{"score": true, "count": true}
		if !valid[raw] {
			return nil, fmt.Errorf("invalid metric %q, valid options: score, count", raw)
		}
		filter.Metric = &raw
	}

	return filter, nil
}
