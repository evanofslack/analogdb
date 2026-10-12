package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/config"
)

type mockHashScrape struct {
	mockScrape
	match      int
	lookups    []analogdb.ImageMatchQuery
	fail       bool
	reportFail bool
	reported   []analogdb.DuplicatePost
}

func (m *mockHashScrape) FindImageMatch(ctx context.Context, q analogdb.ImageMatchQuery) (analogdb.ImageMatch, bool, error) {
	m.lookups = append(m.lookups, q)
	if m.fail {
		return analogdb.ImageMatch{}, false, errors.New("lookup failed")
	}
	return analogdb.ImageMatch{ID: m.match, Distance: 1, SameAuthor: true}, m.match != 0, nil
}

func (m *mockHashScrape) DuplicatePosts(ctx context.Context) ([]analogdb.DuplicatePost, error) {
	return []analogdb.DuplicatePost{{KeepID: 9156, DuplicateID: 36395}}, nil
}

func (m *mockHashScrape) ReportDuplicates(ctx context.Context, dupes []analogdb.DuplicatePost) (int, error) {
	if m.reportFail {
		return 0, errors.New("report failed")
	}
	m.reported = append(m.reported, dupes...)
	return len(dupes), nil
}

func TestDuplicateRoutes(t *testing.T) {
	s := newRolesServer(t, &config.Config{})
	mock := &mockHashScrape{}
	s.ScrapeService = mock

	w := roleRequest(s, http.MethodGet, "/v1/scrape/duplicates", &scraperPair, "")
	if w.Code != http.StatusOK {
		t.Fatalf("duplicates: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var dupes DuplicatesResponse
	if err := json.NewDecoder(w.Body).Decode(&dupes); err != nil {
		t.Fatal(err)
	}
	if len(dupes.Duplicates) != 1 || dupes.Duplicates[0].KeepID != 9156 || dupes.Duplicates[0].DuplicateID != 36395 {
		t.Errorf("unexpected duplicates %+v", dupes.Duplicates)
	}

	w = roleRequest(s, http.MethodPost, "/v1/scrape/duplicates/report", &scraperPair, "")
	if w.Code != http.StatusOK {
		t.Fatalf("report duplicates: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var report ReportDuplicatesResponse
	if err := json.NewDecoder(w.Body).Decode(&report); err != nil {
		t.Fatal(err)
	}
	if report.Reported != 1 || len(report.Duplicates) != 1 || len(mock.reported) != 1 || mock.reported[0].DuplicateID != 36395 {
		t.Errorf("unexpected report response %+v, reported %+v", report, mock.reported)
	}
	if w := roleRequest(s, http.MethodPost, "/v1/scrape/duplicates/report", &webPair, ""); w.Code != http.StatusForbidden {
		t.Errorf("report duplicates: want 403 for web, got %d", w.Code)
	}

	if w := roleRequest(s, http.MethodGet, "/v1/scrape/duplicates", &webPair, ""); w.Code != http.StatusForbidden {
		t.Errorf("duplicates: want 403 for web, got %d", w.Code)
	}
}

func TestCreatePostDuplicate(t *testing.T) {
	hash := "c3a1f0e0d8b8b0f0"
	tests := []struct {
		name       string
		mode       string
		hash       *string
		match      int
		fail       bool
		reportFail bool
		want       int
		lookups    int
		reported   int
	}{
		{name: "no hash", mode: config.DuplicateReport, want: http.StatusCreated},
		{name: "off", mode: config.DuplicateOff, hash: &hash, match: 9156, want: http.StatusCreated},
		{name: "no match", mode: config.DuplicateReport, hash: &hash, want: http.StatusCreated, lookups: 1},
		{name: "match logged", mode: config.DuplicateLog, hash: &hash, match: 9156, want: http.StatusCreated, lookups: 1},
		{name: "match reported", mode: config.DuplicateReport, hash: &hash, match: 9156, want: http.StatusCreated, lookups: 1, reported: 1},
		{name: "report fails", mode: config.DuplicateReport, hash: &hash, match: 9156, reportFail: true, want: http.StatusCreated, lookups: 1},
		{name: "match blocked", mode: config.DuplicateBlock, hash: &hash, match: 9156, want: http.StatusConflict, lookups: 1},
		{name: "lookup fails", mode: config.DuplicateBlock, hash: &hash, fail: true, want: http.StatusCreated, lookups: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.App.DuplicateMode = tt.mode
			s := newTestServer(t, cfg)
			posts := &mockPostService{}
			scrape := &mockHashScrape{match: tt.match, fail: tt.fail, reportFail: tt.reportFail}
			s.PostService = posts
			s.ScrapeService = scrape
			s.SimilarityService = &mockRecordSimilarityService{}

			body, err := json.Marshal(analogdb.CreatePost{
				Title:     "repost",
				Author:    "u/a",
				Images:    []analogdb.Image{{Label: "low"}, {Label: "medium"}, {Label: "high"}, {Label: "raw", Width: 2000, Height: 3000}},
				ImageHash: tt.hash,
			})
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			s.createPost(w, httptest.NewRequest(http.MethodPost, "/post", bytes.NewReader(body)))

			if w.Code != tt.want {
				t.Fatalf("want %d, got %d: %s", tt.want, w.Code, w.Body.String())
			}
			if len(scrape.lookups) != tt.lookups {
				t.Errorf("want %d lookups, got %d", tt.lookups, len(scrape.lookups))
			}
			if tt.lookups > 0 {
				q := scrape.lookups[0]
				if q.Author != "u/a" || q.Width != 2000 || q.Height != 3000 {
					t.Errorf("want author and raw size in the lookup, got %+v", q)
				}
			}
			if len(scrape.reported) != tt.reported {
				t.Errorf("want %d reports, got %+v", tt.reported, scrape.reported)
			}
			if tt.reported > 0 && (scrape.reported[0].KeepID != 9156 || scrape.reported[0].DuplicateID != posts.created[0].Id || !scrape.reported[0].SameAuthor) {
				t.Errorf("want the new post reported against 9156, got %+v", scrape.reported[0])
			}
			created := tt.want == http.StatusCreated
			if got := len(posts.created) == 1; got != created {
				t.Errorf("want created %v, got %d posts", created, len(posts.created))
			}
			if !created && !strings.Contains(w.Body.String(), "duplicate of post 9156") {
				t.Errorf("want duplicate message, got %s", w.Body.String())
			}
		})
	}

	cfg := &config.Config{}
	cfg.App.DuplicateMode = config.DuplicateReport
	s := newTestServer(t, cfg)
	s.PostService = &mockPostService{}
	s.ScrapeService = &mockHashScrape{}
	w := httptest.NewRecorder()
	s.createPost(w, httptest.NewRequest(http.MethodPost, "/post", strings.NewReader(`{"title":"x","image_hash":"short"}`)))
	if w.Code != http.StatusBadRequest {
		t.Errorf("bad hash: want 400, got %d", w.Code)
	}
}

func TestDeletePostReason(t *testing.T) {
	s := newRolesServer(t, &config.Config{})
	posts := &mockDeletePosts{}
	s.PostService = posts
	s.SimilarityService = &mockDeleteVectors{}

	w := roleRequest(s, http.MethodDelete, "/v1/post/36395?reason=duplicate+of+9156", &adminPair, "")
	if w.Code != http.StatusOK {
		t.Fatalf("want 200 for admin, got %d: %s", w.Code, w.Body.String())
	}
	if got := posts.deleted[36395]; got != "duplicate of 9156" {
		t.Errorf("want reason passed through, got %q", got)
	}

	if w := roleRequest(s, http.MethodDelete, "/v1/post/7", &adminPair, ""); w.Code != http.StatusOK {
		t.Fatalf("want 200 for admin, got %d", w.Code)
	}
	if got, ok := posts.deleted[7]; !ok || got != "" {
		t.Errorf("want empty reason, got %q", got)
	}

	long := strings.Repeat("a", maxDeleteReason+1)
	if w := roleRequest(s, http.MethodDelete, "/v1/post/8?reason="+long, &adminPair, ""); w.Code != http.StatusBadRequest {
		t.Errorf("long reason: want 400, got %d", w.Code)
	}
	if _, ok := posts.deleted[8]; ok {
		t.Error("want no delete for a long reason")
	}
}
