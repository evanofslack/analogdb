package server

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/evanofslack/analogdb"
)

type mockSearchService struct {
	hits    []analogdb.SearchHit
	hasMore bool
	filters []analogdb.SearchFilter
	block   chan struct{}
	started chan struct{}
}

func (m *mockSearchService) SearchText(ctx context.Context, filter *analogdb.SearchFilter) ([]analogdb.SearchHit, bool, error) {
	m.filters = append(m.filters, *filter)
	return m.hits, m.hasMore, nil
}

func (m *mockSearchService) SearchImage(ctx context.Context, filter *analogdb.SearchFilter) ([]analogdb.SearchHit, error) {
	if m.started != nil {
		m.started <- struct{}{}
	}
	if m.block != nil {
		<-m.block
	}
	return m.hits, nil
}

type mockSearchPostService struct {
	analogdb.PostService
	posts map[int]*analogdb.Post
}

func (m *mockSearchPostService) FindPosts(ctx context.Context, filter *analogdb.PostFilter) ([]*analogdb.Post, int, error) {
	posts := []*analogdb.Post{}
	for _, id := range *filter.IDs {
		if p, ok := m.posts[id]; ok {
			posts = append(posts, p)
		}
	}
	slices.Reverse(posts)
	return posts, len(posts), nil
}

func openSearchServer(t *testing.T, hits []analogdb.SearchHit, hasMore bool, postIDs ...int) (*Server, *mockSearchService) {
	t.Helper()
	s := mustOpen(t)
	t.Cleanup(func() { mustClose(t, s) })
	posts := map[int]*analogdb.Post{}
	for _, id := range postIDs {
		posts[id] = &analogdb.Post{Id: id}
	}
	search := &mockSearchService{hits: hits, hasMore: hasMore}
	s.SearchService = search
	s.PostService = &mockSearchPostService{posts: posts}
	s.KeywordService = &mockKeywordService{}
	return s, search
}

func doSearch(t *testing.T, s *Server, target string) (*httptest.ResponseRecorder, SearchResponse) {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, target, nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)
	var resp SearchResponse
	if w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
	}
	return w, resp
}

func TestSearchCursor(t *testing.T) {
	cursor, err := encodeSearchCursor(150)
	if err != nil {
		t.Fatal(err)
	}
	if offset, err := decodeSearchCursor(cursor); err != nil || offset != 150 {
		t.Errorf("want 150, got %d %v", offset, err)
	}
	over, _ := encodeSearchCursor(maxSearchOffset + 1)
	for _, bad := range []string{"garbage!", "bm90IGpzb24", over} {
		if _, err := decodeSearchCursor(bad); analogdb.ErrorCode(err) != analogdb.ERRBADREQUEST {
			t.Errorf("%q: want bad request, got %v", bad, err)
		}
	}
}

func TestSearchParams(t *testing.T) {
	s, search := openSearchServer(t, nil, false)
	long := strings.Repeat("a", maxSearchQueryLen+1)

	tests := []struct {
		target string
		status int
	}{
		{"/v1/search", http.StatusBadRequest},
		{"/v1/search?q=%20%20", http.StatusBadRequest},
		{"/v1/search?q=" + long, http.StatusBadRequest},
		{"/v1/search?q=" + strings.Repeat("a", maxSearchQueryLen), http.StatusOK},
		{"/v1/search?q=cat&page_size=abc", http.StatusBadRequest},
		{"/v1/search?q=cat&nsfw=maybe", http.StatusBadRequest},
		{"/v1/search?q=cat&cursor=garbage", http.StatusBadRequest},
		{"/search?q=cat", http.StatusNotFound},
	}
	for _, tt := range tests {
		if w, _ := doSearch(t, s, tt.target); w.Code != tt.status {
			t.Errorf("%s: want %d, got %d", tt.target, tt.status, w.Code)
		}
	}

	search.filters = nil
	for _, tt := range []struct {
		pageSize string
		want     int
	}{{"", 50}, {"0", 50}, {"20", 20}, {"500", 100}} {
		doSearch(t, s, "/v1/search?q=cat&page_size="+tt.pageSize)
		if got := search.filters[len(search.filters)-1].Limit; got != tt.want {
			t.Errorf("page_size %q: want %d, got %d", tt.pageSize, tt.want, got)
		}
	}

	doSearch(t, s, "/v1/search?q=Black+and+White+flowers&nsfw=false&grayscale=true")
	f := search.filters[len(search.filters)-1]
	if f.Nsfw == nil || *f.Nsfw || f.Grayscale == nil || !*f.Grayscale || f.Sprocket != nil {
		t.Errorf("unexpected flags %+v", f)
	}
	if f.Query != "black and white flowers" || f.Expanded != "black and white flowers monochrome flower" {
		t.Errorf("unexpected query %q expanded %q", f.Query, f.Expanded)
	}
}

