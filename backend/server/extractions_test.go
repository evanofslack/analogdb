package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evanofslack/analogdb"
)

type mockExtractions struct {
	upserted []*analogdb.PostExtraction
	filter   *analogdb.ExtractionFilter
}

func (m *mockExtractions) UpsertExtractions(ctx context.Context, extractions []*analogdb.PostExtraction) (int, []int, error) {
	m.upserted = extractions
	return len(extractions) - 1, []int{extractions[len(extractions)-1].PostID}, nil
}

func (m *mockExtractions) FindExtractions(ctx context.Context, filter *analogdb.ExtractionFilter) ([]*analogdb.PostExtraction, error) {
	m.filter = filter
	extractions := make([]*analogdb.PostExtraction, 0, filter.Limit)
	for i := 0; i < filter.Limit; i++ {
		extractions = append(extractions, &analogdb.PostExtraction{PostID: 100 - i})
	}
	return extractions, nil
}

func extractionJSON(postID int) string {
	return fmt.Sprintf(`{"post_id": %d, "extractor_version": "v", "model": "m", "input": "title: x",
		"input_hash": "h", "raw": {"cameras": []}, "unmatched": [{"kind": "camera", "key": "nikonfm"}]}`, postID)
}

func postExtractions(s *Server, body string, authed bool) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/extractions", strings.NewReader(body))
	if authed {
		r.SetBasicAuth("admin", "secret")
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)
	return w
}

func TestAdminExtractionsRequireAuth(t *testing.T) {
	s := mustOpenAdmin(t)
	defer mustClose(t, s)
	s.ExtractionService = &mockExtractions{}

	if w := adminRequest(s, "/v1/admin/extractions", false); w.Code != http.StatusUnauthorized {
		t.Errorf("GET: want 401 without auth, got %d", w.Code)
	}
	body := `{"extractions": [` + extractionJSON(1) + `]}`
	if w := postExtractions(s, body, false); w.Code != http.StatusUnauthorized {
		t.Errorf("POST: want 401 without auth, got %d", w.Code)
	}
}

func TestAdminUpsertExtractions(t *testing.T) {
	s := mustOpenAdmin(t)
	defer mustClose(t, s)
	mock := &mockExtractions{}
	s.ExtractionService = mock

	body := `{"extractions": [` + extractionJSON(1) + `, ` + extractionJSON(2) + `]}`
	w := postExtractions(s, body, true)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("want Cache-Control no-store, got %q", got)
	}
	var resp UpsertExtractionsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Written != 1 || len(resp.Skipped) != 1 || resp.Skipped[0] != 2 {
		t.Errorf("unexpected response %+v", resp)
	}
	if len(mock.upserted) != 2 || mock.upserted[0].Input != "title: x" || string(mock.upserted[0].Raw) != `{"cameras": []}` {
		t.Errorf("unexpected upserted %+v", mock.upserted)
	}
}

func TestAdminUpsertExtractionsValidation(t *testing.T) {
	s := mustOpenAdmin(t)
	defer mustClose(t, s)
	s.ExtractionService = &mockExtractions{}

	many := make([]string, maxExtractionsBatch+1)
	for i := range many {
		many[i] = extractionJSON(i + 1)
	}
	cases := map[string]string{
		"empty":           `{"extractions": []}`,
		"too many":        `{"extractions": [` + strings.Join(many, ",") + `]}`,
		"bad post id":     `{"extractions": [` + extractionJSON(0) + `]}`,
		"missing model":   `{"extractions": [{"post_id": 1, "extractor_version": "v", "input_hash": "h", "raw": {}}]}`,
		"raw not object":  `{"extractions": [{"post_id": 1, "extractor_version": "v", "model": "m", "input_hash": "h", "raw": []}]}`,
		"unmatched wrong": `{"extractions": [{"post_id": 1, "extractor_version": "v", "model": "m", "input_hash": "h", "raw": {}, "unmatched": {}}]}`,
	}
	for name, body := range cases {
		if w := postExtractions(s, body, true); w.Code != http.StatusBadRequest {
			t.Errorf("%s: want 400, got %d: %s", name, w.Code, w.Body.String())
		}
	}

	if w := postExtractions(s, `not json`, true); w.Code != http.StatusUnprocessableEntity {
		t.Errorf("bad json: want 422, got %d", w.Code)
	}
}

