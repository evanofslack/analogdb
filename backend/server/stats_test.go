package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/config"
)

type mockStats struct {
	seed *analogdb.StatsSeed
}

func (m *mockStats) ReplaceSeed(ctx context.Context, seed *analogdb.StatsSeed) error {
	m.seed = seed
	return nil
}

func (m *mockStats) FindSeed(ctx context.Context) (*analogdb.StatsSeed, error) {
	if m.seed == nil {
		return nil, &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "stats seed not found"}
	}
	return m.seed, nil
}

func statsVector(i int) []float32 {
	v := make([]float32, analogdb.StatsVectorDims)
	v[i] = 1
	return v
}

func validStatsSeed() *analogdb.StatsSeed {
	border := analogdb.StatsRoleBorder
	seed := &analogdb.StatsSeed{
		Version: "2026-10-16T00:00:00Z",
		Model:   "ViT-B-16-laion2b_s34b_b88k",
		Themes: []analogdb.StatsTheme{
			{Key: "t00", Name: "Beach and ocean", Center: statsVector(0)},
			{Key: "t01", Name: "Film border", Role: &border, Center: statsVector(1)},
		},
	}
	for i, key := range analogdb.StatsPromptKeys {
		seed.Prompts = append(seed.Prompts, analogdb.StatsPrompt{Key: key, Label: key, Text: "a photo", Vector: statsVector(10 + i)})
	}
	return seed
}

func putSeed(s *Server, seed *analogdb.StatsSeed, creds *pair) *httptest.ResponseRecorder {
	body, _ := json.Marshal(seed)
	r := httptest.NewRequest(http.MethodPut, "/v1/admin/stats/seed", strings.NewReader(string(body)))
	if creds != nil {
		r.SetBasicAuth(creds.username, creds.password)
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)
	return w
}

func TestPutStatsSeed(t *testing.T) {
	s := newRolesServer(t, &config.Config{})
	stats := &mockStats{}
	s.StatsService = stats

	tests := []struct {
		name   string
		mutate func(*analogdb.StatsSeed)
		want   int
	}{
		{name: "valid", mutate: func(*analogdb.StatsSeed) {}, want: http.StatusOK},
		{name: "missing prompt", mutate: func(s *analogdb.StatsSeed) { s.Prompts = s.Prompts[1:] }, want: http.StatusUnprocessableEntity},
		{name: "unknown prompt", mutate: func(s *analogdb.StatsSeed) { s.Prompts[0].Key = "pastel" }, want: http.StatusUnprocessableEntity},
		{name: "wrong vector length", mutate: func(s *analogdb.StatsSeed) { s.Themes[0].Center = s.Themes[0].Center[:511] }, want: http.StatusUnprocessableEntity},
		{name: "non unit vector", mutate: func(s *analogdb.StatsSeed) { s.Prompts[2].Vector[10+2] = 0.5 }, want: http.StatusUnprocessableEntity},
		{name: "duplicate key", mutate: func(s *analogdb.StatsSeed) { s.Themes[1].Key = "t00" }, want: http.StatusUnprocessableEntity},
		{name: "bad key", mutate: func(s *analogdb.StatsSeed) { s.Themes[0].Key = "theme0" }, want: http.StatusUnprocessableEntity},
		{name: "empty name", mutate: func(s *analogdb.StatsSeed) { s.Themes[0].Name = "" }, want: http.StatusUnprocessableEntity},
		{name: "long name", mutate: func(s *analogdb.StatsSeed) { s.Themes[0].Name = strings.Repeat("a", 61) }, want: http.StatusUnprocessableEntity},
		{name: "bad role", mutate: func(s *analogdb.StatsSeed) { role := "frame"; s.Themes[0].Role = &role }, want: http.StatusUnprocessableEntity},
		{name: "two borders", mutate: func(s *analogdb.StatsSeed) { s.Themes[0].Role = s.Themes[1].Role }, want: http.StatusUnprocessableEntity},
		{name: "one theme", mutate: func(s *analogdb.StatsSeed) { s.Themes = s.Themes[:1] }, want: http.StatusUnprocessableEntity},
		{name: "no version", mutate: func(s *analogdb.StatsSeed) { s.Version = "" }, want: http.StatusUnprocessableEntity},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats.seed = nil
			seed := validStatsSeed()
			tt.mutate(seed)
			w := putSeed(s, seed, &adminPair)
			if w.Code != tt.want {
				t.Fatalf("want %d, got %d: %s", tt.want, w.Code, w.Body.String())
			}
			if tt.want != http.StatusOK && stats.seed != nil {
				t.Error("want nothing written for an invalid seed")
			}
		})
	}
}

func TestPutStatsSeedFullSize(t *testing.T) {
	s := newRolesServer(t, &config.Config{})
	s.StatsService = &mockStats{}

	seed := validStatsSeed()
	seed.Themes = seed.Themes[:0]
	for i := 0; i < analogdb.StatsMaxThemes; i++ {
		v := make([]float32, analogdb.StatsVectorDims)
		for j := range v {
			v[j] = 0.044194173
		}
		seed.Themes = append(seed.Themes, analogdb.StatsTheme{Key: "t" + string(rune('0'+i/10)) + string(rune('0'+i%10)), Name: "theme", Center: v})
	}
	if w := putSeed(s, seed, &adminPair); w.Code != http.StatusOK {
		t.Fatalf("want a full size seed accepted, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStatsSeedRoles(t *testing.T) {
	s := newRolesServer(t, &config.Config{})
	s.StatsService = &mockStats{seed: validStatsSeed()}

	if w := putSeed(s, validStatsSeed(), nil); w.Code != http.StatusUnauthorized {
		t.Errorf("anonymous put: want 401, got %d", w.Code)
	}
	if w := putSeed(s, validStatsSeed(), &scraperPair); w.Code != http.StatusForbidden {
		t.Errorf("scraper put: want 403, got %d", w.Code)
	}
	if w := roleRequest(s, http.MethodGet, "/v1/admin/stats/seed", nil, ""); w.Code != http.StatusUnauthorized {
		t.Errorf("anonymous get: want 401, got %d", w.Code)
	}
}

func TestGetStatsSeed(t *testing.T) {
	s := newRolesServer(t, &config.Config{})
	stats := &mockStats{}
	s.StatsService = stats

	if w := roleRequest(s, http.MethodGet, "/v1/admin/stats/seed", &adminPair, ""); w.Code != http.StatusNotFound {
		t.Errorf("want 404 before a seed, got %d", w.Code)
	}

	stats.seed = validStatsSeed()
	w := roleRequest(s, http.MethodGet, "/v1/admin/stats/seed", &adminPair, "")
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "center") || strings.Contains(w.Body.String(), "vector") {
		t.Errorf("want no vectors in the response, got %s", w.Body.String())
	}
	var got adminStatsSeedResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Model != stats.seed.Model || len(got.Themes) != 2 || got.Themes[1].Role == nil {
		t.Errorf("want model and 2 themes with the border role, got %+v", got)
	}
}
