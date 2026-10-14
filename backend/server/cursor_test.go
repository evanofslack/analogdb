package server

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/evanofslack/analogdb"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestCursorRoundTrip(t *testing.T) {
	post := &analogdb.Post{Id: 40211, DisplayPost: analogdb.DisplayPost{Time: 1752244116, Score: 153}}

	tests := []struct {
		name     string
		sort     analogdb.PostSort
		asc      bool
		seed     int
		expected analogdb.Cursor
		wantSeed int
	}{
		{"time", analogdb.PostSortTime, false, 0, analogdb.Cursor{Value: 1752244116, ID: 40211}, 0},
		{"time asc", analogdb.PostSortTime, true, 0, analogdb.Cursor{Value: 1752244116, ID: 40211}, 0},
		{"score", analogdb.PostSortScore, false, 0, analogdb.Cursor{Value: 153, ID: 40211}, 0},
		{"score asc", analogdb.PostSortScore, true, 0, analogdb.Cursor{Value: 153, ID: 40211}, 0},
		{"random", analogdb.PostSortRandom, false, 37, analogdb.Cursor{Hash: randomSortKey(40211, 37), ID: 40211}, 37},
		{"random ignores asc", analogdb.PostSortRandom, true, 37, analogdb.Cursor{Hash: randomSortKey(40211, 37), ID: 40211}, 37},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := encodeCursor(tt.sort, tt.asc, post, tt.seed)
			if err != nil {
				t.Fatal(err)
			}
			cursor, seed, err := decodeCursor(encoded, tt.sort, tt.asc)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(*cursor, tt.expected) {
				t.Errorf("expected %+v, got %+v", tt.expected, *cursor)
			}
			if seed != tt.wantSeed {
				t.Errorf("expected seed %d, got %d", tt.wantSeed, seed)
			}
		})
	}
}

func TestRandomSortKey(t *testing.T) {
	// postgres: SELECT md5(40::text || 37::text)
	if got, want := randomSortKey(40, 37), "d360a502598a4b64b936683b44a5523a"; got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestDecodeCursorErrors(t *testing.T) {
	encode := func(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }

	tests := []struct {
		name    string
		cursor  string
		sort    analogdb.PostSort
		asc     bool
		message string
	}{
		{"not base64", "!!!", analogdb.PostSortTime, false, "invalid cursor"},
		{"not json", encode("hello"), analogdb.PostSortTime, false, "invalid cursor"},
		{"missing value", encode(`{"s":"time","id":1}`), analogdb.PostSortTime, false, "invalid cursor"},
		{"string value for time", encode(`{"s":"time","v":"abc","id":1}`), analogdb.PostSortTime, false, "invalid cursor"},
		{"sort mismatch", encode(`{"s":"score","v":153,"id":1}`), analogdb.PostSortTime, false, "cursor does not match sort"},
		{"unknown sort", encode(`{"s":"best","v":153,"id":1}`), analogdb.PostSortTime, false, "cursor does not match sort"},
		{"random without seed", encode(`{"s":"random","v":"c0e190d8267e36708f955d7ab048990d","id":1}`), analogdb.PostSortRandom, false, "invalid cursor"},
		{"random bad hash", encode(`{"s":"random","v":"xyz","id":1,"seed":3}`), analogdb.PostSortRandom, false, "invalid cursor"},
		{"asc cursor on desc request", encode(`{"s":"time","v":1,"id":1,"o":"asc"}`), analogdb.PostSortTime, false, "cursor does not match sort"},
		{"desc cursor on asc request", encode(`{"s":"time","v":1,"id":1}`), analogdb.PostSortTime, true, "cursor does not match sort"},
		{"unknown order", encode(`{"s":"time","v":1,"id":1,"o":"up"}`), analogdb.PostSortTime, false, "cursor does not match sort"},
		{"random int value", encode(`{"s":"random","v":12,"id":1,"seed":3}`), analogdb.PostSortRandom, false, "invalid cursor"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := decodeCursor(tt.cursor, tt.sort, tt.asc)
			var e *analogdb.Error
			if !errors.As(err, &e) {
				t.Fatalf("expected analogdb.Error, got %v", err)
			}
			if e.Code != analogdb.ERRBADREQUEST {
				t.Errorf("expected code %s, got %s", analogdb.ERRBADREQUEST, e.Code)
			}
			if e.Message != tt.message {
				t.Errorf("expected message %q, got %q", tt.message, e.Message)
			}
		})
	}
}

