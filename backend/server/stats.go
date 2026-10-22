package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/stats"
	"github.com/go-chi/chi/v5"
)

const (
	maxStatsSeedBodyBytes = 2 << 20
	statsVectorNormSlack  = 0.01
)

var statsThemeKey = regexp.MustCompile(`^t\d{2}$`)

type adminStatsTheme struct {
	Key    string  `json:"key"`
	Name   string  `json:"name"`
	Hidden bool    `json:"hidden"`
	Role   *string `json:"role"`
}

type adminStatsSeedResponse struct {
	Version string            `json:"version"`
	Model   string            `json:"model"`
	Themes  []adminStatsTheme `json:"themes"`
}

func (s *Server) mountAdminStatsHandlers(r chi.Router) {
	r.Put("/stats/seed", s.putStatsSeed)
	r.Get("/stats/seed", s.getStatsSeed)
}

func (s *Server) putStatsSeed(w http.ResponseWriter, r *http.Request) {
	var seed analogdb.StatsSeed
	if err := s.decodeBodyLimit(w, r, &seed, "invalid stats seed", maxStatsSeedBodyBytes); err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := validateStatsSeed(&seed); err != nil {
		s.writeError(w, r, &analogdb.Error{Code: analogdb.ERRUNPROCESSABLE, Message: err.Error()})
		return
	}
	if err := s.StatsService.ReplaceSeed(r.Context(), &seed); err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeStatsSeed(w, r, &seed)
}

func (s *Server) getStatsSeed(w http.ResponseWriter, r *http.Request) {
	seed, err := s.StatsService.FindSeed(r.Context())
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeStatsSeed(w, r, seed)
}

func (s *Server) writeStatsSeed(w http.ResponseWriter, r *http.Request, seed *analogdb.StatsSeed) {
	response := adminStatsSeedResponse{Version: seed.Version, Model: seed.Model, Themes: make([]adminStatsTheme, 0, len(seed.Themes))}
	for _, t := range seed.Themes {
		response.Themes = append(response.Themes, adminStatsTheme{Key: t.Key, Name: t.Name, Hidden: t.Hidden, Role: t.Role})
	}
	if err := encodeResponse(w, r, http.StatusOK, response); err != nil {
		s.writeError(w, r, err)
	}
}

func validateStatsSeed(seed *analogdb.StatsSeed) error {
	if seed.Version == "" || seed.Model == "" {
		return fmt.Errorf("version and model are required")
	}
	if n := len(seed.Themes); n < analogdb.StatsMinThemes || n > analogdb.StatsMaxThemes {
		return fmt.Errorf("want %d to %d themes, got %d", analogdb.StatsMinThemes, analogdb.StatsMaxThemes, n)
	}

	keys := map[string]bool{}
	borders := 0
	for _, t := range seed.Themes {
		if !statsThemeKey.MatchString(t.Key) {
			return fmt.Errorf("theme key %q must look like t00", t.Key)
		}
		if keys[t.Key] {
			return fmt.Errorf("duplicate theme key %q", t.Key)
		}
		keys[t.Key] = true
		if n := utf8.RuneCountInString(t.Name); n < 1 || n > analogdb.StatsMaxThemeLen {
			return fmt.Errorf("theme %s name must be 1 to %d characters", t.Key, analogdb.StatsMaxThemeLen)
		}
		if t.Role != nil {
			if *t.Role != analogdb.StatsRoleBorder {
				return fmt.Errorf("theme %s role %q, want null or %q", t.Key, *t.Role, analogdb.StatsRoleBorder)
			}
			borders++
		}
		if err := validateStatsVector(t.Center); err != nil {
			return fmt.Errorf("theme %s center: %w", t.Key, err)
		}
	}
	if borders > 1 {
		return fmt.Errorf("at most one theme can have role %q", analogdb.StatsRoleBorder)
	}

	prompts := map[string]bool{}
	for _, p := range seed.Prompts {
		if !slices.Contains(analogdb.StatsPromptKeys, p.Key) {
			return fmt.Errorf("unknown prompt %q", p.Key)
		}
		if prompts[p.Key] {
			return fmt.Errorf("duplicate prompt %q", p.Key)
		}
		prompts[p.Key] = true
		if p.Label == "" || p.Text == "" {
			return fmt.Errorf("prompt %s needs a label and text", p.Key)
		}
		if err := validateStatsVector(p.Vector); err != nil {
			return fmt.Errorf("prompt %s vector: %w", p.Key, err)
		}
	}
	for _, key := range analogdb.StatsPromptKeys {
		if !prompts[key] {
			return fmt.Errorf("missing prompt %q", key)
		}
	}
	return nil
}

func validateStatsVector(v []float32) error {
	if len(v) != analogdb.StatsVectorDims {
		return fmt.Errorf("want %d values, got %d", analogdb.StatsVectorDims, len(v))
	}
	var sum float64
	for _, x := range v {
		f := float64(x)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return fmt.Errorf("values must be finite")
		}
		sum += f * f
	}
	if norm := math.Sqrt(sum); math.Abs(norm-1) > statsVectorNormSlack {
		return fmt.Errorf("want unit length, got norm %.3f", norm)
	}
	return nil
}

