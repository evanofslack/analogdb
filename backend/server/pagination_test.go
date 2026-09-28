package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/logger"
	"github.com/evanofslack/analogdb/postgres"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	pagingPosts    = 250
	pagingBaseTime = 1600000000
)

// seeds 250 posts where score repeats every 7 posts and time repeats in groups of 5
const pagingSeedSQL = `
INSERT INTO pictures (
    url, title, author, permalink, description, score, nsfw, greyscale, time, width, height, sprocket,
    lowurl, lowwidth, lowheight, medurl, medwidth, medheight, highurl, highwidth, highheight
)
SELECT
    'https://example.com/raw' || i || '.jpg',
    CASE WHEN i % 3 = 0 THEN 'dunes a&b ' || i ELSE 'plain ' || i END,
    'u/author' || (i % 4),
    'reddit.com/paging_' || i,
    '',
    (i % 7) * 10,
    false,
    false,
    1600000000 + (i / 5) * 3600,
    3000, 2000, false,
    'https://example.com/low.jpg', 300, 200,
    'https://example.com/med.jpg', 800, 533,
    'https://example.com/high.jpg', 1600, 1067
FROM generate_series(1, 250) AS i
`

func mustOpenPagingServer(t *testing.T) (*Server, analogdb.PostService) {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"postgres:15",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("testuser"),
		tcpostgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForAll(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(60*time.Second),
				wait.ForExposedPort(),
			),
		),
	)
	if err != nil {
		t.Fatalf("Start postgres container, err=%v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Terminate postgres container, err=%v", err)
		}
	})

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	log, err := logger.New("error", "debug", "analogdb_test")
	if err != nil {
		t.Fatal(err)
	}
	db := postgres.NewDB(connStr, log, true, "", false)
	if err := db.Open(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	raw, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = raw.Close() }()
	if _, err := raw.ExecContext(ctx, pagingSeedSQL); err != nil {
		t.Fatalf("Seed posts, err=%v", err)
	}

	s := mustOpen(t)
	t.Cleanup(func() { mustClose(t, s) })
	service := postgres.NewPostService(db)
	s.PostService = service
	return s, service
}

func getPostsPage(t *testing.T, s *Server, path string) (*httptest.ResponseRecorder, PostResponse) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	var resp PostResponse
	if w.Code == http.StatusOK {
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}
	}
	return w, resp
}

type walkResult struct {
	ids   []int
	pages []PostResponse
}

// walk follows next returns until it is empty, checking each page as it goes.
func walk(t *testing.T, s *Server, first string, next func(PostResponse) string) walkResult {
	t.Helper()
	var result walkResult
	path := first
	for path != "" {
		w, resp := getPostsPage(t, s, path)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status %d: %s", path, w.Code, w.Body.String())
		}
		for _, p := range resp.Posts {
			result.ids = append(result.ids, p.Id)
		}
		result.pages = append(result.pages, resp)
		if len(result.pages) > pagingPosts+2 {
			t.Fatal("walk did not end")
		}
		path = next(resp)
	}
	return result
}

