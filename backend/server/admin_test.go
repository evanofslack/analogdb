package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evanofslack/analogdb"
)

type mockAdmin struct{}

func (mockAdmin) Stats(ctx context.Context) (*analogdb.AdminStats, error) {
	return &analogdb.AdminStats{Counts: analogdb.AdminCounts{Posts: 10}}, nil
}

func (mockAdmin) Quality(ctx context.Context) (*analogdb.AdminQuality, error) {
	return &analogdb.AdminQuality{Coverage: analogdb.AdminCoverage{Total: 10}}, nil
}

func (mockAdmin) MissingPosts(ctx context.Context, filter *analogdb.MissingPostsFilter) ([]*analogdb.AdminPost, error) {
	posts := make([]*analogdb.AdminPost, 0, filter.Limit)
	for i := 0; i < filter.Limit; i++ {
		posts = append(posts, &analogdb.AdminPost{ID: 100 - i})
	}
	return posts, nil
}

func (mockAdmin) PostsByIDs(ctx context.Context, ids []int) ([]*analogdb.AdminPost, error) {
	posts := make([]*analogdb.AdminPost, 0, len(ids))
	for _, id := range ids {
		posts = append(posts, &analogdb.AdminPost{ID: id})
	}
	return posts, nil
}

type mockAnalytics struct {
	err error
}

func (m mockAnalytics) Traffic(ctx context.Context, r analogdb.TrafficRange) (*analogdb.Traffic, error) {
	return &analogdb.Traffic{Range: r}, m.err
}

func (m mockAnalytics) Analytics(ctx context.Context, r analogdb.TrafficRange) (*analogdb.Analytics, error) {
	posts := []analogdb.AnalyticsPost{{PostID: 12, ViewCounts: analogdb.ViewCounts{PageViews: 5}}, {PostID: 404, ViewCounts: analogdb.ViewCounts{PageViews: 2}}}
	return &analogdb.Analytics{Range: r, Posts: posts}, m.err
}

func (m mockAnalytics) Audit(ctx context.Context, filter *analogdb.AuditFilter) ([]*analogdb.AuditEntry, error) {
	return []*analogdb.AuditEntry{{StartMs: 5}}, m.err
}

func (m mockAnalytics) Readyz(ctx context.Context) error { return m.err }

type mockVectors struct{}

func (mockVectors) CountObjects(ctx context.Context) (int, error) { return 12, nil }

func mustOpenAdmin(t *testing.T) *Server {
	t.Helper()
	s := mustOpen(t)
	s.config.Auth.Username = "admin"
	s.config.Auth.Password = "secret"
	s.AdminService = mockAdmin{}
	s.VectorCounter = mockVectors{}
	s.AnalyticsService = mockAnalytics{}
	return s
}

func adminRequest(s *Server, path string, authed bool) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	if authed {
		r.SetBasicAuth("admin", "secret")
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)
	return w
}

func TestAdminRequiresAuth(t *testing.T) {
	s := mustOpenAdmin(t)
	defer mustClose(t, s)

	for _, path := range []string{
		"/v1/admin/overview",
		"/v1/admin/quality",
		"/v1/admin/posts/missing?field=camera",
		"/v1/admin/posts/missing?field=caption",
		"/v1/admin/traffic",
		"/v1/admin/analytics",
		"/v1/admin/audit",
	} {
		if w := adminRequest(s, path, false); w.Code != http.StatusUnauthorized {
			t.Errorf("%s: want 401 without auth, got %d", path, w.Code)
		}
		w := adminRequest(s, path, true)
		if w.Code != http.StatusOK {
			t.Errorf("%s: want 200 with auth, got %d: %s", path, w.Code, w.Body.String())
		}
		if got := w.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("%s: want Cache-Control no-store, got %q", path, got)
		}
	}
}

func TestAdminNotOnLegacyRoutes(t *testing.T) {
	s := mustOpenAdmin(t)
	defer mustClose(t, s)

	if w := adminRequest(s, "/admin/overview", true); w.Code != http.StatusNotFound {
		t.Errorf("want 404 on legacy admin route, got %d", w.Code)
	}
}

