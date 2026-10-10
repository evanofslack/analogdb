package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/evanofslack/analogdb"
	"github.com/go-chi/chi/v5"
)

const (
	searchPath              = "/search"
	defaultSearchLimit      = 50
	maxSearchLimit          = 100
	maxSearchQueryLen       = 200
	maxSearchOffset         = 500
	maxSearchImageBytes     = 5 << 20
	searchImageSlots        = 2
	searchImageWait         = 5 * time.Second
	searchTextSlots         = 4
	searchTextWait          = 5 * time.Second
	errCodeUnsupportedMedia = "unsupported_media"
)

var searchImageTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
}

type SearchMeta struct {
	PageSize int `json:"page_size" example:"50"`
	// Opaque cursor for the next page, empty at the end
	NextCursor string `json:"next_cursor" example:"eyJvIjo1MH0"`
	PageURL    string `json:"next_page_url" example:"/search?cursor=eyJvIjo1MH0&page_size=50&q=girl+at+sunset"`
}

type SearchResponse struct {
	Meta            SearchMeta      `json:"meta"`
	Posts           []analogdb.Post `json:"posts"`
	RelatedKeywords []string        `json:"related_keywords" example:"beach,silhouette,dusk"`
}

// searchCursor is the JSON form of a search cursor before base64url encoding
type searchCursor struct {
	Offset int `json:"o"`
}

func encodeSearchCursor(offset int) (string, error) {
	b, err := json.Marshal(searchCursor{Offset: offset})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func decodeSearchCursor(s string) (int, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return 0, badRequest("invalid cursor")
	}
	var wire searchCursor
	if err := json.Unmarshal(b, &wire); err != nil || wire.Offset < 0 || wire.Offset > maxSearchOffset {
		return 0, badRequest("invalid cursor")
	}
	return wire.Offset, nil
}

func (s *Server) mountSearchHandlers(r chi.Router) {
	r.Route(searchPath, func(r chi.Router) {
		r.Get("/", s.searchText)
		r.Post("/image", s.searchImage)
	})
}

// parseSearchFlags reads page_size and the nsfw, grayscale and sprocket flags
func parseSearchFlags(values url.Values, filter *analogdb.SearchFilter) error {
	filter.Limit = defaultSearchLimit
	if limit := values.Get("page_size"); limit != "" {
		intLimit, err := stringToInt(limit)
		if err != nil {
			return err
		}
		filter.Limit = clampLimit(intLimit, defaultSearchLimit, 1, maxSearchLimit)
	}
	flags := []struct {
		name string
		dst  **bool
	}{
		{"nsfw", &filter.Nsfw},
		{"grayscale", &filter.Grayscale},
		{"sprocket", &filter.Sprocket},
	}
	for _, flag := range flags {
		if v := values.Get(flag.name); v != "" {
			val, err := stringToBool(v)
			if err != nil {
				return err
			}
			*flag.dst = &val
		}
	}
	return nil
}

func parseSearchText(r *http.Request) (*analogdb.SearchFilter, searchQuery, error) {
	values := r.URL.Query()
	filter := &analogdb.SearchFilter{}

	q := strings.TrimSpace(values.Get("q"))
	if q == "" {
		return nil, searchQuery{}, badRequest("must include a search query q")
	}
	if utf8.RuneCountInString(q) > maxSearchQueryLen {
		return nil, searchQuery{}, badRequest("search query must be at most %d characters", maxSearchQueryLen)
	}
	if err := parseSearchFlags(values, filter); err != nil {
		return nil, searchQuery{}, err
	}
	if cursor := values.Get("cursor"); cursor != "" {
		offset, err := decodeSearchCursor(cursor)
		if err != nil {
			return nil, searchQuery{}, err
		}
		filter.Offset = offset
	}

	query := normalizeQuery(q)
	filter.Query = strings.Join(strings.Fields(strings.ToLower(q)), " ")
	filter.Expanded = query.expanded
	return filter, query, nil
}

// searchValues turns a search filter back into query parameters
func searchValues(filter *analogdb.SearchFilter, q string) url.Values {
	values := url.Values{}
	values.Set("q", q)
	values.Set("page_size", strconv.Itoa(filter.Limit))
	if nsfw := filter.Nsfw; nsfw != nil {
		values.Set("nsfw", strconv.FormatBool(*nsfw))
	}
	if grayscale := filter.Grayscale; grayscale != nil {
		values.Set("grayscale", strconv.FormatBool(*grayscale))
	}
	if sprock := filter.Sprocket; sprock != nil {
		values.Set("sprocket", strconv.FormatBool(*sprock))
	}
	return values
}

