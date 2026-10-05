package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/evanofslack/analogdb"
)

func TestKeywordSummaryInvalidPageSize(t *testing.T) {
	s := mustOpen(t)
	defer mustClose(t, s)
	r := httptest.NewRequest(http.MethodGet, "/keywords/summary?page_size=abc", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)

	if want, got := http.StatusBadRequest, w.Code; got != want {
		t.Errorf("want status %d, got %d", want, got)
	}
}

func TestKeywordSummaryParams(t *testing.T) {
	tests := []struct {
		url        string
		wantStatus int
		wantTop    *int
		wantMin    *int
		wantDays   *int
	}{
		{url: "/v1/keywords/summary", wantStatus: http.StatusOK},
		{url: "/keywords/summary?min_count=20", wantStatus: http.StatusOK, wantMin: intPtr(20)},
		{url: "/v1/keywords/summary?page_size=60&min_count=20&top_posts=1", wantStatus: http.StatusOK, wantTop: intPtr(1), wantMin: intPtr(20)},
		{url: "/v1/keywords/summary?top_posts=50&days=7", wantStatus: http.StatusOK, wantTop: intPtr(maxTopPosts), wantDays: intPtr(7)},
		{url: "/v1/keywords/summary?top_posts=0", wantStatus: http.StatusOK, wantTop: intPtr(1)},
		{url: "/v1/keywords/summary?top_posts=abc", wantStatus: http.StatusBadRequest},
		{url: "/v1/keywords/summary?min_count=0", wantStatus: http.StatusBadRequest},
		{url: "/v1/keywords/summary?min_count=abc", wantStatus: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			s := mustOpen(t)
			defer mustClose(t, s)
			keywords := &mockKeywordService{}
			s.KeywordService = keywords

			w := httptest.NewRecorder()
			s.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.url, nil))
			if w.Code != tt.wantStatus {
				t.Fatalf("want status %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}
			if tt.wantStatus != http.StatusOK {
				if keywords.filter != nil {
					t.Error("want no query on a bad request")
				}
				return
			}
			f := keywords.filter
			if !equalIntPtr(f.TopPosts, tt.wantTop) || !equalIntPtr(f.MinCount, tt.wantMin) || !equalIntPtr(f.Days, tt.wantDays) {
				t.Errorf("unexpected filter top=%v min=%v days=%v", f.TopPosts, f.MinCount, f.Days)
			}
		})
	}
}

func TestKeywordSummaryOmitsTopPostsByDefault(t *testing.T) {
	b, err := json.Marshal(analogdb.KeywordSummary{Word: "beach", Count: 3})
	if err != nil {
		t.Fatal(err)
	}
	if want, got := `{"word":"beach","count":3}`, string(b); got != want {
		t.Errorf("want %s, got %s", want, got)
	}
}

type detailKeywordService struct {
	mockKeywordService
	words    []string
	topPosts int
	shared   map[string]int
	coCalls  int
}

func (m *detailKeywordService) FindKeyword(ctx context.Context, word string, topPosts int) (*analogdb.KeywordDetail, error) {
	m.words = append(m.words, word)
	m.topPosts = topPosts
	if word == "zzzz" {
		return nil, &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "keyword not found"}
	}
	return &analogdb.KeywordDetail{Word: word, Count: 10, TopPosts: []analogdb.CatalogPost{{Id: 1, Score: 50}}}, nil
}

func (m *detailKeywordService) CoOccurring(ctx context.Context, word string) (map[string]int, error) {
	m.coCalls++
	return m.shared, nil
}