const (
	statsPath         = "/stats"
	statsWriteTimeout = 5 * time.Minute
)

type StatsRebuildResponse struct {
	ID      int64     `json:"id" example:"12"`
	Created time.Time `json:"created"`
	BuildMs int       `json:"build_ms" example:"24366"`
	Films   int       `json:"films" example:"38"`
	Themes  int       `json:"themes" example:"36"`
}

type adminStatsSnapshotsResponse struct {
	Snapshots []*analogdb.StatsSnapshot `json:"snapshots"`
}

func (s *Server) mountStatsHandlers(r chi.Router) {
	r.With(s.edgeCache(fixedTTL(catalogEdgeTTL))).Get(statsPath, s.getStats)
}

// @Summary Get collection stats
// @Description The latest stats snapshot: how each film looks, film families, subjects, trends, visual themes, popularity and posting time. Rebuilt daily.
// @Tags stats
// @Produce json
// @Success 200 {object} analogdb.Stats
// @Failure 404 {object} ErrorResponse "No snapshot built yet"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /stats [get]
func (s *Server) getStats(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.StatsService.FindLatestSnapshot(r.Context())
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(snapshot.Body); err != nil {
		s.logger.WarnContext(r.Context(), "Fail write stats", "error", err)
	}
}

// @Summary Rebuild stats
// @Description Build a new stats snapshot and serve it from GET /stats. One build runs at a time and a failed build keeps the previous snapshot. Can take a minute (requires authentication)
// @Tags scrape
// @Produce json
// @Success 200 {object} StatsRebuildResponse
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 404 {object} ErrorResponse "No stats seed uploaded"
// @Failure 409 {object} ErrorResponse "A build is already running"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BasicAuth
// @Router /admin/stats/rebuild [post]
func (s *Server) rebuildStats(w http.ResponseWriter, r *http.Request) {
	if !s.statsBuild.TryLock() {
		s.writeError(w, r, &analogdb.Error{Code: analogdb.ERRCONFLICT, Message: "a stats build is already running"})
		return
	}
	defer s.statsBuild.Unlock()

	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(statsWriteTimeout)); err != nil {
		s.logger.WarnContext(r.Context(), "Fail extend write deadline for stats", "error", err)
	}

	start := time.Now()
	snapshot, built, err := s.buildStats(r.Context())
	if err != nil {
		s.stats.statsBuildErrors.Inc()
		s.logger.ErrorContext(r.Context(), "Fail build stats", "error", err)
		s.writeError(w, r, err)
		return
	}
	s.stats.statsBuildSeconds.Observe(time.Since(start).Seconds())

	response := StatsRebuildResponse{
		ID:      snapshot.ID,
		Created: snapshot.Created,
		BuildMs: snapshot.BuildMs,
		Films:   len(built.Films),
		Themes:  len(built.Themes),
	}
	if err := encodeResponse(w, r, http.StatusOK, response); err != nil {
		s.writeError(w, r, err)
	}
}

func (s *Server) buildStats(ctx context.Context) (*analogdb.StatsSnapshot, *analogdb.Stats, error) {
	if s.VectorScanner == nil {
		return nil, nil, &analogdb.Error{Code: analogdb.ERRUNAVAILABLE, Message: "vector database is not enabled"}
	}
	start := time.Now()

	seed, err := s.StatsService.FindSeed(ctx)
	if err != nil {
		return nil, nil, err
	}
	in, err := s.StatsService.LoadStatsInput(ctx, stats.LoadSince, stats.Subjects)
	if err != nil {
		return nil, nil, err
	}

	window := make(map[int]bool, len(in.Posts))
	for _, p := range in.Posts {
		window[p.ID] = true
	}
	vecs := make(map[int][]float32, len(in.Posts))
	err = s.VectorScanner.ScanVectors(ctx, func(postID int, vector []float32) error {
		if _, seen := vecs[postID]; window[postID] && !seen {
			vecs[postID] = vector
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	built, err := stats.Build(in, vecs, seed, time.Now())
	if err != nil {
		return nil, nil, err
	}
	body, err := json.Marshal(built)
	if err != nil {
		return nil, nil, err
	}
	snapshot := &analogdb.StatsSnapshot{
		SeedVersion: seed.Version,
		BuildMs:     int(time.Since(start).Milliseconds()),
		Body:        body,
	}
	if err := s.StatsService.CreateSnapshot(ctx, snapshot); err != nil {
		return nil, nil, err
	}
	s.logger.InfoContext(ctx, "Built stats snapshot", "id", snapshot.ID, "build_ms", snapshot.BuildMs, "size", len(body), "vectors", len(vecs))
	return snapshot, built, nil
}

func (s *Server) getStatsSnapshots(w http.ResponseWriter, r *http.Request) {
	snapshots, err := s.StatsService.FindSnapshots(r.Context())
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := encodeResponse(w, r, http.StatusOK, adminStatsSnapshotsResponse{Snapshots: snapshots}); err != nil {
		s.writeError(w, r, err)
	}
}
