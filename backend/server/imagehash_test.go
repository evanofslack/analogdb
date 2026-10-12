package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/config"
)

type mockHashScrape struct {
	mockScrape
	set     []analogdb.ImageHash
	match   int
	lookups int
	fail    bool
}

func (m *mockHashScrape) ImageHashMissingPosts(ctx context.Context) ([]analogdb.ImageHashMissing, error) {
	return []analogdb.ImageHashMissing{{ID: 3, URL: "https://example.com/low3.jpg"}}, nil
}

func (m *mockHashScrape) SetImageHashes(ctx context.Context, hashes []analogdb.ImageHash) (int, error) {
	m.set = append(m.set, hashes...)
	return len(hashes), nil
}

func (m *mockHashScrape) FindImageMatch(ctx context.Context, hash int64, grayscale bool) (int, bool, error) {
	m.lookups++
	if m.fail {
		return 0, false, errors.New("lookup failed")
	}
	return m.match, m.match != 0, nil
}

func (m *mockHashScrape) DuplicatePosts(ctx context.Context) ([]analogdb.DuplicatePost, error) {
	return []analogdb.DuplicatePost{{KeepID: 9156, DuplicateID: 36395}}, nil
}

func bodyRequest(s *Server, method, path, body string, creds *pair) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if creds != nil {
		r.SetBasicAuth(creds.username, creds.password)
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)
	return w
}

func TestImageHashRoutes(t *testing.T) {
	s := newRolesServer(t, &config.Config{})
	mock := &mockHashScrape{}
	s.ScrapeService = mock

	w := roleRequest(s, http.MethodGet, "/v1/scrape/hashes/missing", &scraperPair, "")
	if w.Code != http.StatusOK {
		t.Fatalf("missing: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var missing HashesMissingResponse
	if err := json.NewDecoder(w.Body).Decode(&missing); err != nil {
		t.Fatal(err)
	}
	if len(missing.Posts) != 1 || missing.Posts[0].ID != 3 || missing.Posts[0].URL == "" {
		t.Errorf("unexpected missing posts %+v", missing.Posts)
	}

	w = bodyRequest(s, http.MethodPut, "/v1/scrape/hashes", `{"hashes":[{"id":1,"image_hash":"c3a1f0e0d8b8b0f0"},{"id":2,"image_hash":"ffffffffffffffff"}]}`, &scraperPair)
	if w.Code != http.StatusOK {
		t.Fatalf("put: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var set SetHashesResponse
	if err := json.NewDecoder(w.Body).Decode(&set); err != nil {
		t.Fatal(err)
	}
	if set.Updated != 2 || len(mock.set) != 2 {
		t.Errorf("want 2 hashes set, got %d and %v", set.Updated, mock.set)
	}

	mock.set = nil
	if w := bodyRequest(s, http.MethodPut, "/v1/scrape/hashes", `{"hashes":[{"id":1,"image_hash":"c3a1f0e0d8b8b0f0"},{"id":2,"image_hash":"nothex"}]}`, &scraperPair); w.Code != http.StatusBadRequest {
		t.Errorf("bad hash: want 400, got %d", w.Code)
	}
	if mock.set != nil {
		t.Error("want no hashes set when one is invalid")
	}

	hashes := make([]string, maxHashesPerPut+1)
	for i := range hashes {
		hashes[i] = fmt.Sprintf(`{"id":%d,"image_hash":"0000000000000000"}`, i+1)
	}
	if w := bodyRequest(s, http.MethodPut, "/v1/scrape/hashes", `{"hashes":[`+strings.Join(hashes, ",")+`]}`, &scraperPair); w.Code != http.StatusBadRequest {
		t.Errorf("too many: want 400, got %d", w.Code)
	}

	w = roleRequest(s, http.MethodGet, "/v1/scrape/duplicates", &scraperPair, "")
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

	for _, path := range []string{"/v1/scrape/hashes/missing", "/v1/scrape/duplicates"} {
		if w := roleRequest(s, http.MethodGet, path, &webPair, ""); w.Code != http.StatusForbidden {
			t.Errorf("%s: want 403 for web, got %d", path, w.Code)
		}
	}
}

func TestCreatePostDuplicate(t *testing.T) {
	hash := "c3a1f0e0d8b8b0f0"
	tests := []struct {
		name    string
		block   bool
		hash    *string
		match   int
		fail    bool
		want    int
		lookups int
	}{
		{name: "no hash", block: true, want: http.StatusCreated},
		{name: "no match", block: true, hash: &hash, want: http.StatusCreated, lookups: 1},
		{name: "match logged", hash: &hash, match: 9156, want: http.StatusCreated, lookups: 1},
		{name: "match blocked", block: true, hash: &hash, match: 9156, want: http.StatusConflict, lookups: 1},
		{name: "lookup fails", block: true, hash: &hash, fail: true, want: http.StatusCreated, lookups: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.App.DuplicateBlock = tt.block
			s := newTestServer(t, cfg)
			posts := &mockPostService{}
			scrape := &mockHashScrape{match: tt.match, fail: tt.fail}
			s.PostService = posts
			s.ScrapeService = scrape
			s.SimilarityService = &mockRecordSimilarityService{}

			body, err := json.Marshal(analogdb.CreatePost{Title: "repost", ImageHash: tt.hash})
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			s.createPost(w, httptest.NewRequest(http.MethodPost, "/post", bytes.NewReader(body)))

			if w.Code != tt.want {
				t.Fatalf("want %d, got %d: %s", tt.want, w.Code, w.Body.String())
			}
			if scrape.lookups != tt.lookups {
				t.Errorf("want %d lookups, got %d", tt.lookups, scrape.lookups)
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

	s := newTestServer(t, &config.Config{})
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

	w := roleRequest(s, http.MethodDelete, "/v1/post/36395?reason=duplicate+of+9156", &scraperPair, "")
	if w.Code != http.StatusOK {
		t.Fatalf("want 200 for scraper, got %d: %s", w.Code, w.Body.String())
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
	if w := roleRequest(s, http.MethodDelete, "/v1/post/8?reason="+long, &scraperPair, ""); w.Code != http.StatusBadRequest {
		t.Errorf("long reason: want 400, got %d", w.Code)
	}
	if _, ok := posts.deleted[8]; ok {
		t.Error("want no delete for a long reason")
	}
}
