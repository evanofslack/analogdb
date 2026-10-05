package server

import (
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/evanofslack/analogdb"
	"github.com/go-chi/chi/v5"
)

const (
	keywordsPath           = "/keywords"
	keywordPath            = "/keyword"
	defaultKeywordLimit    = 50
	maxKeywordLimit        = 500
	minKeywordDays         = 1
	maxKeywordDays         = 90
	defaultKeywordTopPosts = 6
	defaultKeywordRelated  = 12
	maxKeywordRelated      = 30
	keywordRelatedMinPosts = 3
)

type KeywordsResponse struct {
	Keywords []analogdb.KeywordSummary `json:"keywords"`
}

type KeywordResponse struct {
	Word     string                    `json:"word" example:"beach"`
	Count    int                       `json:"count" example:"1204"`
	TopPosts []analogdb.CatalogPost    `json:"top_posts"`
	Related  []analogdb.KeywordSummary `json:"related"`
}

func (s *Server) mountKeywordHandlers(r chi.Router) {
	r.Route(keywordsPath, func(r chi.Router) {
		r.Get("/summary", s.getSummary)
	})
}

func (s *Server) mountKeywordDetailHandlers(r chi.Router) {
	r.Get(keywordPath+"/{word}", s.getKeyword)
}

// @Summary Get keyword summary
// @Description Most common keywords with their post counts, optionally only from posts created in the last days. Top posts are the highest scoring non nsfw posts of all time, also with days
// @Tags keywords
// @Produce json
// @Param page_size query int false "Number of keywords to return" default(50)
// @Param days query int false "Only count keywords of posts created in the last days, 1 to 90"
// @Param min_count query int false "Only return keywords on at least this many posts"
// @Param top_posts query int false "Attach this many top scoring posts to each keyword (1 to 10)"
// @Success 200 {object} KeywordsResponse
// @Failure 400 {object} analogdb.Error "Invalid request"
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Router /keywords/summary [get]
func (s *Server) getSummary(w http.ResponseWriter, r *http.Request) {
	limit := defaultKeywordLimit
	var err error

	if strLimit := r.URL.Query().Get("page_size"); strLimit != "" {
		limit, err = stringToInt(strLimit)
		if err != nil {
			s.writeError(w, r, err)
			return
		}
		limit = clampLimit(limit, defaultKeywordLimit, 1, maxKeywordLimit)
	}

	filter := &analogdb.KeywordFilter{Limit: &limit}
	if strDays := r.URL.Query().Get("days"); strDays != "" {
		days, err := stringToInt(strDays)
		if err != nil {
			s.writeError(w, r, err)
			return
		}
		if days < minKeywordDays || days > maxKeywordDays {
			s.writeError(w, r, badRequest("days must be between %d and %d", minKeywordDays, maxKeywordDays))
			return
		}
		filter.Days = &days
	}
	if strMin := r.URL.Query().Get("min_count"); strMin != "" {
		minCount, err := stringToInt(strMin)
		if err != nil {
			s.writeError(w, r, err)
			return
		}
		if minCount < 1 {
			s.writeError(w, r, badRequest("invalid min_count parameter %d, must be at least 1", minCount))
			return
		}
		filter.MinCount = &minCount
	}
	if strTop := r.URL.Query().Get("top_posts"); strTop != "" {
		top, err := stringToInt(strTop)
		if err != nil {
			s.writeError(w, r, err)
			return
		}
		top = clampLimit(top, 1, 1, maxTopPosts)
		filter.TopPosts = &top
	}

	keywords, err := s.KeywordService.GetKeywordSummary(r.Context(), filter)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	response := KeywordsResponse{
		Keywords: *keywords,
	}
	if err := encodeResponse(w, r, http.StatusOK, response); err != nil {
		s.writeError(w, r, err)
	}
}

