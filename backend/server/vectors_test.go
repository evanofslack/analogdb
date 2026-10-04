package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/evanofslack/analogdb"
)

type mockVectorLister struct {
	ids   []int
	err   error
	calls atomic.Int32
}

func (m *mockVectorLister) VectorPostIDs(ctx context.Context) ([]int, error) {
	m.calls.Add(1)
	return m.ids, m.err
}

type mockPostIDs struct {
	analogdb.PostService
	ids []int
}

func (m *mockPostIDs) AllPostIDs(ctx context.Context) ([]int, error) {
	return m.ids, nil
}

func mustOpenVectors(t *testing.T, lister *mockVectorLister) *Server {
	t.Helper()
	s := mustOpenAdmin(t)
	s.VectorLister = lister
	s.PostService = &mockPostIDs{ids: []int{1, 2, 3, 4, 5, 6, 7}}
	return s
}

func TestMissingVectors(t *testing.T) {
	lister := &mockVectorLister{ids: []int{2, 5, 8, 9}}
	s := mustOpenVectors(t, lister)
	defer mustClose(t, s)

	missing, extra, err := s.missingVectors(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(missing, []int{7, 6, 4, 3, 1}) || extra != 2 {
		t.Errorf("want missing [7 6 4 3 1] and extra 2, got %v and %d", missing, extra)
	}

	if _, _, err := s.missingVectors(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := lister.calls.Load(); got != 1 {
		t.Errorf("want one vector scan while cached, got %d", got)
	}
}

func TestMissingVectorsUnavailable(t *testing.T) {
	s := mustOpenVectors(t, &mockVectorLister{err: errors.New("connection refused")})
	defer mustClose(t, s)

	for _, path := range []string{"/v1/admin/posts/missing?field=vector", "/v1/scrape/vectors/missing"} {
		w := adminRequest(s, path, true)
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("%s: want 503, got %d", path, w.Code)
		}
		if !strings.Contains(w.Body.String(), "vector database unavailable") {
			t.Errorf("%s: unexpected body %s", path, w.Body.String())
		}
	}

	s.VectorLister = nil
	if w := adminRequest(s, "/v1/scrape/vectors/missing", true); w.Code != http.StatusServiceUnavailable {
		t.Errorf("want 503 without lister, got %d", w.Code)
	}
}

func TestAdminMissingVectorsPaging(t *testing.T) {
	s := mustOpenVectors(t, &mockVectorLister{ids: []int{2, 5}})
	defer mustClose(t, s)

	page := func(path string) adminMissingResponse {
		t.Helper()
		w := adminRequest(s, path, true)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: want 200, got %d: %s", path, w.Code, w.Body.String())
		}
		var resp adminMissingResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}
		return resp
	}
	ids := func(resp adminMissingResponse) []int {
		out := []int{}
		for _, p := range resp.Posts {
			out = append(out, p.ID)
		}
		return out
	}

	resp := page("/v1/admin/posts/missing?field=vector&limit=2")
	if !slices.Equal(ids(resp), []int{7, 6}) || resp.NextBeforeID == nil || *resp.NextBeforeID != 6 {
		t.Fatalf("want [7 6] and next 6, got %v and %v", ids(resp), resp.NextBeforeID)
	}
	resp = page("/v1/admin/posts/missing?field=vector&limit=2&before_id=6")
	if !slices.Equal(ids(resp), []int{4, 3}) || resp.NextBeforeID == nil || *resp.NextBeforeID != 3 {
		t.Fatalf("want [4 3] and next 3, got %v and %v", ids(resp), resp.NextBeforeID)
	}
	resp = page("/v1/admin/posts/missing?field=vector&limit=2&before_id=3")
	if !slices.Equal(ids(resp), []int{1}) || resp.NextBeforeID != nil {
		t.Fatalf("want [1] and no next, got %v and %v", ids(resp), resp.NextBeforeID)
	}
	resp = page("/v1/admin/posts/missing?field=vector&before_id=1")
	if len(resp.Posts) != 0 || resp.Posts == nil {
		t.Fatalf("want empty posts list, got %v", resp.Posts)
	}
}

func TestAdminMissingInvalidField(t *testing.T) {
	s := mustOpenAdmin(t)
	defer mustClose(t, s)

	w := adminRequest(s, "/v1/admin/posts/missing?field=title", true)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "colors, caption or vector") {
		t.Errorf("unexpected message %s", w.Body.String())
	}
}

func TestVectorsMissing(t *testing.T) {
	s := mustOpenVectors(t, &mockVectorLister{ids: []int{2, 5, 9}})
	defer mustClose(t, s)

	if w := adminRequest(s, "/v1/scrape/vectors/missing", false); w.Code != http.StatusUnauthorized {
		t.Errorf("want 401 without auth, got %d", w.Code)
	}

	w := adminRequest(s, "/v1/scrape/vectors/missing", true)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp VectorsMissingResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(resp.Ids, []int{1, 3, 4, 6, 7}) || resp.Extra != 1 {
		t.Errorf("want ids [1 3 4 6 7] and extra 1, got %v and %d", resp.Ids, resp.Extra)
	}

	missing, _, err := s.missingVectors(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(missing, []int{7, 6, 4, 3, 1}) {
		t.Errorf("cached ids changed order: %v", missing)
	}
}