func mustEncodeCursor(t *testing.T, sort analogdb.PostSort, post *analogdb.Post, seed int) string {
	t.Helper()
	c, err := encodeCursor(sort, false, post, seed)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestParseToPostFilterSeed(t *testing.T) {
	post := &analogdb.Post{Id: 5, DisplayPost: analogdb.DisplayPost{Time: 1000}}
	randomCursor := mustEncodeCursor(t, analogdb.PostSortRandom, post, 11)

	const pool = -1
	tests := []struct {
		name     string
		url      string
		expected int
	}{
		{"random with seed", "/posts?sort=random&seed=37", 37},
		{"random without seed", "/posts?sort=random", pool},
		{"zero seed", "/posts?sort=random&seed=0", pool},
		{"negative seed", "/posts?sort=random&seed=-5", pool},
		{"unparseable seed", "/posts?sort=random&seed=abc", pool},
		{"seed above int32", "/posts?sort=random&seed=4294967296", pool},
		{"cursor seed wins over param", "/posts?sort=random&seed=37&cursor=" + randomCursor, 11},
		{"cursor seed without param", "/posts?sort=random&cursor=" + randomCursor, 11},
		{"time drops seed", "/posts?sort=time&seed=37", 0},
		{"default sort drops seed", "/posts?seed=37", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			filter, err := parseToPostFilter(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			switch tt.expected {
			case 0:
				if filter.Seed != nil {
					t.Errorf("expected no seed, got %d", *filter.Seed)
				}
			case pool:
				if filter.Seed == nil || *filter.Seed < 1 || *filter.Seed > randomSeedPool {
					t.Errorf("expected seed in 1..%d, got %v", randomSeedPool, filter.Seed)
				}
			default:
				if filter.Seed == nil || *filter.Seed != tt.expected {
					t.Errorf("expected seed %d, got %v", tt.expected, filter.Seed)
				}
			}
		})
	}
}

func TestParseToPostFilterUnseededRandomVaries(t *testing.T) {
	seeds := map[int]bool{}
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest(http.MethodGet, "/posts?sort=random", nil)
		filter, err := parseToPostFilter(req)
		if err != nil {
			t.Fatal(err)
		}
		seeds[*filter.Seed] = true
	}
	if len(seeds) < 2 {
		t.Errorf("expected different seeds, got %v", seeds)
	}
}

func TestParseToPostFilterPaging(t *testing.T) {
	post := &analogdb.Post{Id: 5, DisplayPost: analogdb.DisplayPost{Time: 1000, Score: 20}}
	timeCursor := mustEncodeCursor(t, analogdb.PostSortTime, post, 0)
	scoreCursor := mustEncodeCursor(t, analogdb.PostSortScore, post, 0)

	t.Run("page_id sets keyset", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/posts?sort=time&page_id=1000", nil)
		filter, err := parseToPostFilter(req)
		if err != nil {
			t.Fatal(err)
		}
		if filter.Keyset == nil || *filter.Keyset != 1000 || filter.Cursor != nil {
			t.Errorf("expected keyset 1000 and no cursor, got %v %v", filter.Keyset, filter.Cursor)
		}
	})

	t.Run("cursor wins over page_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/posts?sort=time&page_id=1&cursor="+timeCursor, nil)
		filter, err := parseToPostFilter(req)
		if err != nil {
			t.Fatal(err)
		}
		if filter.Keyset != nil {
			t.Errorf("expected no keyset, got %d", *filter.Keyset)
		}
		if want := (analogdb.Cursor{Value: 1000, ID: 5}); filter.Cursor == nil || *filter.Cursor != want {
			t.Errorf("expected cursor %+v, got %+v", want, filter.Cursor)
		}
	})

	t.Run("asc cursor", func(t *testing.T) {
		c, err := encodeCursor(analogdb.PostSortScore, true, post, 0)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodGet, "/posts?sort=score&order=asc&cursor="+c, nil)
		filter, err := parseToPostFilter(req)
		if err != nil {
			t.Fatal(err)
		}
		if !filter.Ascending() {
			t.Error("expected ascending order")
		}
		if want := (analogdb.Cursor{Value: 20, ID: 5}); filter.Cursor == nil || *filter.Cursor != want {
			t.Errorf("expected cursor %+v, got %+v", want, filter.Cursor)
		}
	})

	for _, url := range []string{"/posts?sort=time", "/posts?sort=time&order=desc", "/posts?sort=time&order=DESC", "/posts?sort=random&order=asc"} {
		t.Run("no order for "+url, func(t *testing.T) {
			filter, err := parseToPostFilter(httptest.NewRequest(http.MethodGet, url, nil))
			if err != nil {
				t.Fatal(err)
			}
			if filter.Order != nil {
				t.Errorf("expected no order, got %s", *filter.Order)
			}
		})
	}

	t.Run("cursor with random ignores page_id", func(t *testing.T) {
		randomCursor := mustEncodeCursor(t, analogdb.PostSortRandom, post, 9)
		req := httptest.NewRequest(http.MethodGet, "/posts?sort=random&page_id=1&cursor="+randomCursor, nil)
		if _, err := parseToPostFilter(req); err != nil {
			t.Fatal(err)
		}
	})

	errTests := []struct {
		name    string
		url     string
		message string
	}{
		{"page_id with random", "/posts?sort=random&page_id=1000", "page_id is not supported with sort=random; use cursor"},
		{"page_id with random and seed", "/posts?sort=random&seed=3&page_id=1000", "page_id is not supported with sort=random; use cursor"},
		{"garbage cursor", "/posts?cursor=garbage!", "invalid cursor"},
		{"cursor sort mismatch", "/posts?sort=time&cursor=" + scoreCursor, "cursor does not match sort"},
		{"cursor default sort mismatch", "/posts?cursor=" + scoreCursor, "cursor does not match sort"},
		{"desc cursor with asc", "/posts?sort=time&order=asc&cursor=" + timeCursor, "cursor does not match sort"},
		{"invalid order", "/posts?sort=time&order=up", "invalid order parameter up, valid options are 'asc', 'desc'"},
		{"page_id with asc", "/posts?sort=time&order=asc&page_id=1000", "page_id is not supported with order=asc; use cursor"},
	}
	for _, tt := range errTests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			_, err := parseToPostFilter(req)
			var e *analogdb.Error
			if !errors.As(err, &e) || e.Code != analogdb.ERRBADREQUEST {
				t.Fatalf("expected bad request, got %v", err)
			}
			if e.Message != tt.message {
				t.Errorf("expected message %q, got %q", tt.message, e.Message)
			}
		})
	}
}