// @Summary Search posts by text
// @Description Hybrid search over image vectors, tags, captions and titles. Related keywords are returned on the first page only
// @Tags search
// @Produce json
// @Param q query string true "Search text, 1 to 200 characters"
// @Param page_size query int false "Number of posts per page" default(50)
// @Param cursor query string false "Opaque cursor from next_cursor for the next page"
// @Param nsfw query bool false "Filter by NSFW (true=only, false=exclude)"
// @Param grayscale query bool false "Filter by black and white (true=only, false=exclude)"
// @Param sprocket query bool false "Filter by sprocketshots (true=only, false=exclude)"
// @Success 200 {object} SearchResponse
// @Failure 400 {object} analogdb.Error "Invalid request"
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Router /search [get]
func (s *Server) searchText(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	filter, query, err := parseSearchText(r)
	if err != nil {
		s.writeError(w, r, err)
		return
	}

	resp := SearchResponse{Meta: SearchMeta{PageSize: filter.Limit}, Posts: []analogdb.Post{}, RelatedKeywords: []string{}}
	if filter.Expanded == "" {
		s.observeSearch("text", resp, start)
		if err := encodeResponse(w, r, http.StatusOK, resp); err != nil {
			s.writeError(w, r, err)
		}
		return
	}

	hits, hasMore, err := s.SearchService.SearchText(r.Context(), filter)
	if err != nil {
		s.stats.searchRequests.WithLabelValues("text", "error").Inc()
		s.writeError(w, r, err)
		return
	}

	posts, err := s.hydrateHits(r.Context(), hits)
	if err != nil {
		s.stats.searchRequests.WithLabelValues("text", "error").Inc()
		s.writeError(w, r, err)
		return
	}
	resp.Posts = posts

	if filter.Offset == 0 {
		resp.RelatedKeywords = s.related(r.Context(), hits, query.terms, filter.Grayscale)
	}

	next := filter.Offset + filter.Limit
	if hasMore && next <= maxSearchOffset {
		cursor, err := encodeSearchCursor(next)
		if err != nil {
			s.writeError(w, r, err)
			return
		}
		resp.Meta.NextCursor = cursor
		values := searchValues(filter, strings.TrimSpace(r.URL.Query().Get("q")))
		values.Set("cursor", cursor)
		resp.Meta.PageURL = searchPath + "?" + values.Encode()
	}

	s.observeSearch("text", resp, start)
	if err := encodeResponse(w, r, http.StatusOK, resp); err != nil {
		s.writeError(w, r, err)
	}
}

// @Summary Search posts by image
// @Description Find posts that look like an uploaded JPEG, PNG or WebP image of at most 5 MB. Returns one page
// @Tags search
// @Accept multipart/form-data
// @Produce json
// @Param image formData file true "Image to search with"
// @Param page_size query int false "Number of posts to return" default(50)
// @Param nsfw query bool false "Filter by NSFW (true=only, false=exclude)"
// @Param grayscale query bool false "Filter by black and white (true=only, false=exclude)"
// @Param sprocket query bool false "Filter by sprocketshots (true=only, false=exclude)"
// @Success 200 {object} SearchResponse
// @Failure 400 {object} analogdb.Error "Invalid request"
// @Failure 413 {object} analogdb.Error "Image too large"
// @Failure 415 {object} analogdb.Error "Unsupported image type"
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Failure 503 {object} analogdb.Error "Too many image searches"
// @Router /search/image [post]
func (s *Server) searchImage(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	filter := &analogdb.SearchFilter{}
	if err := parseSearchFlags(r.URL.Query(), filter); err != nil {
		s.writeError(w, r, err)
		return
	}

	image, err := readSearchImage(w, r)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	filter.Image = image

	release, err := acquireSlot(r.Context(), s.imageSlots, s.imageWait, "too many image searches, try again")
	if err != nil {
		s.stats.searchRequests.WithLabelValues("image", "error").Inc()
		s.writeError(w, r, err)
		return
	}
	hits, err := s.SearchService.SearchImage(r.Context(), filter)
	release()
	if err != nil {
		s.stats.searchRequests.WithLabelValues("image", "error").Inc()
		s.writeError(w, r, err)
		return
	}

	posts, err := s.hydrateHits(r.Context(), hits)
	if err != nil {
		s.stats.searchRequests.WithLabelValues("image", "error").Inc()
		s.writeError(w, r, err)
		return
	}
	resp := SearchResponse{
		Meta:            SearchMeta{PageSize: filter.Limit},
		Posts:           posts,
		RelatedKeywords: s.related(r.Context(), hits, nil, filter.Grayscale),
	}

	s.observeSearch("image", resp, start)
	if err := encodeResponse(w, r, http.StatusOK, resp); err != nil {
		s.writeError(w, r, err)
	}
}

