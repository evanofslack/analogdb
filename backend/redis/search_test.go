package redis

import (
	"context"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/evanofslack/analogdb"
)

type fakeSearchService struct {
	textCalls  atomic.Int64
	imageCalls atomic.Int64
}

func (f *fakeSearchService) SearchText(ctx context.Context, filter *analogdb.SearchFilter) ([]analogdb.SearchHit, bool, error) {
	f.textCalls.Add(1)
	return []analogdb.SearchHit{{PostID: 4, Tags: []string{"cat"}}, {PostID: 2, Tags: []string{}}}, true, nil
}

func (f *fakeSearchService) SearchImage(ctx context.Context, filter *analogdb.SearchFilter) ([]analogdb.SearchHit, error) {
	f.imageCalls.Add(1)
	return []analogdb.SearchHit{{PostID: 7}}, nil
}

func TestSearchCache(t *testing.T) {
	rdb, _ := newMiniRDB(t)
	inner := &fakeSearchService{}
	s := NewCacheSearchService(rdb, inner)
	ctx := context.Background()

	filter := &analogdb.SearchFilter{Query: "cats", Expanded: "cats cat", Limit: 50}
	for range 2 {
		hits, hasMore, err := s.SearchText(ctx, filter)
		if err != nil {
			t.Fatal(err)
		}
		if !hasMore || len(hits) != 2 || hits[0].PostID != 4 || !slices.Equal(hits[0].Tags, []string{"cat"}) {
			t.Fatalf("unexpected hits %+v %v", hits, hasMore)
		}
	}
	if got := inner.textCalls.Load(); got != 1 {
		t.Errorf("want 1 text search, got %d", got)
	}

	next := *filter
	next.Offset = 50
	if _, _, err := s.SearchText(ctx, &next); err != nil {
		t.Fatal(err)
	}
	if got := inner.textCalls.Load(); got != 2 {
		t.Errorf("want a new search for another page, got %d", got)
	}

	image := &analogdb.SearchFilter{Image: []byte{1, 2, 3}, Limit: 50}
	for range 2 {
		if _, err := s.SearchImage(ctx, image); err != nil {
			t.Fatal(err)
		}
	}
	if got := inner.imageCalls.Load(); got != 2 {
		t.Errorf("want image searches not cached, got %d", got)
	}
}
