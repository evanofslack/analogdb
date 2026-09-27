package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/evanofslack/analogdb"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestParamJoiner(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected string
		newValue int
	}{
		{"first param", 0, "?", 1},
		{"second param", 1, "&", 2},
		{"third param", 5, "&", 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			numParams := tt.input
			result := paramJoiner(&numParams)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
			if numParams != tt.newValue {
				t.Errorf("expected numParams to be %d, got %d", tt.newValue, numParams)
			}
		})
	}
}

func TestStringToBool(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  bool
		expectErr bool
	}{
		{"true string", "true", true, false},
		{"false string", "false", false, false},
		{"1 string", "1", true, false},
		{"0 string", "0", false, false},
		{"invalid string", "invalid", false, true},
		{"empty string", "", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := stringToBool(tt.input)
			if tt.expectErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %t, got %t", tt.expected, result)
			}
		})
	}
}

func TestStringToInt(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  int
		expectErr bool
	}{
		{"positive integer", "123", 123, false},
		{"negative integer", "-123", -123, false},
		{"zero", "0", 0, false},
		{"invalid string", "abc", 0, true},
		{"float string", "12.34", 0, true},
		{"empty string", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := stringToInt(tt.input)
			if tt.expectErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestSetMeta(t *testing.T) {
	limit20 := 20
	limit2 := 2
	limit1 := 1
	sortTime := analogdb.PostSortTime
	sortScore := analogdb.PostSortScore
	sortRandom := analogdb.PostSortRandom
	seed42 := 42

	tests := []struct {
		name     string
		filter   *analogdb.PostFilter
		posts    []*analogdb.Post
		count    int
		expected Meta
	}{
		{
			name:   "basic meta with time sort",
			filter: &analogdb.PostFilter{Limit: &limit2, Sort: &sortTime},
			posts: []*analogdb.Post{
				{DisplayPost: analogdb.DisplayPost{Time: 1000, Score: 100}},
				{DisplayPost: analogdb.DisplayPost{Time: 2000, Score: 200}},
			},
			count: 100,
			expected: Meta{
				TotalPosts: 100,
				PageSize:   2,
				PageID:     2000,
				PageURL:    "/posts?sort=time&page_size=2&page_id=2000",
			},
		},
		{
			name:   "meta with score sort",
			filter: &analogdb.PostFilter{Limit: &limit2, Sort: &sortScore},
			posts: []*analogdb.Post{
				{DisplayPost: analogdb.DisplayPost{Time: 1000, Score: 100}},
				{DisplayPost: analogdb.DisplayPost{Time: 2000, Score: 200}},
			},
			count: 50,
			expected: Meta{
				TotalPosts: 50,
				PageSize:   2,
				PageID:     200,
				PageURL:    "/posts?sort=score&page_size=2&page_id=200",
			},
		},
		{
			name:   "meta with random sort and seed",
			filter: &analogdb.PostFilter{Limit: &limit1, Sort: &sortRandom, Seed: &seed42},
			posts: []*analogdb.Post{
				{DisplayPost: analogdb.DisplayPost{Time: 1000, Score: 100}},
			},
			count: 10,
			expected: Meta{
				TotalPosts: 10,
				PageSize:   1,
				PageID:     1000,
				PageURL:    "/posts?sort=random&page_size=1&page_id=1000",
				Seed:       42,
			},
		},
		{
			name:   "end of pagination",
			filter: &analogdb.PostFilter{Limit: &limit20, Sort: &sortTime},
			posts: []*analogdb.Post{
				{DisplayPost: analogdb.DisplayPost{Time: 1000, Score: 100}},
			},
			count: 100,
			expected: Meta{
				TotalPosts: 100,
				PageSize:   20,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := setMeta(tt.filter, tt.posts, tt.count)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

func TestParseToPostFilterPageSize(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected int
	}{
		{"missing", "/posts", defaultPostsLimit},
		{"zero", "/posts?page_size=0", defaultPostsLimit},
		{"negative", "/posts?page_size=-1", defaultPostsLimit},
		{"within range", "/posts?page_size=50", 50},
		{"above max", "/posts?page_size=1000", maxPostsLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			filter, err := parseToPostFilter(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if filter.Limit == nil || *filter.Limit != tt.expected {
				t.Errorf("expected limit %d, got %v", tt.expected, filter.Limit)
			}
		})
	}
}

func FuzzParseToPostFilter(f *testing.F) {
	seeds := []string{
		"",
		"page_size=0",
		"page_size=abc",
		"sort=random&seed=7",
		"sort=bad",
		"author=x",
		"time_start=abc",
		"color=red&min_color=0.2",
		"min_color=abc",
		"width_min=1.5&ratio_max=abc",
		"keyword=a&keyword=b",
		"nsfw=maybe",
		"id=1",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, query string) {
		values, err := url.ParseQuery(query)
		if err != nil {
			return
		}
		req := httptest.NewRequest(http.MethodGet, "/posts", nil)
		req.URL.RawQuery = values.Encode()

		_, err = parseToPostFilter(req)
		if err == nil {
			return
		}
		var e *analogdb.Error
		if !errors.As(err, &e) {
			t.Fatalf("expected analogdb.Error, got %T", err)
		}
		if code := errorStatusCode(e.Code); code < 400 || code >= 500 {
			t.Fatalf("expected 4xx status, got %d", code)
		}
	})
}

type mockPostService struct {
	analogdb.PostService
	created []*analogdb.Post
}

func (m *mockPostService) CreatePost(ctx context.Context, post *analogdb.CreatePost) (*analogdb.Post, error) {
	created := &analogdb.Post{Id: len(m.created) + 1, DisplayPost: analogdb.DisplayPost{Title: post.Title}}
	m.created = append(m.created, created)
	return created, nil
}

type mockFailSimilarityService struct {
	analogdb.SimilarityService
}

func (m *mockFailSimilarityService) BatchEncodePosts(ctx context.Context, ids []int, batchSize int) error {
	return errors.New("encode failed")
}

func TestCreatePostEncodeFailure(t *testing.T) {
	s := mustOpen(t)
	defer mustClose(t, s)

	postService := &mockPostService{}
	s.PostService = postService
	s.SimilarityService = &mockFailSimilarityService{}

	body, err := json.Marshal(analogdb.CreatePost{Title: "encode failure"})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "/post", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.createPost(w, r)

	if want, got := http.StatusCreated, w.Code; got != want {
		t.Errorf("want status %d, got %d", want, got)
	}
	if len(postService.created) != 1 {
		t.Errorf("want 1 created post, got %d", len(postService.created))
	}

	var resp CreatePostResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Post.Id != postService.created[0].Id {
		t.Errorf("want post id %d, got %d", postService.created[0].Id, resp.Post.Id)
	}

	if got := testutil.ToFloat64(s.stats.postEncodeFailures); got != 1 {
		t.Errorf("want 1 encode failure, got %v", got)
	}
}