func TestAdminOverview(t *testing.T) {
	s := mustOpenAdmin(t)
	defer mustClose(t, s)
	s.CacheReadyService = nil

	w := adminRequest(s, "/v1/admin/overview", true)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Status  map[string]string `json:"status"`
		Counts  analogdb.AdminCounts
		Vectors adminVectors `json:"vectors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Status["postgres"] != readyOK || resp.Status["clickhouse"] != readyOK || resp.Status["redis"] != readyDisabled {
		t.Errorf("unexpected status %v", resp.Status)
	}
	if resp.Counts.Posts != 10 {
		t.Errorf("want 10 posts, got %d", resp.Counts.Posts)
	}
	if resp.Vectors.Objects == nil || *resp.Vectors.Objects != 12 || resp.Vectors.Posts != 10 {
		t.Errorf("unexpected vectors %+v", resp.Vectors)
	}

	s.AnalyticsService = nil
	w = adminRequest(s, "/v1/admin/overview", true)
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Status["clickhouse"] != readyDisabled {
		t.Errorf("want clickhouse disabled, got %q", resp.Status["clickhouse"])
	}
}

func TestAdminBadParams(t *testing.T) {
	s := mustOpenAdmin(t)
	defer mustClose(t, s)

	for _, path := range []string{
		"/v1/admin/posts/missing",
		"/v1/admin/posts/missing?field=title",
		"/v1/admin/posts/missing?field=camera&before_id=abc",
		"/v1/admin/traffic?range=1y",
		"/v1/admin/analytics?range=1y",
		"/v1/admin/audit?before=abc",
	} {
		if w := adminRequest(s, path, true); w.Code != http.StatusBadRequest {
			t.Errorf("%s: want 400, got %d", path, w.Code)
		}
	}
}

func TestAdminMissingPostsCursor(t *testing.T) {
	s := mustOpenAdmin(t)
	defer mustClose(t, s)

	w := adminRequest(s, "/v1/admin/posts/missing?field=film&limit=3", true)
	var resp adminMissingResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Posts) != 3 || resp.NextBeforeID == nil || *resp.NextBeforeID != 98 {
		t.Errorf("want 3 posts and next before 98, got %d posts, next %v", len(resp.Posts), resp.NextBeforeID)
	}
}

func TestAdminAnalyticsUnavailable(t *testing.T) {
	s := mustOpenAdmin(t)
	defer mustClose(t, s)

	s.AnalyticsService = nil
	for _, path := range []string{"/v1/admin/traffic", "/v1/admin/analytics", "/v1/admin/audit"} {
		if w := adminRequest(s, path, true); w.Code != http.StatusServiceUnavailable {
			t.Errorf("%s: want 503 when disabled, got %d", path, w.Code)
		}
	}

	s.AnalyticsService = mockAnalytics{err: errors.New("dial tcp: connection refused")}
	for _, path := range []string{"/v1/admin/traffic", "/v1/admin/analytics", "/v1/admin/audit"} {
		if w := adminRequest(s, path, true); w.Code != http.StatusServiceUnavailable {
			t.Errorf("%s: want 503 when down, got %d", path, w.Code)
		}
	}
}

type analyticsAdmin struct {
	mockAdmin
}

func (analyticsAdmin) PostsByIDs(ctx context.Context, ids []int) ([]*analogdb.AdminPost, error) {
	return []*analogdb.AdminPost{{ID: 12, Title: "Harbour", LowURL: "https://images.test/12-low.jpg"}}, nil
}

func TestAdminAnalyticsPosts(t *testing.T) {
	s := mustOpenAdmin(t)
	defer mustClose(t, s)
	s.AdminService = analyticsAdmin{}

	w := adminRequest(s, "/v1/admin/analytics?range=24h", true)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp analogdb.Analytics
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Range != analogdb.TrafficDay || len(resp.Posts) != 2 {
		t.Fatalf("unexpected analytics %+v", resp)
	}
	if resp.Posts[0].Title != "Harbour" || resp.Posts[0].LowURL == "" {
		t.Errorf("want post 12 with title and thumbnail, got %+v", resp.Posts[0])
	}
	if resp.Posts[1].PostID != 404 || resp.Posts[1].Title != "" {
		t.Errorf("want missing post 404 without title, got %+v", resp.Posts[1])
	}
}