func TestSearchText(t *testing.T) {
	hits := []analogdb.SearchHit{
		{PostID: 3, Tags: []string{"cat", "sofa", "window"}},
		{PostID: 9, Tags: []string{"cat", "sofa"}},
		{PostID: 1, Tags: []string{"cat", "window"}},
	}
	s, search := openSearchServer(t, hits, true, 1, 3)

	w, resp := doSearch(t, s, "/v1/search?q=cats&page_size=3&grayscale=false")
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var ids []int
	for _, p := range resp.Posts {
		ids = append(ids, p.Id)
	}
	if want := []int{3, 1}; !slices.Equal(ids, want) {
		t.Errorf("want posts %v, got %v", want, ids)
	}
	if want := []string{"sofa", "window"}; !slices.Equal(resp.RelatedKeywords, want) {
		t.Errorf("want related %v, got %v", want, resp.RelatedKeywords)
	}
	if resp.Meta.PageSize != 3 || resp.Meta.NextCursor == "" {
		t.Fatalf("unexpected meta %+v", resp.Meta)
	}
	next, err := url.Parse(resp.Meta.PageURL)
	if err != nil || next.Path != "/search" {
		t.Fatalf("unexpected next page url %q", resp.Meta.PageURL)
	}
	if q := next.Query(); q.Get("q") != "cats" || q.Get("page_size") != "3" || q.Get("grayscale") != "false" || q.Get("cursor") != resp.Meta.NextCursor {
		t.Errorf("unexpected next page url %q", resp.Meta.PageURL)
	}

	w, resp = doSearch(t, s, "/v1"+resp.Meta.PageURL)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if got := search.filters[len(search.filters)-1].Offset; got != 3 {
		t.Errorf("want offset 3, got %d", got)
	}
	if len(resp.RelatedKeywords) != 0 {
		t.Errorf("want no related keywords after page one, got %v", resp.RelatedKeywords)
	}
}

func TestSearchTextOffsetCap(t *testing.T) {
	s, _ := openSearchServer(t, []analogdb.SearchHit{{PostID: 1}}, true, 1)

	cursor, _ := encodeSearchCursor(450)
	_, resp := doSearch(t, s, "/v1/search?q=cat&cursor="+cursor)
	if resp.Meta.NextCursor == "" {
		t.Error("want a next cursor up to the offset cap")
	}
	cursor, _ = encodeSearchCursor(500)
	_, resp = doSearch(t, s, "/v1/search?q=cat&cursor="+cursor)
	if resp.Meta.NextCursor != "" || resp.Meta.PageURL != "" {
		t.Errorf("want no next cursor past the offset cap, got %+v", resp.Meta)
	}
}

func TestSearchTextEmptyQuery(t *testing.T) {
	s, search := openSearchServer(t, nil, false)
	w, resp := doSearch(t, s, "/v1/search?q=%21%21")
	if w.Code != http.StatusOK || len(resp.Posts) != 0 || len(search.filters) != 0 {
		t.Errorf("want empty result without a search, got %d %+v", w.Code, resp)
	}
}