func TestGetKeyword(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantStatus int
		wantWord   string
		wantTop    int
	}{
		{name: "single word", url: "/v1/keyword/beach", wantStatus: http.StatusOK, wantWord: "beach", wantTop: defaultKeywordTopPosts},
		{name: "multi word", url: "/v1/keyword/new%20york", wantStatus: http.StatusOK, wantWord: "new york", wantTop: defaultKeywordTopPosts},
		{name: "hyphenated and upper case", url: "/v1/keyword/Self-Portrait", wantStatus: http.StatusOK, wantWord: "self-portrait", wantTop: defaultKeywordTopPosts},
		{name: "escaped slash", url: "/v1/keyword/rock%2Froll", wantStatus: http.StatusOK, wantWord: "rock/roll", wantTop: defaultKeywordTopPosts},
		{name: "escaped percent", url: "/v1/keyword/100%25", wantStatus: http.StatusOK, wantWord: "100%", wantTop: defaultKeywordTopPosts},
		{name: "top posts clamped", url: "/v1/keyword/beach?top_posts=50", wantStatus: http.StatusOK, wantWord: "beach", wantTop: maxTopPosts},
		{name: "top posts below range", url: "/v1/keyword/beach?top_posts=0", wantStatus: http.StatusOK, wantWord: "beach", wantTop: 1},
		{name: "bad top posts", url: "/v1/keyword/beach?top_posts=abc", wantStatus: http.StatusBadRequest},
		{name: "bad related", url: "/v1/keyword/beach?related=abc", wantStatus: http.StatusBadRequest},
		{name: "unknown", url: "/v1/keyword/zzzz", wantStatus: http.StatusNotFound, wantWord: "zzzz", wantTop: defaultKeywordTopPosts},
		{name: "blank", url: "/v1/keyword/%20", wantStatus: http.StatusNotFound},
		{name: "not on legacy routes", url: "/keyword/beach", wantStatus: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := mustOpen(t)
			defer mustClose(t, s)
			keywords := &detailKeywordService{}
			s.KeywordService = keywords

			w := httptest.NewRecorder()
			s.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.url, nil))
			if w.Code != tt.wantStatus {
				t.Fatalf("want status %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}
			if tt.wantWord == "" {
				if len(keywords.words) != 0 {
					t.Errorf("want no lookup, got %v", keywords.words)
				}
				return
			}
			if !slices.Equal(keywords.words, []string{tt.wantWord}) || keywords.topPosts != tt.wantTop {
				t.Errorf("want lookup of %q with %d top posts, got %v with %d", tt.wantWord, tt.wantTop, keywords.words, keywords.topPosts)
			}
		})
	}
}

func TestGetKeywordResponse(t *testing.T) {
	s := mustOpen(t)
	defer mustClose(t, s)
	keywords := &detailKeywordService{shared: map[string]int{"ocean": 6, "beach ball": 3, "rare": 2}}
	s.KeywordService = keywords

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/keyword/beach", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("want status 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp KeywordResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Word != "beach" || resp.Count != 10 || len(resp.TopPosts) != 1 {
		t.Errorf("unexpected response %+v", resp)
	}
	words := []string{}
	for _, kw := range resp.Related {
		words = append(words, kw.Word)
	}
	if want := []string{"ocean", "beach ball"}; !slices.Equal(words, want) {
		t.Errorf("want related %v, got %v", want, words)
	}

	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/keyword/beach?related=0", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("want status 200, got %d", w.Code)
	}
	if keywords.coCalls != 1 {
		t.Errorf("want no co-occurrence query with related=0, got %d calls", keywords.coCalls)
	}
	resp = KeywordResponse{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Related == nil || len(resp.Related) != 0 {
		t.Errorf("want empty related, got %v", resp.Related)
	}
}

func TestRankCoOccurring(t *testing.T) {
	// the keyword is on 100 of 10,000 tagged posts, film is on half the archive
	shared := map[string]int{
		"film":   60,
		"ocean":  40,
		"wave":   20,
		"sand":   20,
		"rare":   2,
		"palm":   5,
		"people": 30,
	}
	df := map[string]int{"film": 5000, "ocean": 400, "wave": 100, "sand": 100, "rare": 2, "palm": 10, "people": 3000}

	got := rankCoOccurring(shared, 100, df, 10000, 30)
	if want := []string{"sand", "wave", "palm", "ocean", "film", "people"}; !slices.Equal(summaryWords(got), want) {
		t.Errorf("want %v, got %v", want, summaryWords(got))
	}
	if got[2].Count != 5 {
		t.Errorf("want co-occurrence count, got %+v", got[2])
	}

	if got := rankCoOccurring(shared, 100, df, 10000, 2); len(got) != 2 {
		t.Errorf("want 2 keywords, got %v", got)
	}

	byCount := rankCoOccurring(shared, 100, nil, 0, 3)
	if want := []string{"film", "ocean", "people"}; !slices.Equal(summaryWords(byCount), want) {
		t.Errorf("want %v without counts, got %v", want, summaryWords(byCount))
	}
}

func summaryWords(keywords []analogdb.KeywordSummary) []string {
	words := []string{}
	for _, kw := range keywords {
		words = append(words, kw.Word)
	}
	return words
}

func equalIntPtr(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
