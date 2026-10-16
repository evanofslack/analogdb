package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/config"
)

type mockStats struct {
	seed      *analogdb.StatsSeed
	snapshots []*analogdb.StatsSnapshot
	// block, when set, holds LoadStatsInput until it is closed
	block   chan struct{}
	started chan struct{}
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

func (m *mockStats) LoadStatsInput(ctx context.Context, since time.Time, subjects []string) (*analogdb.StatsInput, error) {
	if m.started != nil {
		close(m.started)
	}
	if m.block != nil {
		<-m.block
	}
	in := &analogdb.StatsInput{}
	for i := 1; i <= 3; i++ {
		in.Posts = append(in.Posts, analogdb.StatsInputPost{ID: i, Author: "a", Time: since.Unix() + int64(i), Popularity: 0.5})
	}
	return in, nil
}

func (m *mockStats) CreateSnapshot(ctx context.Context, snapshot *analogdb.StatsSnapshot) error {
	snapshot.ID = int64(len(m.snapshots) + 1)
	snapshot.Created = time.Now()
	m.snapshots = append(m.snapshots, snapshot)
	return nil
}

func (m *mockStats) FindLatestSnapshot(ctx context.Context) (*analogdb.StatsSnapshot, error) {
	if len(m.snapshots) == 0 {
		return nil, &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "stats not built yet"}
	}
	return m.snapshots[len(m.snapshots)-1], nil
}

func (m *mockStats) FindSnapshots(ctx context.Context) ([]*analogdb.StatsSnapshot, error) {
	return m.snapshots, nil
}

type mockScanner struct{}

func (mockScanner) ScanVectors(ctx context.Context, fn func(postID int, vector []float32) error) error {
	for id := 1; id <= 3; id++ {
		if err := fn(id, statsVector(id)); err != nil {
			return err
		}
	}
	return fn(99, statsVector(0))
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

func TestGetStats(t *testing.T) {
	s := newRolesServer(t, &config.Config{})
	stats := &mockStats{seed: validStatsSeed()}
	s.StatsService = stats
	s.VectorScanner = mockScanner{}

	if w := roleRequest(s, http.MethodGet, "/v1/stats", nil, ""); w.Code != http.StatusNotFound {
		t.Fatalf("want 404 before a build, got %d", w.Code)
	}

	w := roleRequest(s, http.MethodPost, "/v1/admin/stats/rebuild", &scraperPair, "")
	if w.Code != http.StatusOK {
		t.Fatalf("rebuild: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var rebuilt StatsRebuildResponse
	if err := json.Unmarshal(w.Body.Bytes(), &rebuilt); err != nil {
		t.Fatal(err)
	}
	if rebuilt.ID != 1 || rebuilt.Themes == 0 {
		t.Errorf("want snapshot 1 with themes, got %+v", rebuilt)
	}

	w = roleRequest(s, http.MethodGet, "/v1/stats", nil, "")
	if w.Code != http.StatusOK {
		t.Fatalf("want 200 after a build, got %d", w.Code)
	}
	if cc := w.Header().Get("Cache-Control"); !strings.Contains(cc, "s-maxage=3600") {
		t.Errorf("want edge cacheable, got %q", cc)
	}
	var got analogdb.Stats
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Totals.Photos != 3 || got.SeedVersion != stats.seed.Version {
		t.Errorf("want 3 photos and the seed version, got %+v", got.Totals)
	}

	w = roleRequest(s, http.MethodGet, "/v1/admin/stats/snapshots", &adminPair, "")
	if w.Code != http.StatusOK || strings.Contains(w.Body.String(), "themes") {
		t.Errorf("want snapshots without bodies, got %d %s", w.Code, w.Body.String())
	}
}

func TestRebuildStatsRoles(t *testing.T) {
	s := newRolesServer(t, &config.Config{})
	s.StatsService = &mockStats{seed: validStatsSeed()}
	s.VectorScanner = mockScanner{}

	tests := []struct {
		name  string
		creds *pair
		want  int
	}{
		{name: "anonymous", want: http.StatusUnauthorized},
		{name: "web", creds: &webPair, want: http.StatusForbidden},
		{name: "scraper", creds: &scraperPair, want: http.StatusOK},
		{name: "admin", creds: &adminPair, want: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if w := roleRequest(s, http.MethodPost, "/v1/admin/stats/rebuild", tt.creds, ""); w.Code != tt.want {
				t.Errorf("want %d, got %d", tt.want, w.Code)
			}
		})
	}
	if w := roleRequest(s, http.MethodGet, "/v1/admin/stats/snapshots", &scraperPair, ""); w.Code != http.StatusForbidden {
		t.Errorf("scraper snapshots: want 403, got %d", w.Code)
	}
}

func TestRebuildStatsBusy(t *testing.T) {
	s := newRolesServer(t, &config.Config{})
	stats := &mockStats{seed: validStatsSeed(), block: make(chan struct{}), started: make(chan struct{})}
	s.StatsService = stats
	s.VectorScanner = mockScanner{}

	first := make(chan int)
	go func() {
		first <- roleRequest(s, http.MethodPost, "/v1/admin/stats/rebuild", &scraperPair, "").Code
	}()
	<-stats.started
	if w := roleRequest(s, http.MethodPost, "/v1/admin/stats/rebuild", &scraperPair, ""); w.Code != http.StatusConflict {
		t.Errorf("want 409 while a build runs, got %d", w.Code)
	}
	close(stats.block)
	if code := <-first; code != http.StatusOK {
		t.Errorf("want the first build to finish, got %d", code)
	}
	if len(stats.snapshots) != 1 {
		t.Errorf("want one snapshot, got %d", len(stats.snapshots))
	}
}

func TestRebuildStatsKeepsPrevious(t *testing.T) {
	s := newRolesServer(t, &config.Config{})
	stats := &mockStats{snapshots: []*analogdb.StatsSnapshot{{ID: 1, Body: []byte(`{"version":1}`)}}}
	s.StatsService = stats
	s.VectorScanner = mockScanner{}

	if w := roleRequest(s, http.MethodPost, "/v1/admin/stats/rebuild", &adminPair, ""); w.Code != http.StatusNotFound {
		t.Errorf("want the missing seed reported, got %d", w.Code)
	}
	if len(stats.snapshots) != 1 {
		t.Errorf("want nothing stored on failure, got %d snapshots", len(stats.snapshots))
	}
}