func TestAdminUpsertExtractionsLargeBody(t *testing.T) {
	s := mustOpenAdmin(t)
	defer mustClose(t, s)
	s.ExtractionService = &mockExtractions{}

	input := strings.Repeat("a", 4000)
	items := make([]string, maxExtractionsBatch)
	for i := range items {
		items[i] = fmt.Sprintf(`{"post_id": %d, "extractor_version": "v", "model": "m", "input": %q, "input_hash": "h", "raw": {}}`, i+1, input)
	}
	body := `{"extractions": [` + strings.Join(items, ",") + `]}`
	if len(body) <= maxBodyBytes {
		t.Fatalf("want body over the default limit, got %d bytes", len(body))
	}
	if w := postExtractions(s, body, true); w.Code != http.StatusOK {
		t.Errorf("want 200 for a full batch, got %d", w.Code)
	}
}

func TestAdminGetExtractions(t *testing.T) {
	s := mustOpenAdmin(t)
	defer mustClose(t, s)
	mock := &mockExtractions{}
	s.ExtractionService = mock

	w := adminRequest(s, "/v1/admin/extractions?has_unmatched=true&kind=camera&key=nikonfm&full=true&before_id=50&limit=2", true)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	f := mock.filter
	if f.HasUnmatched == nil || !*f.HasUnmatched || *f.Kind != "camera" || *f.Key != "nikonfm" || !f.Full || *f.BeforeID != 50 || f.Limit != 2 {
		t.Errorf("unexpected filter %+v", f)
	}
	var resp ExtractionsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.NextBeforeID == nil || *resp.NextBeforeID != 99 {
		t.Errorf("want next_before_id 99, got %v", resp.NextBeforeID)
	}

	adminRequest(s, "/v1/admin/extractions?limit=10000", true)
	if mock.filter.Limit != maxExtractionsLimit {
		t.Errorf("want limit clamped to %d, got %d", maxExtractionsLimit, mock.filter.Limit)
	}

	for _, path := range []string{
		"/v1/admin/extractions?kind=lens",
		"/v1/admin/extractions?has_unmatched=maybe",
		"/v1/admin/extractions?before_id=x",
	} {
		if w := adminRequest(s, path, true); w.Code != http.StatusBadRequest {
			t.Errorf("%s: want 400, got %d", path, w.Code)
		}
	}
}

type mockPatchPostService struct {
	analogdb.PostService
	patched *analogdb.PatchPost
}

func (m *mockPatchPostService) PatchPost(ctx context.Context, patch *analogdb.PatchPost, id int) error {
	m.patched = patch
	return nil
}

func TestPatchPostClear(t *testing.T) {
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

	if code := patch(`{"clear": ["film_speed", "focal_length"]}`); code != http.StatusOK {
		t.Fatalf("want 200, got %d", code)
	}
	if got := posts.patched.Clear; len(got) != 2 || got[0] != "film_speed" {
		t.Errorf("want clear passed through, got %v", got)
	}

	posts.patched = nil
	for body, name := range map[string]string{
		`{"clear": ["score"]}`:                         "not clearable",
		`{"film_speed": 400, "clear": ["film_speed"]}`: "set and cleared",
	} {
		if code := patch(body); code != http.StatusBadRequest {
			t.Errorf("%s: want 400, got %d", name, code)
		}
	}
	if posts.patched != nil {
		t.Error("want no patch for invalid clear")
	}
}
