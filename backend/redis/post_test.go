package redis

import (
	"testing"

	"github.com/mitchellh/hashstructure/v2"

	"github.com/evanofslack/analogdb"
)

func TestPostFilterHashIncludesCursorAndSeed(t *testing.T) {
	filter := func(seed int, cursor *analogdb.Cursor) *analogdb.PostFilter {
		limit := 21
		sort := analogdb.PostSortRandom
		f := analogdb.NewPostFilter(&limit, &sort, nil, nil, nil, nil, &seed, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		f.Cursor = cursor
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