func multipartImage(t *testing.T, data []byte) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	part, err := mw.CreateFormFile("image", "upload")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return body, mw.FormDataContentType()
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func postImage(s *Server, data []byte, contentType string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/v1/search/image?page_size=10", bytes.NewReader(data))
	r.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)
	return w
}

func TestSearchImage(t *testing.T) {
	hits := []analogdb.SearchHit{
		{PostID: 2, Tags: []string{"beach", "dusk"}},
		{PostID: 5, Tags: []string{"beach", "dusk"}},
	}
	s, _ := openSearchServer(t, hits, false, 2, 5)

	body, ct := multipartImage(t, pngBytes(t))
	w := postImage(s, body.Bytes(), ct)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp SearchResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Posts) != 2 || resp.Posts[0].Id != 2 || resp.Meta.NextCursor != "" || resp.Meta.PageSize != 10 {
		t.Errorf("unexpected response %+v", resp)
	}
	if len(resp.RelatedKeywords) != 2 {
		t.Errorf("want related keywords from hits, got %v", resp.RelatedKeywords)
	}
}

func TestSearchImageRejects(t *testing.T) {
	s, _ := openSearchServer(t, nil, false)

	text, ct := multipartImage(t, []byte("just some text, not an image"))
	if w := postImage(s, text.Bytes(), ct); w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("want 415, got %d", w.Code)
	}

	large := append(pngBytes(t), make([]byte, maxSearchImageBytes)...)
	big, ct := multipartImage(t, large)
	if w := postImage(s, big.Bytes(), ct); w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("want 413, got %d", w.Code)
	}

	if w := postImage(s, []byte("{}"), "application/json"); w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}

	empty, ct := multipartImage(t, nil)
	if w := postImage(s, empty.Bytes(), ct); w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("want 415 for an empty file, got %d", w.Code)
	}
}

func TestSearchImageBusy(t *testing.T) {
	s, search := openSearchServer(t, nil, false)
	s.imageWait = 50 * time.Millisecond
	search.block = make(chan struct{})
	search.started = make(chan struct{}, searchImageSlots)

	body, ct := multipartImage(t, pngBytes(t))
	done := make(chan int, searchImageSlots)
	for range searchImageSlots {
		go func() { done <- postImage(s, body.Bytes(), ct).Code }()
	}
	for range searchImageSlots {
		<-search.started
	}

	if w := postImage(s, body.Bytes(), ct); w.Code != http.StatusServiceUnavailable {
		t.Errorf("want 503 with all slots busy, got %d", w.Code)
	}

	close(search.block)
	for range searchImageSlots {
		if code := <-done; code != http.StatusOK {
			t.Errorf("want 200 for the blocked searches, got %d", code)
		}
	}
}

func TestKeywordSummaryDays(t *testing.T) {
	s := mustOpen(t)
	defer mustClose(t, s)
	keywords := &mockKeywordService{}
	s.KeywordService = keywords

	for _, tt := range []struct {
		query  string
		status int
		days   *int
	}{
		{"", http.StatusOK, nil},
		{"?days=7", http.StatusOK, intPtr(7)},
		{"?days=0", http.StatusBadRequest, nil},
		{"?days=91", http.StatusBadRequest, nil},
		{"?days=abc", http.StatusBadRequest, nil},
	} {
		keywords.filter = nil
		r := httptest.NewRequest(http.MethodGet, "/v1/keywords/summary"+tt.query, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, r)
		if w.Code != tt.status {
			t.Errorf("%q: want %d, got %d", tt.query, tt.status, w.Code)
			continue
		}
		if tt.status != http.StatusOK {
			continue
		}
		if f := keywords.filter; f == nil || f.Limit == nil || *f.Limit != defaultKeywordLimit {
			t.Errorf("%q: unexpected filter %+v", tt.query, f)
		} else if (f.Days == nil) != (tt.days == nil) || (f.Days != nil && *f.Days != *tt.days) {
			t.Errorf("%q: want days %v, got %v", tt.query, tt.days, f.Days)
		}
	}
}

func intPtr(i int) *int { return &i }
