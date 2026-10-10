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

type mockReports struct {
	created      []*analogdb.CreateReport
	filter       *analogdb.ReportFilter
	report       *analogdb.Report
	resolved     []int
	resolvedPost []int
}

func (m *mockReports) CreateReport(ctx context.Context, report *analogdb.CreateReport) (int, error) {
	if report.PostID == 9999 {
		return 0, &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "post not found"}
	}
	m.created = append(m.created, report)
	return len(m.created), nil
}

func (m *mockReports) FindReports(ctx context.Context, filter *analogdb.ReportFilter) ([]*analogdb.Report, error) {
	m.filter = filter
	reports := make([]*analogdb.Report, 0, filter.Limit)
	for i := 0; i < filter.Limit; i++ {
		reports = append(reports, &analogdb.Report{ID: 100 - i})
	}
	return reports, nil
}

func (m *mockReports) FindReportByID(ctx context.Context, id int) (*analogdb.Report, error) {
	if m.report == nil || m.report.ID != id {
		return nil, &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "report not found"}
	}
	return m.report, nil
}

func (m *mockReports) ResolveReport(ctx context.Context, id int) error {
	m.resolved = append(m.resolved, id)
	return nil
}

func (m *mockReports) ResolvePostReports(ctx context.Context, postID int) error {
	m.resolvedPost = append(m.resolvedPost, postID)
	return nil
}

func (m *mockReports) RemovedPermalinks(ctx context.Context) ([]string, error) {
	return []string{"/r/analog/comments/abc/"}, nil
}

type mockDeletePosts struct {
	analogdb.PostService
	missing bool
	deleted map[int]string
}

func (m *mockDeletePosts) DeletePost(ctx context.Context, id int, reason string) error {
	if m.missing {
		return &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "post not found"}
	}
	if m.deleted == nil {
		m.deleted = map[int]string{}
	}
	m.deleted[id] = reason
	return nil
}

type mockDeleteVectors struct {
	analogdb.SimilarityService
	deleted []int
}

func (m *mockDeleteVectors) DeletePost(ctx context.Context, id int) error {
	m.deleted = append(m.deleted, id)
	// a post with no vector is fine for callers
	return &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "post not found"}
}

func postReport(s *Server, path, body string, creds *pair, remote string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	if creds != nil {
		r.SetBasicAuth(creds.username, creds.password)
	}
	if remote != "" {
		r.RemoteAddr = remote
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)
	return w
}

func TestCreateReport(t *testing.T) {
	s := newRolesServer(t, &config.Config{})
	reports := &mockReports{}
	s.ReportService = reports

	tests := []struct {
		name string
		path string
		body string
		want int
	}{
		{name: "takedown with email", path: "/v1/post/1/report", body: `{"reason": "takedown", "message": "  mine  ", "email": "me@example.com"}`, want: http.StatusCreated},
		{name: "reason only", path: "/v1/post/1/report", body: `{"reason": "not_film"}`, want: http.StatusCreated},
		{name: "bad reason", path: "/v1/post/1/report", body: `{"reason": "spam"}`, want: http.StatusBadRequest},
		{name: "no reason", path: "/v1/post/1/report", body: `{}`, want: http.StatusBadRequest},
		{name: "long message", path: "/v1/post/1/report", body: `{"reason": "other", "message": "` + strings.Repeat("a", maxReportMessage+1) + `"}`, want: http.StatusBadRequest},
		{name: "bad email", path: "/v1/post/1/report", body: `{"reason": "other", "email": "not an email"}`, want: http.StatusBadRequest},
		{name: "missing post", path: "/v1/post/9999/report", body: `{"reason": "other"}`, want: http.StatusNotFound},
		{name: "bad id", path: "/v1/post/abc/report", body: `{"reason": "other"}`, want: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if w := postReport(s, tt.path, tt.body, nil, ""); w.Code != tt.want {
				t.Errorf("want %d, got %d: %s", tt.want, w.Code, w.Body.String())
			}
		})
	}

	if len(reports.created) != 2 {
		t.Fatalf("want 2 reports created, got %d", len(reports.created))
	}
	if got := reports.created[0]; got.PostID != 1 || got.Message != "mine" || got.Email != "me@example.com" {
		t.Errorf("want trimmed report for post 1, got %+v", got)
	}
}

func TestCreateReportRateLimit(t *testing.T) {
	cfg := &config.Config{}
	cfg.App.RateLimitEnabled = true
	s := newRolesServer(t, cfg)
	s.ReportService = &mockReports{}

	body := `{"reason": "other"}`
	for i := 0; i < reportRateLimit; i++ {
		if w := postReport(s, "/v1/post/1/report", body, nil, "1.1.1.1:1234"); w.Code != http.StatusCreated {
			t.Fatalf("report %d: want 201, got %d", i, w.Code)
		}
	}
	if w := postReport(s, "/v1/post/1/report", body, nil, "1.1.1.1:1234"); w.Code != http.StatusTooManyRequests {
		t.Errorf("want 429 after %d reports, got %d", reportRateLimit, w.Code)
	}
	if w := postReport(s, "/v1/post/1/report", body, nil, "2.2.2.2:1234"); w.Code != http.StatusCreated {
		t.Errorf("want another client allowed, got %d", w.Code)
	}
	if w := postReport(s, "/v1/post/1/report", body, &adminPair, "1.1.1.1:1234"); w.Code != http.StatusCreated {
		t.Errorf("want admin never limited, got %d", w.Code)
	}
	if w := postReport(s, "/v1/post/1/report", body, &webPair, "1.1.1.1:1234"); w.Code != http.StatusCreated {
		t.Errorf("want web left to its own per visitor limit, got %d", w.Code)
	}
}