// readSearchImage reads the image form field, rejecting large bodies and
// files that are not JPEG, PNG or WebP
func readSearchImage(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSearchImageBytes)
	if err := r.ParseMultipartForm(maxSearchImageBytes); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return nil, &analogdb.Error{Code: errCodeTooLarge, Message: "image too large"}
		}
		return nil, badRequest("must upload an image as multipart form data")
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()

	file, _, err := r.FormFile("image")
	if err != nil {
		return nil, badRequest("must upload an image in the image field")
	}
	defer func() { _ = file.Close() }()

	image, err := io.ReadAll(file)
	if err != nil {
		return nil, badRequest("could not read image")
	}
	if _, ok := searchImageTypes[http.DetectContentType(image)]; !ok {
		return nil, &analogdb.Error{Code: errCodeUnsupportedMedia, Message: "image must be jpeg, png or webp"}
	}
	return image, nil
}

// acquireSlot waits for a free search slot, giving up after wait
func acquireSlot(ctx context.Context, slots chan struct{}, wait time.Duration, busy string) (func(), error) {
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case slots <- struct{}{}:
		return func() { <-slots }, nil
	case <-timer.C:
		return nil, &analogdb.Error{Code: analogdb.ERRUNAVAILABLE, Message: busy}
	case <-ctx.Done():
		return nil, &analogdb.Error{Code: analogdb.ERRUNAVAILABLE, Message: "search cancelled"}
	}
}

// textSlotSearch caps concurrent text searches, so a flood can't starve the
// CLIP encoder
type textSlotSearch struct {
	analogdb.SearchService
	slots chan struct{}
	wait  time.Duration
}

func (t textSlotSearch) SearchText(ctx context.Context, filter *analogdb.SearchFilter) ([]analogdb.SearchHit, bool, error) {
	release, err := acquireSlot(ctx, t.slots, t.wait, "too many searches, try again")
	if err != nil {
		return nil, false, err
	}
	defer release()
	return t.SearchService.SearchText(ctx, filter)
}

// LimitTextSearch wraps the vector search service with the server's text
// search slots. Wrap it inside any cache, so cache hits never wait
func (s *Server) LimitTextSearch(svc analogdb.SearchService) analogdb.SearchService {
	return textSlotSearch{SearchService: svc, slots: s.textSlots, wait: s.textWait}
}

// hydrateHits loads the posts of hits from the post service in hit order,
// dropping posts that no longer exist
func (s *Server) hydrateHits(ctx context.Context, hits []analogdb.SearchHit) ([]analogdb.Post, error) {
	out := []analogdb.Post{}
	if len(hits) == 0 {
		return out, nil
	}
	ids := make([]int, 0, len(hits))
	for _, hit := range hits {
		ids = append(ids, hit.PostID)
	}
	ids = analogdb.DedupeIDs(ids)

	posts, _, err := s.PostService.FindPosts(ctx, analogdb.NewPostFilterWithIDs(ids))
	if err != nil {
		return nil, err
	}
	for _, p := range analogdb.OrderPostsByIDs(posts, ids) {
		out = append(out, *p)
	}
	return out, nil
}

func (s *Server) related(ctx context.Context, hits []analogdb.SearchHit, terms []string, grayscale *bool) []string {
	df, total, err := s.tagCounts(ctx)
	if err != nil {
		s.logger.WarnContext(ctx, "Fail get tag counts, ranking related keywords by count", "error", err)
		df, total = nil, 0
	}
	return relatedKeywords(hits, terms, df, total, grayscale != nil && *grayscale)
}

func (s *Server) observeSearch(kind string, resp SearchResponse, start time.Time) {
	result := "ok"
	if len(resp.Posts) == 0 {
		result = "empty"
	}
	s.stats.searchRequests.WithLabelValues(kind, result).Inc()
	s.stats.searchDuration.WithLabelValues(kind).Observe(time.Since(start).Seconds())
}
