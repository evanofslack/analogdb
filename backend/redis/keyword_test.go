package redis

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/evanofslack/analogdb"
)

type fakeKeywordService struct {
	summaryCalls atomic.Int64
	detailCalls  atomic.Int64
	sharedCalls  atomic.Int64
	countCalls   atomic.Int64
}

func (f *fakeKeywordService) GetKeywordSummary(ctx context.Context, filter *analogdb.KeywordFilter) (*[]analogdb.KeywordSummary, error) {
	f.summaryCalls.Add(1)
	summary := []analogdb.KeywordSummary{{Word: "beach", Count: 4, TopPosts: []analogdb.CatalogPost{{Id: 6, Score: 30}}}}
	return &summary, nil
}

func (f *fakeKeywordService) TagCounts(ctx context.Context) (map[string]int, int, error) {
	f.countCalls.Add(1)
	return map[string]int{"beach": 4}, 10, nil
}

func (f *fakeKeywordService) FindKeyword(ctx context.Context, word string, topPosts int) (*analogdb.KeywordDetail, error) {
	f.detailCalls.Add(1)
	if word == "zzzz" {
		return nil, &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "keyword not found"}
	}
	return &analogdb.KeywordDetail{Word: word, Count: 4, TopPosts: []analogdb.CatalogPost{{Id: 6}}}, nil
}

func (f *fakeKeywordService) CoOccurring(ctx context.Context, word string) (map[string]int, error) {
	f.sharedCalls.Add(1)
	return map[string]int{"ocean": 3}, nil
}

func TestKeywordCache(t *testing.T) {
	rdb, _ := newMiniRDB(t)
	inner := &fakeKeywordService{}
	s := NewCacheKeywordService(rdb, inner)
	ctx := context.Background()

	limit, top := 60, 1
	for range 2 {
		summary, err := s.GetKeywordSummary(ctx, &analogdb.KeywordFilter{Limit: &limit, TopPosts: &top})
		if err != nil {
			t.Fatal(err)
		}
		if len(*summary) != 1 || (*summary)[0].TopPosts[0].Id != 6 {
			t.Fatalf("unexpected summary %+v", *summary)
		}
	}
	if got := inner.summaryCalls.Load(); got != 1 {
		t.Errorf("want 1 summary query, got %d", got)
	}
	if _, err := s.GetKeywordSummary(ctx, &analogdb.KeywordFilter{Limit: &limit}); err != nil {
		t.Fatal(err)
	}
	if got := inner.summaryCalls.Load(); got != 2 {
		t.Errorf("want a new query for another filter, got %d", got)
	}

	for range 2 {
		detail, err := s.FindKeyword(ctx, "new york", 6)
		if err != nil {
			t.Fatal(err)
		}
		if detail.Word != "new york" || detail.Count != 4 || len(detail.TopPosts) != 1 {
			t.Fatalf("unexpected detail %+v", detail)
		}
		shared, err := s.CoOccurring(ctx, "new york")
		if err != nil || shared["ocean"] != 3 {
			t.Fatalf("unexpected shared %v %v", shared, err)
		}
	}
	if got := inner.detailCalls.Load(); got != 1 {
		t.Errorf("want 1 detail query, got %d", got)
	}
	if got := inner.sharedCalls.Load(); got != 1 {
		t.Errorf("want 1 co-occurrence query, got %d", got)
	}
	if _, err := s.FindKeyword(ctx, "new york", 1); err != nil {
		t.Fatal(err)
	}
	if got := inner.detailCalls.Load(); got != 2 {
		t.Errorf("want a new query for other top posts, got %d", got)
	}

	for range 2 {
		if _, err := s.FindKeyword(ctx, "zzzz", 6); analogdb.ErrorCode(err) != analogdb.ERRNOTFOUND {
			t.Fatalf("want not found, got %v", err)
		}
	}
	if got := inner.detailCalls.Load(); got != 4 {
		t.Errorf("want not found not cached, got %d detail queries", got)
	}
}