func TestAdminReportRoles(t *testing.T) {
	s := newRolesServer(t, &config.Config{})
	s.ReportService = &mockReports{}

	tests := []struct {
		name   string
		method string
		path   string
		creds  *pair
		want   int
	}{
		{name: "anonymous list", method: http.MethodGet, path: "/v1/admin/reports", want: http.StatusUnauthorized},
		{name: "web list", method: http.MethodGet, path: "/v1/admin/reports", creds: &webPair, want: http.StatusForbidden},
		{name: "scraper list", method: http.MethodGet, path: "/v1/admin/reports", creds: &scraperPair, want: http.StatusForbidden},
		{name: "admin list", method: http.MethodGet, path: "/v1/admin/reports", creds: &adminPair, want: http.StatusOK},
		{name: "scraper resolve", method: http.MethodPost, path: "/v1/admin/reports/1/resolve", creds: &scraperPair, want: http.StatusForbidden},
		{name: "anonymous removed", method: http.MethodGet, path: "/v1/admin/removed/permalinks", want: http.StatusUnauthorized},
		{name: "web removed", method: http.MethodGet, path: "/v1/admin/removed/permalinks", creds: &webPair, want: http.StatusForbidden},
		{name: "scraper removed", method: http.MethodGet, path: "/v1/admin/removed/permalinks", creds: &scraperPair, want: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if w := roleRequest(s, tt.method, tt.path, tt.creds, ""); w.Code != tt.want {
				t.Errorf("want %d, got %d", tt.want, w.Code)
			}
		})
	}
}

func TestAdminReports(t *testing.T) {
	s := newRolesServer(t, &config.Config{})
	reports := &mockReports{}
	s.ReportService = reports

	w := roleRequest(s, http.MethodGet, "/v1/admin/reports?status=all&limit=2&before=50", &adminPair, "")
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var got adminReportsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if reports.filter.Status != analogdb.ReportStatusAll || *reports.filter.BeforeID != 50 || reports.filter.Limit != 2 {
		t.Errorf("unexpected filter %+v", reports.filter)
	}
	if got.NextBefore == nil || *got.NextBefore != 99 {
		t.Errorf("want next before 99, got %v", got.NextBefore)
	}

	if w := roleRequest(s, http.MethodGet, "/v1/admin/reports?status=bogus", &adminPair, ""); w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for bad status, got %d", w.Code)
	}

	if w := roleRequest(s, http.MethodPost, "/v1/admin/reports/7/resolve", &adminPair, ""); w.Code != http.StatusOK {
		t.Errorf("want 200 resolving, got %d", w.Code)
	}
	if len(reports.resolved) != 1 || reports.resolved[0] != 7 {
		t.Errorf("want report 7 resolved, got %v", reports.resolved)
	}
}

func TestTakedownReport(t *testing.T) {
	s := newRolesServer(t, &config.Config{})
	reports := &mockReports{report: &analogdb.Report{ID: 3, PostID: 42, Reason: analogdb.ReportTakedown}}
	posts := &mockDeletePosts{}
	vectors := &mockDeleteVectors{}
	s.ReportService = reports
	s.PostService = posts
	s.SimilarityService = vectors

	if w := roleRequest(s, http.MethodPost, "/v1/admin/reports/3/takedown", &adminPair, ""); w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	if posts.deleted[42] != string(analogdb.ReportTakedown) {
		t.Errorf("want post 42 deleted with the report reason, got %v", posts.deleted)
	}
	if len(vectors.deleted) != 1 || vectors.deleted[0] != 42 {
		t.Errorf("want vector 42 deleted, got %v", vectors.deleted)
	}
	if len(reports.resolvedPost) != 1 || reports.resolvedPost[0] != 42 {
		t.Errorf("want reports on post 42 resolved, got %v", reports.resolvedPost)
	}

	t.Run("retry after the post is gone", func(t *testing.T) {
		posts.missing = true
		reports.report.Post.Removed = true
		if w := roleRequest(s, http.MethodPost, "/v1/admin/reports/3/takedown", &adminPair, ""); w.Code != http.StatusOK {
			t.Errorf("want 200, got %d", w.Code)
		}
	})

	t.Run("missing post with no tombstone", func(t *testing.T) {
		reports.report.Post.Removed = false
		if w := roleRequest(s, http.MethodPost, "/v1/admin/reports/3/takedown", &adminPair, ""); w.Code != http.StatusNotFound {
			t.Errorf("want 404, got %d", w.Code)
		}
	})

	t.Run("missing report", func(t *testing.T) {
		if w := roleRequest(s, http.MethodPost, "/v1/admin/reports/4/takedown", &adminPair, ""); w.Code != http.StatusNotFound {
			t.Errorf("want 404, got %d", w.Code)
		}
	})
}