// @Summary Get keyword
// @Description Post count, top scoring non nsfw posts and related keywords of one keyword. Related keywords share posts with the keyword, ranked by smoothed lift, and their count is the number of shared posts
// @Tags keywords
// @Produce json
// @Param word path string true "Keyword, URL encoded"
// @Param top_posts query int false "Number of top scoring posts (1 to 10)" default(6)
// @Param related query int false "Number of related keywords (0 to 30)" default(12)
// @Success 200 {object} KeywordResponse
// @Failure 400 {object} analogdb.Error "Invalid request"
// @Failure 404 {object} analogdb.Error "Keyword not found"
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Router /keyword/{word} [get]
func (s *Server) getKeyword(w http.ResponseWriter, r *http.Request) {
	word, err := keywordParam(r)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if word == "" {
		s.writeError(w, r, &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "keyword not found"})
		return
	}

	topPosts := defaultKeywordTopPosts
	if strTop := r.URL.Query().Get("top_posts"); strTop != "" {
		if topPosts, err = stringToInt(strTop); err != nil {
			s.writeError(w, r, err)
			return
		}
		topPosts = clampLimit(topPosts, 1, 1, maxTopPosts)
	}

	related := defaultKeywordRelated
	if strRelated := r.URL.Query().Get("related"); strRelated != "" {
		if related, err = stringToInt(strRelated); err != nil {
			s.writeError(w, r, err)
			return
		}
		related = clampLimit(related, 0, 0, maxKeywordRelated)
	}

	detail, err := s.KeywordService.FindKeyword(r.Context(), word, topPosts)
	if err != nil {
		s.writeError(w, r, err)
		return
	}

	response := KeywordResponse{
		Word:     detail.Word,
		Count:    detail.Count,
		TopPosts: detail.TopPosts,
		Related:  []analogdb.KeywordSummary{},
	}
	if response.TopPosts == nil {
		response.TopPosts = []analogdb.CatalogPost{}
	}

	if related > 0 {
		shared, err := s.KeywordService.CoOccurring(r.Context(), word)
		if err != nil {
			s.writeError(w, r, err)
			return
		}
		df, total, err := s.tagCounts(r.Context())
		if err != nil {
			s.logger.WarnContext(r.Context(), "Fail get tag counts, ranking related keywords by count", "error", err)
		}
		response.Related = rankCoOccurring(shared, detail.Count, df, total, related)
	}

	if err := encodeResponse(w, r, http.StatusOK, response); err != nil {
		s.writeError(w, r, err)
	}
}

// keywordParam returns the lowercased word from the path. chi matches on the
// raw path when the URL has escapes that differ from the default encoding.
func keywordParam(r *http.Request) (string, error) {
	word := chi.URLParam(r, "word")
	if r.URL.RawPath != "" {
		unescaped, err := url.PathUnescape(word)
		if err != nil {
			return "", badRequest("invalid keyword %q", word)
		}
		word = unescaped
	}
	return strings.ToLower(strings.TrimSpace(word)), nil
}

// rankCoOccurring ranks tags that share posts with a keyword by smoothed lift
// over the whole archive, like the related search chips. The share of the
// keyword's posts with the tag is divided by the tag's global share plus the
// smoothing share. Without global counts it ranks by shared posts only.
func rankCoOccurring(shared map[string]int, count int, df map[string]int, total int, limit int) []analogdb.KeywordSummary {
	type ranked struct {
		word  string
		count int
		lift  float64
	}
	useLift := len(df) > 0 && total > 0 && count > 0

	tags := []ranked{}
	for word, n := range shared {
		if n < keywordRelatedMinPosts {
			continue
		}
		rt := ranked{word: word, count: n}
		if useLift {
			rt.lift = (float64(n) / float64(count)) / (float64(df[word])/float64(total) + relatedSmooth)
		}
		tags = append(tags, rt)
	}

	sort.Slice(tags, func(i, j int) bool {
		a, b := tags[i], tags[j]
		if a.lift != b.lift {
			return a.lift > b.lift
		}
		if a.count != b.count {
			return a.count > b.count
		}
		return a.word < b.word
	})

	related := []analogdb.KeywordSummary{}
	for _, rt := range tags {
		if len(related) == limit {
			break
		}
		related = append(related, analogdb.KeywordSummary{Word: rt.word, Count: rt.count})
	}
	return related
}
