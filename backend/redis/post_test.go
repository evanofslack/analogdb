package redis

import (
	"context"
	"testing"

	"github.com/mitchellh/hashstructure/v2"

	"github.com/evanofslack/analogdb"
)

func TestPostFilterHashIncludesCursorAndSeed(t *testing.T) {
	filter := func(seed int, cursor *analogdb.Cursor) *analogdb.PostFilter {
		limit := 21
		sort := analogdb.PostSortRandom
		f := analogdb.NewPostFilter(&limit, &sort, nil, nil, nil, &seed, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		f.Cursor = cursor
		return f
	}
	ascending := func(f *analogdb.PostFilter) *analogdb.PostFilter {
		asc := analogdb.PostOrderAsc
		f.Order = &asc
		return f
	}
	hash := func(f *analogdb.PostFilter) uint64 {
		h, err := hashstructure.Hash(f, hashstructure.FormatV2, nil)
		if err != nil {
			t.Fatal(err)
		}
		return h
	}

	base := hash(filter(37, nil))
	if base != hash(filter(37, nil)) {
		t.Fatal("expected equal filters to hash the same")
	}
	cases := map[string]*analogdb.PostFilter{
		"seed":        filter(38, nil),
		"cursor":      filter(37, &analogdb.Cursor{Hash: "8f", ID: 3}),
		"cursor id":   filter(37, &analogdb.Cursor{Hash: "8f", ID: 4}),
		"cursor hash": filter(37, &analogdb.Cursor{Hash: "90", ID: 3}),
		"order":       ascending(filter(37, nil)),
		"cursor rank": filter(37, &analogdb.Cursor{Rank: 0.25, ID: 3}),
		"color sort": func() *analogdb.PostFilter {
			f := filter(37, nil)
			sort := analogdb.PostSortColor
			f.Sort = &sort
			return f
		}(),
		"colors": func() *analogdb.PostFilter {
			f := filter(37, nil)
			f.Colors = &[]string{"bb3333"}
			f.SetMinColorPercent()
			return f
		}(),
	}
	seen := map[uint64]string{base: "base"}
	for name, f := range cases {
		h := hash(f)
		if other, ok := seen[h]; ok {
			t.Errorf("%s hashes the same as %s", name, other)
		}
		seen[h] = name
	}
}

type rankedPostService struct {
	*fakePostService
}

// FindPosts gives posts a sort rank and counts fewer posts for the rainbow
func (f *rankedPostService) FindPosts(ctx context.Context, filter *analogdb.PostFilter) ([]*analogdb.Post, int, error) {
	posts, count, err := f.fakePostService.FindPosts(ctx, filter)
	for _, p := range posts {
		p.SortRank = float64(p.Id) + 0.125
	}
	if filter.IsRainbow() {
		count--
	}
	return posts, count, err
}

func TestFindPostsCachesSortRank(t *testing.T) {
	ctx := context.Background()
	rdb, _ := newMiniRDB(t)
	db := &rankedPostService{newFakePostService(1, 2)}
	s := NewCachePostService(rdb, db)

	limit := 10
	timeSort, colorSort := analogdb.PostSortTime, analogdb.PostSortColor
	byTime := analogdb.NewPostFilter(&limit, &timeSort, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	rainbow := analogdb.NewPostFilter(&limit, &colorSort, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	check := func() {
		t.Helper()
		posts, count, err := s.FindPosts(ctx, rainbow)
		if err != nil {
			t.Fatal(err)
		}
		if count != 1 || posts[1].SortRank != 2.125 {
			t.Fatalf("expected count 1 and rank 2.125, got %d %v", count, posts[1].SortRank)
		}
	}
	check()
	check()
	if got := db.findCalls.Load(); got != 1 {
		t.Fatalf("expected the rainbow page to be cached, got %d db calls", got)
	}
	if _, count, err := s.FindPosts(ctx, byTime); err != nil || count != 2 {
		t.Fatalf("expected the time count apart from the rainbow count, got %d %v", count, err)
	}
	check()
}
