package server

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/evanofslack/analogdb"
	"golang.org/x/sync/singleflight"
)

const (
	relatedHits    = 50
	relatedMinHits = 2
	relatedLimit   = 8
	// smoothing as a share of tagged posts, 5 posts in 2,000 and about 60 in 23,000
	relatedSmooth   = 0.0025
	tagCountsMaxAge = time.Hour
)

type relatedTag struct {
	tag      string
	weighted float64
	hits     int
	lift     float64
}

// relatedKeywords ranks the tags of the top hits by smoothed lift over the
// whole archive. Hit weights fall from 1.0 to 0.5 by position, like tag
// weights in the tagger. Without counts it ranks by weighted count only.
func relatedKeywords(hits []analogdb.SearchHit, queryTerms []string, df map[string]int, total int, grayscaleOnly bool) []string {
	n := min(len(hits), relatedHits)
	byTag := map[string]*relatedTag{}
	for i, hit := range hits[:n] {
		weight := 1.0
		if n > 1 {
			weight = 1.0 - 0.5*float64(i)/float64(n-1)
		}
		seen := map[string]struct{}{}
		for _, tag := range hit.Tags {
			if _, ok := seen[tag]; ok || tag == "" {
				continue
			}
			seen[tag] = struct{}{}
			rt, ok := byTag[tag]
			if !ok {
				rt = &relatedTag{tag: tag}
				byTag[tag] = rt
			}
			rt.weighted += weight
			rt.hits++
		}
	}

	terms := map[string]struct{}{}
	for _, t := range queryTerms {
		terms[t] = struct{}{}
	}
	useLift := len(df) > 0 && total > 0

	ranked := []*relatedTag{}
	for _, rt := range byTag {
		if rt.hits < relatedMinHits || inQuery(rt.tag, terms) {
			continue
		}
		if grayscaleOnly && rt.tag == monochrome {
			continue
		}
		if useLift {
			rt.lift = (rt.weighted / float64(n)) / (float64(df[rt.tag])/float64(total) + relatedSmooth)
		}
		ranked = append(ranked, rt)
	}

	sort.Slice(ranked, func(i, j int) bool {
		a, b := ranked[i], ranked[j]
		if a.lift != b.lift {
			return a.lift > b.lift
		}
		if a.weighted != b.weighted {
			return a.weighted > b.weighted
		}
		return a.tag < b.tag
	})

	related := []string{}
	for _, rt := range ranked {
		if len(related) == relatedLimit {
			break
		}
		related = append(related, rt.tag)
	}
	return related
}

// inQuery is true when the tag, or every word of a multi word tag, is a query term
func inQuery(tag string, terms map[string]struct{}) bool {
	if _, ok := terms[tag]; ok {
		return true
	}
	words := strings.Fields(tag)
	if len(words) < 2 {
		return false
	}
	for _, w := range words {
		if _, ok := terms[w]; !ok {
			return false
		}
	}
	return true
}

// tagCountCache holds the global tag counts, refreshed at most hourly
type tagCountCache struct {
	mu      sync.Mutex
	counts  map[string]int
	total   int
	fetched time.Time
	group   singleflight.Group
}

type tagCounts struct {
	counts map[string]int
	total  int
}

func (s *Server) tagCounts(ctx context.Context) (map[string]int, int, error) {
	c := &s.tagCountCache
	c.mu.Lock()
	if c.counts != nil && time.Since(c.fetched) < tagCountsMaxAge {
		counts, total := c.counts, c.total
		c.mu.Unlock()
		return counts, total, nil
	}
	c.mu.Unlock()

	v, err, _ := c.group.Do("tags", func() (any, error) {
		counts, total, err := s.KeywordService.TagCounts(context.WithoutCancel(ctx))
		if err != nil {
			return nil, err
		}
		c.mu.Lock()
		c.counts, c.total, c.fetched = counts, total, time.Now()
		c.mu.Unlock()
		return tagCounts{counts: counts, total: total}, nil
	})
	if err != nil {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.counts != nil {
			return c.counts, c.total, nil
		}
		return nil, 0, err
	}
	tc := v.(tagCounts)
	return tc.counts, tc.total, nil
}
