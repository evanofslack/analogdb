package server

import (
	"context"
	"errors"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/evanofslack/analogdb"
)

func hitsWithTags(tags ...[]string) []analogdb.SearchHit {
	hits := make([]analogdb.SearchHit, len(tags))
	for i, t := range tags {
		hits[i] = analogdb.SearchHit{PostID: i + 1, Tags: t}
	}
	return hits
}

func TestRelatedKeywordsDropsQueryAndRareTags(t *testing.T) {
	hits := hitsWithTags(
		[]string{"sunset", "girl", "beach", "dress", "new york"},
		[]string{"sunset", "girl", "beach", "new york"},
		[]string{"sunset", "dog"},
	)
	got := relatedKeywords(hits, []string{"girl", "sunset", "new", "york"}, nil, 0, false)
	if want := []string{"beach"}; !slices.Equal(got, want) {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestRelatedKeywordsWeightsByPosition(t *testing.T) {
	hits := hitsWithTags(
		[]string{"early"},
		[]string{"early"},
		[]string{"late"},
		[]string{"late"},
	)
	got := relatedKeywords(hits, nil, nil, 0, false)
	if want := []string{"early", "late"}; !slices.Equal(got, want) {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestRelatedKeywordsSmoothedLift(t *testing.T) {
	hits := hitsWithTags(
		[]string{"sky", "rare", "dusk"},
		[]string{"sky", "rare", "dusk"},
		[]string{"sky", "dusk"},
	)
	df := map[string]int{"sky": 5000, "rare": 1, "dusk": 40}
	got := relatedKeywords(hits, nil, df, 10000, false)
	if want := []string{"rare", "dusk", "sky"}; !slices.Equal(got, want) {
		t.Errorf("want %v, got %v", want, got)
	}

	// smoothing keeps a tag on two posts from beating a common one on many hits
	hits = hitsWithTags(
		[]string{"dusk", "rare"},
		[]string{"dusk", "rare"},
		[]string{"dusk"},
		[]string{"dusk"},
		[]string{"dusk"},
		[]string{"dusk"},
	)
	df = map[string]int{"dusk": 5, "rare": 1}
	got = relatedKeywords(hits, nil, df, 10000, false)
	if want := []string{"dusk", "rare"}; !slices.Equal(got, want) {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestRelatedKeywordsMonochrome(t *testing.T) {
	hits := hitsWithTags(
		[]string{"monochrome", "street"},
		[]string{"monochrome", "street"},
	)
	if got := relatedKeywords(hits, nil, nil, 0, true); !slices.Equal(got, []string{"street"}) {
		t.Errorf("want monochrome dropped, got %v", got)
	}
	if got := relatedKeywords(hits, nil, nil, 0, false); !slices.Equal(got, []string{"monochrome", "street"}) {
		t.Errorf("want monochrome kept, got %v", got)
	}
}

func TestRelatedKeywordsLimit(t *testing.T) {
	tags := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	got := relatedKeywords(hitsWithTags(tags, tags), nil, nil, 0, false)
	if len(got) != relatedLimit {
		t.Errorf("want %d keywords, got %v", relatedLimit, got)
	}
	if got := relatedKeywords(nil, nil, nil, 0, false); len(got) != 0 {
		t.Errorf("want no keywords, got %v", got)
	}
}

type mockKeywordService struct {
	analogdb.KeywordService
	calls  atomic.Int64
	err    error
	filter *analogdb.KeywordFilter
}

func (m *mockKeywordService) TagCounts(ctx context.Context) (map[string]int, int, error) {
	m.calls.Add(1)
	if m.err != nil {
		return nil, 0, m.err
	}
	return map[string]int{"beach": 3}, 10, nil
}

func (m *mockKeywordService) GetKeywordSummary(ctx context.Context, filter *analogdb.KeywordFilter) (*[]analogdb.KeywordSummary, error) {
	m.filter = filter
	return &[]analogdb.KeywordSummary{}, nil
}

func TestTagCountsCached(t *testing.T) {
	s := mustOpen(t)
	defer mustClose(t, s)
	keywords := &mockKeywordService{}
	s.KeywordService = keywords

	for range 3 {
		counts, total, err := s.tagCounts(context.Background())
		if err != nil || total != 10 || counts["beach"] != 3 {
			t.Fatalf("unexpected counts %v %d %v", counts, total, err)
		}
	}
	if got := keywords.calls.Load(); got != 1 {
		t.Errorf("want 1 call, got %d", got)
	}

	failing := &mockKeywordService{err: errors.New("down")}
	s2 := mustOpen(t)
	defer mustClose(t, s2)
	s2.KeywordService = failing
	if _, _, err := s2.tagCounts(context.Background()); err == nil {
		t.Error("want error without cached counts")
	}
}
