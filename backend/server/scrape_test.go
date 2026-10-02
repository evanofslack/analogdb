package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockCaptionScrape struct {
	mockScrape
	version *string
}

func (m *mockCaptionScrape) CaptionMissingPostIDs(ctx context.Context, version *string) ([]int, error) {
	m.version = version
	return []int{3, 7}, nil
}

func TestCaptionsMissing(t *testing.T) {
	s := mustOpenAdmin(t)
	defer mustClose(t, s)
	mock := &mockCaptionScrape{}
	s.ScrapeService = mock

	if w := adminRequest(s, "/v1/scrape/captions/missing", false); w.Code != http.StatusUnauthorized {
		t.Errorf("want 401 without auth, got %d", w.Code)
	}

	w := adminRequest(s, "/v1/scrape/captions/missing", true)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp CaptionsMissingResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Ids) != 2 || resp.Ids[0] != 3 || resp.Ids[1] != 7 {
		t.Errorf("unexpected ids %v", resp.Ids)
	}
	if mock.version != nil {
		t.Errorf("want no version, got %q", *mock.version)
	}

	if w := adminRequest(s, "/v1/scrape/captions/missing?version=v2", true); w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if mock.version == nil || *mock.version != "v2" {
		t.Errorf("want version v2, got %v", mock.version)
	}
}

func TestPostCaptionValidation(t *testing.T) {
	s := mustOpenAdmin(t)
	defer mustClose(t, s)
	posts := &mockPatchPostService{}
	s.PostService = posts

	patch := func(body string) int {
		r := httptest.NewRequest(http.MethodPatch, "/v1/post/1", strings.NewReader(body))
		r.SetBasicAuth("admin", "secret")
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, r)
		return w.Code
	}

	if code := patch(`{"caption": {"caption": "a dog", "model": "m", "version": "v1", "raw": {"tags": []}}}`); code != http.StatusOK {
		t.Fatalf("want 200, got %d", code)
	}
	if c := posts.patched.Caption; c == nil || c.Caption == nil || *c.Caption != "a dog" || string(c.Raw) != `{"tags": []}` {
		t.Errorf("want caption passed through, got %+v", c)
	}

	posts.patched = nil
	for name, body := range map[string]string{
		"missing model":  `{"caption": {"version": "v1", "raw": {}}}`,
		"missing raw":    `{"caption": {"model": "m", "version": "v1"}}`,
		"raw not object": `{"caption": {"model": "m", "version": "v1", "raw": []}}`,
	} {
		if code := patch(body); code != http.StatusBadRequest {
			t.Errorf("%s: want 400, got %d", name, code)
		}
	}
	if posts.patched != nil {
		t.Error("want no patch for invalid caption")
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/post", strings.NewReader(`{"title": "x", "caption": {"model": "m"}}`))
	r.SetBasicAuth("admin", "secret")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("create: want 400, got %d", w.Code)
	}
}