func fullOrder(t *testing.T, service analogdb.PostService, sort analogdb.PostSort, seed int, modify func(*analogdb.PostFilter)) []int {
	t.Helper()
	limit := 10000
	filter := analogdb.NewPostFilter(&limit, &sort, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if seed != 0 {
		filter.Seed = &seed
	}
	if modify != nil {
		modify(filter)
	}
	posts, _, err := service.FindPosts(context.Background(), filter)
	if err != nil {
		t.Fatal(err)
	}
	ids := []int{}
	for _, p := range posts {
		ids = append(ids, p.Id)
	}
	return ids
}

func checkWalk(t *testing.T, result walkResult, want []int) {
	t.Helper()
	total := result.pages[0].Meta.TotalPosts
	seen := map[int]bool{}
	for _, id := range result.ids {
		if seen[id] {
			t.Fatalf("duplicate post %d", id)
		}
		seen[id] = true
	}
	if len(seen) != total {
		t.Errorf("expected %d posts, saw %d", total, len(seen))
	}
	if !reflect.DeepEqual(result.ids, want) {
		t.Errorf("walk order differs from one full query\nwant %v\ngot  %v", want, result.ids)
	}
	for i, page := range result.pages {
		last := i == len(result.pages)-1
		if page.Meta.TotalPosts != total {
			t.Errorf("page %d: total_posts %d, expected %d", i, page.Meta.TotalPosts, total)
		}
		if last != (page.Meta.NextCursor == "") {
			t.Errorf("page %d: next_cursor %q on last=%t", i, page.Meta.NextCursor, last)
		}
		if last != (page.Meta.PageURL == "") {
			t.Errorf("page %d: next_page_url %q on last=%t", i, page.Meta.PageURL, last)
		}
		if last != (page.Meta.PageID == 0) {
			t.Errorf("page %d: next_page_id %d on last=%t", i, page.Meta.PageID, last)
		}
	}
}

func TestPostsPagination(t *testing.T) {
	s, service := mustOpenPagingServer(t)

	sorts := []struct {
		sort analogdb.PostSort
		seed int
	}{
		{analogdb.PostSortTime, 0},
		{analogdb.PostSortScore, 0},
		{analogdb.PostSortRandom, 37},
	}

	for _, sc := range sorts {
		want := fullOrder(t, service, sc.sort, sc.seed, nil)
		if len(want) != pagingPosts {
			t.Fatalf("expected %d seeded posts, got %d", pagingPosts, len(want))
		}
		for _, size := range []int{1, 7, 20} {
			base := fmt.Sprintf("/v1/posts?sort=%s&page_size=%d", sc.sort, size)
			if sc.seed != 0 {
				base += fmt.Sprintf("&seed=%d", sc.seed)
			}

			t.Run(fmt.Sprintf("%s_%d_cursor", sc.sort, size), func(t *testing.T) {
				result := walk(t, s, base, func(resp PostResponse) string {
					if resp.Meta.NextCursor == "" {
						return ""
					}
					return base + "&cursor=" + url.QueryEscape(resp.Meta.NextCursor)
				})
				checkWalk(t, result, want)
				if sc.seed != 0 {
					for _, page := range result.pages {
						if page.Meta.Seed != sc.seed {
							t.Fatalf("expected seed %d, got %d", sc.seed, page.Meta.Seed)
						}
					}
				}
			})

			t.Run(fmt.Sprintf("%s_%d_next_page_url", sc.sort, size), func(t *testing.T) {
				result := walk(t, s, base, func(resp PostResponse) string {
					if resp.Meta.PageURL == "" {
						return ""
					}
					return "/v1" + resp.Meta.PageURL
				})
				checkWalk(t, result, want)
			})
		}
	}

	t.Run("exact multiple", func(t *testing.T) {
		start, end := pagingBaseTime+10*3600, pagingBaseTime+18*3600
		path := fmt.Sprintf("/v1/posts?sort=time&page_size=20&time_start=%d&time_end=%d", start, end)
		result := walk(t, s, path, func(resp PostResponse) string {
			if resp.Meta.PageURL == "" {
				return ""
			}
			return "/v1" + resp.Meta.PageURL
		})
		if len(result.pages) != 2 {
			t.Fatalf("expected 2 pages, got %d", len(result.pages))
		}
		if total := result.pages[0].Meta.TotalPosts; total != 40 {
			t.Fatalf("expected 40 posts, got %d", total)
		}
		if len(result.ids) != 40 || result.pages[1].Meta.NextCursor != "" {
			t.Errorf("expected 40 posts and an empty last cursor, got %d and %q", len(result.ids), result.pages[1].Meta.NextCursor)
		}
	})

	t.Run("legacy page_id on time", func(t *testing.T) {
		want := fullOrder(t, service, analogdb.PostSortTime, 0, nil)
		w, first := getPostsPage(t, s, "/v1/posts?sort=time&page_size=7")
		if w.Code != http.StatusOK {
			t.Fatal(w.Body.String())
		}
		keyset := first.Meta.PageID
		if keyset != first.Posts[6].Time {
			t.Fatalf("expected next_page_id %d, got %d", first.Posts[6].Time, keyset)
		}
		w, second := getPostsPage(t, s, fmt.Sprintf("/v1/posts?sort=time&page_size=7&page_id=%d", keyset))
		if w.Code != http.StatusOK {
			t.Fatal(w.Body.String())
		}
		if w.Header().Get("Deprecation") != "true" || w.Header().Get("Sunset") != sunsetDate {
			t.Error("expected deprecation headers")
		}
		// page_id pages on time alone: every post older than the keyset, newest first
		older := []int{}
		for _, id := range want {
			if pagingTime(id) < keyset {
				older = append(older, id)
			}
		}
		got := []int{}
		for _, p := range second.Posts {
			got = append(got, p.Id)
		}
		if !reflect.DeepEqual(got, older[:7]) {
			t.Errorf("expected %v, got %v", older[:7], got)
		}
		if second.Meta.TotalPosts != pagingPosts {
			t.Errorf("expected total_posts %d, got %d", pagingPosts, second.Meta.TotalPosts)
		}
	})

	t.Run("legacy page_id with random", func(t *testing.T) {
		w, _ := getPostsPage(t, s, "/v1/posts?sort=random&page_id=1600000000")
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
		var e ErrorResponse
		if err := json.NewDecoder(w.Body).Decode(&e); err != nil {
			t.Fatal(err)
		}
		if e.Error != "page_id is not supported with sort=random; use cursor" {
			t.Errorf("unexpected message %q", e.Error)
		}
		if w.Header().Get("Deprecation") != "true" {
			t.Error("expected deprecation header")
		}
	})

	t.Run("cursor errors", func(t *testing.T) {
		for _, path := range []string{"/v1/posts?cursor=garbage", "/v1/posts?sort=time&cursor=" + mustEncodeCursor(t, analogdb.PostSortScore, &analogdb.Post{Id: 1}, 0)} {
			if w, _ := getPostsPage(t, s, path); w.Code != http.StatusBadRequest {
				t.Errorf("%s: expected 400, got %d", path, w.Code)
			}
		}
	})

	t.Run("next_page_url round trip", func(t *testing.T) {
		start := pagingBaseTime + 5*3600
		values := url.Values{}
		values.Set("sort", "score")
		values.Set("page_size", "5")
		values.Set("title", "a&b")
		values.Set("time_start", fmt.Sprint(start))
		base := "/v1/posts?" + values.Encode()

		want := fullOrder(t, service, analogdb.PostSortScore, 0, func(f *analogdb.PostFilter) {
			title := "a&b"
			ts := time.Unix(int64(start), 0)
			f.Title = &title
			f.TimeStart = &ts
		})

		_, first := getPostsPage(t, s, base)
		if first.Meta.TotalPosts != len(want) || len(want) <= 5 {
			t.Fatalf("expected total %d over one page, got %d", len(want), first.Meta.TotalPosts)
		}
		next, err := url.Parse(first.Meta.PageURL)
		if err != nil {
			t.Fatal(err)
		}
		if next.Path != "/posts" {
			t.Errorf("expected relative /posts path, got %q", next.Path)
		}
		q := next.Query()
		if q.Get("title") != "a&b" || q.Get("time_start") != fmt.Sprint(start) || q.Get("page_id") != "" || q.Get("cursor") != first.Meta.NextCursor {
			t.Errorf("next_page_url lost parameters: %s", first.Meta.PageURL)
		}

		_, viaURL := getPostsPage(t, s, "/v1"+first.Meta.PageURL)
		_, viaCursor := getPostsPage(t, s, base+"&cursor="+url.QueryEscape(first.Meta.NextCursor))
		gotURL, gotCursor := []int{}, []int{}
		for i := range viaURL.Posts {
			gotURL = append(gotURL, viaURL.Posts[i].Id)
		}
		for i := range viaCursor.Posts {
			gotCursor = append(gotCursor, viaCursor.Posts[i].Id)
		}
		if !reflect.DeepEqual(gotURL, want[5:10]) || !reflect.DeepEqual(gotURL, gotCursor) {
			t.Errorf("expected page 2 %v, got %v via url and %v via cursor", want[5:10], gotURL, gotCursor)
		}
		if viaURL.Meta.TotalPosts != first.Meta.TotalPosts {
			t.Errorf("total_posts changed from %d to %d", first.Meta.TotalPosts, viaURL.Meta.TotalPosts)
		}
	})

	t.Run("unseeded random", func(t *testing.T) {
		seeds := map[int]bool{}
		var first PostResponse
		for i := 0; i < 10 && len(seeds) < 2; i++ {
			_, first = getPostsPage(t, s, "/v1/posts?sort=random&page_size=10")
			if first.Meta.Seed < 1 || first.Meta.Seed > randomSeedPool {
				t.Fatalf("expected seed in pool, got %d", first.Meta.Seed)
			}
			seeds[first.Meta.Seed] = true
		}
		if len(seeds) < 2 {
			t.Errorf("expected different seeds, got %v", seeds)
		}

		want := fullOrder(t, service, analogdb.PostSortRandom, first.Meta.Seed, nil)
		_, second := getPostsPage(t, s, "/v1"+first.Meta.PageURL)
		if second.Meta.Seed != first.Meta.Seed {
			t.Errorf("expected seed %d on page 2, got %d", first.Meta.Seed, second.Meta.Seed)
		}
		got := []int{}
		for _, p := range second.Posts {
			got = append(got, p.Id)
		}
		if !reflect.DeepEqual(got, want[10:20]) {
			t.Errorf("expected page 2 %v, got %v", want[10:20], got)
		}
	})
}

// pagingTime is the seeded time of a post, given ids 1..250 in insert order.
func pagingTime(id int) int {
	return pagingBaseTime + (id/5)*3600
}