type mockFindPostService struct {
	analogdb.PostService
	filters []analogdb.PostFilter
}

func (m *mockFindPostService) FindPosts(ctx context.Context, filter *analogdb.PostFilter) ([]*analogdb.Post, int, error) {
	m.filters = append(m.filters, *filter)
	return []*analogdb.Post{}, 0, nil
}

func TestGetPostsDeprecatedPageID(t *testing.T) {
	s := mustOpen(t)
	defer mustClose(t, s)
	posts := &mockFindPostService{}
	s.PostService = posts

	tests := []struct {
		name       string
		url        string
		userAgent  string
		status     int
		deprecated bool
		client     string
	}{
		{"cursor only", "/v1/posts?sort=time", "analogdb-web/1.0", http.StatusOK, false, ""},
		{"page_id time", "/v1/posts?sort=time&page_id=100", "analogdb-scraper/1.0", http.StatusOK, true, "scraper"},
		{"page_id score", "/v1/posts?sort=score&page_id=100", "analogdb-web/1.0", http.StatusOK, true, "web"},
		{"page_id random", "/v1/posts?sort=random&page_id=100", "curl/8", http.StatusBadRequest, true, "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var before float64
			if tt.deprecated {
				before = testutil.ToFloat64(s.stats.deprecatedParams.WithLabelValues("page_id", tt.client))
			}
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req.Header.Set("User-Agent", tt.userAgent)
			w := httptest.NewRecorder()
			s.router.ServeHTTP(w, req)

			if w.Code != tt.status {
				t.Errorf("expected status %d, got %d: %s", tt.status, w.Code, w.Body.String())
			}
			if got := w.Header().Get("Deprecation") == "true"; got != tt.deprecated {
				t.Errorf("expected deprecation header %t, got %t", tt.deprecated, got)
			}
			if tt.deprecated {
				if got := w.Header().Get("Sunset"); got != sunsetDate {
					t.Errorf("expected sunset %q, got %q", sunsetDate, got)
				}
				after := testutil.ToFloat64(s.stats.deprecatedParams.WithLabelValues("page_id", tt.client))
				if after != before+1 {
					t.Errorf("expected counter to go from %v to %v, got %v", before, before+1, after)
				}
			} else if got := w.Header().Get("Sunset"); got != "" {
				t.Errorf("expected no sunset header, got %q", got)
			}
		})
	}
}

func TestMakePostResponseFetchesOneExtra(t *testing.T) {
	s := mustOpen(t)
	defer mustClose(t, s)
	posts := &mockFindPostService{}
	s.PostService = posts

	req := httptest.NewRequest(http.MethodGet, "/v1/posts?page_size=7", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	if len(posts.filters) != 1 {
		t.Fatalf("expected 1 query, got %d", len(posts.filters))
	}
	if got := *posts.filters[0].Limit; got != 8 {
		t.Errorf("expected limit 8, got %d", got)
	}
}
