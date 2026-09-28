package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/evanofslack/analogdb"
)

var updateGolden = flag.Bool("update", false, "rewrite golden files")

type goldenResult struct {
	Count int              `json:"count"`
	Posts []*analogdb.Post `json:"posts"`
}

func TestPostService_FindPostsGolden(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()

	extra, err := os.ReadFile(filepath.Join("testdata", "golden_seed.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.db.Exec(string(extra)); err != nil {
		t.Fatalf("Seed golden posts, err=%v", err)
	}

	service := NewPostService(db)
	ctx := context.Background()

	filter := func(sort analogdb.PostSort, modify func(f *analogdb.PostFilter)) *analogdb.PostFilter {
		limit := 20
		f := analogdb.NewPostFilter(&limit, &sort, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		if modify != nil {
			modify(f)
		}
		f.SetMinColorPercent()
		return f
	}
	ptr := func(s string) *string { return &s }
	intPtr := func(i int) *int { return &i }
	boolPtr := func(b bool) *bool { return &b }

	cases := []struct {
		name   string
		filter *analogdb.PostFilter
	}{
		{"time", filter(analogdb.PostSortTime, nil)},
		{"time_limit_2", filter(analogdb.PostSortTime, func(f *analogdb.PostFilter) { f.Limit = intPtr(2) })},
		{"time_keyset", filter(analogdb.PostSortTime, func(f *analogdb.PostFilter) { f.Keyset = intPtr(1641427200) })},
		{"score", filter(analogdb.PostSortScore, nil)},
		{"score_keyset", filter(analogdb.PostSortScore, func(f *analogdb.PostFilter) { f.Keyset = intPtr(150) })},
		{"random_seed", filter(analogdb.PostSortRandom, func(f *analogdb.PostFilter) { f.Seed = intPtr(37) })},
		{"random_seed_keyset", filter(analogdb.PostSortRandom, func(f *analogdb.PostFilter) {
			f.Seed = intPtr(37)
			f.Keyset = intPtr(1641254400)
		})},
		{"keyword", filter(analogdb.PostSortTime, func(f *analogdb.PostFilter) { f.Keywords = &[]string{"portrait"} })},
		{"keywords_two", filter(analogdb.PostSortTime, func(f *analogdb.PostFilter) { f.Keywords = &[]string{"portrait", "street"} })},
		{"keyword_with_comma", filter(analogdb.PostSortTime, func(f *analogdb.PostFilter) { f.Keywords = &[]string{"sand, sun"} })},
		{"color", filter(analogdb.PostSortTime, func(f *analogdb.PostFilter) { f.Colors = &[]string{"teal"} })},
		{"color_min", filter(analogdb.PostSortTime, func(f *analogdb.PostFilter) {
			f.Colors = &[]string{"gray"}
			f.ColorPercents = &[]float64{0.2}
		})},
		{"colors_two", filter(analogdb.PostSortTime, func(f *analogdb.PostFilter) {
			f.Colors = &[]string{"navy", "black"}
			f.ColorPercents = &[]float64{0.5, 0.1}
		})},
		{"camera", filter(analogdb.PostSortTime, func(f *analogdb.PostFilter) {
			f.CameraMake = ptr("nikon")
			f.CameraModel = ptr("fm2")
		})},
		{"film", filter(analogdb.PostSortTime, func(f *analogdb.PostFilter) {
			f.FilmMake = ptr("kodak")
			f.FilmType = ptr("portra 400")
			f.FilmSpeed = intPtr(400)
		})},
		{"title", filter(analogdb.PostSortTime, func(f *analogdb.PostFilter) { f.Title = ptr("beach") })},
		{"author", filter(analogdb.PostSortTime, func(f *analogdb.PostFilter) { f.Author = ptr("streetphotographer") })},
		{"nsfw_false", filter(analogdb.PostSortTime, func(f *analogdb.PostFilter) { f.Nsfw = boolPtr(false) })},
		{"grayscale_sprocket", filter(analogdb.PostSortScore, func(f *analogdb.PostFilter) {
			f.Grayscale = boolPtr(false)
			f.Sprocket = boolPtr(false)
		})},
		{"time_range", filter(analogdb.PostSortTime, func(f *analogdb.PostFilter) {
			start, end := time.Unix(1641081600, 0), time.Unix(1641513600, 0)
			f.TimeStart = &start
			f.TimeEnd = &end
		})},
		{"ids", filter(analogdb.PostSortTime, func(f *analogdb.PostFilter) { f.IDs = &[]int{1, 5, 7} })},
		{"no_match", filter(analogdb.PostSortTime, func(f *analogdb.PostFilter) { f.Keywords = &[]string{"missing"} })},
	}

	results := map[string]goldenResult{}
	for _, c := range cases {
		posts, count, err := service.FindPosts(ctx, c.filter)
		if err != nil {
			t.Fatalf("%s: FindPosts failed: %v", c.name, err)
		}
		results[c.name] = goldenResult{Count: count, Posts: posts}
	}

	post, err := service.FindPostByID(ctx, 4)
	if err != nil {
		t.Fatal(err)
	}
	results["by_id"] = goldenResult{Count: 1, Posts: []*analogdb.Post{post}}

	got, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')

	path := filepath.Join("testdata", "find_posts_golden.json")
	if *updateGolden {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("findPosts output differs from %s, run with -update to inspect the diff\n%s", path, got)
	}
}
