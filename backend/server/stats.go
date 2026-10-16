package server

import (
	"fmt"
	"math"
	"net/http"
	"regexp"
	"slices"
	"unicode/utf8"

	"github.com/evanofslack/analogdb"
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
