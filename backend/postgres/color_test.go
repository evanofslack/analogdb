package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"reflect"
	"sort"
	"testing"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/colorspace"
)

var colorTestHexes = []string{"#ff0000", "#c0392b", "#008080", "#0000ff", "#808080", "#ffffff", "#ffa500", "#8b4513", "#d2b48c", "#800080"}

// colorTestPalette gives post i three clusters, with many repeated shares to test ties
func colorTestPalette(i int) []analogdb.Color {
	n := len(colorTestHexes)
	return []analogdb.Color{
		{Hex: colorTestHexes[i%n], Css: "ignored", Html: "ignored", Percent: 0.5},
		{Hex: colorTestHexes[(i*3+1)%n], Percent: 0.3},
		{Hex: colorTestHexes[(i*7+2)%n], Percent: 0.2},
	}
}

func createColorPost(ctx context.Context, t *testing.T, service *PostService, name string, colors []analogdb.Color, grayscale bool) *analogdb.Post {
	t.Helper()
	post, err := service.CreatePost(ctx, &analogdb.CreatePost{
		Title:     name,
		Author:    "u/colors",
		Permalink: "reddit.com/" + name,
		Grayscale: grayscale,
		Time:      1650000000,
		Images: []analogdb.Image{
			{Label: "low", Url: name + "-low.jpg"},
			{Label: "medium", Url: name + "-med.jpg"},
			{Label: "high", Url: name + "-high.jpg"},
			{Label: "raw", Url: name + "-raw.jpg"},
		},
		Colors: colors,
	})
	if err != nil {
		t.Fatalf("CreatePost %s failed: %v", name, err)
	}
	return post
}

type postColorRow struct {
	hue, chroma, rainbow sql.NullFloat64
}

func mustPostColor(t *testing.T, db *DB, id int) postColorRow {
	t.Helper()
	var row postColorRow
	if err := db.db.QueryRow("SELECT color_hue, color_chroma, rainbow_key FROM pictures WHERE id = $1", id).Scan(&row.hue, &row.chroma, &row.rainbow); err != nil {
		t.Fatal(err)
	}
	return row
}

func TestPostService_ColorsDerived(t *testing.T) {
	db, cleanup := mustOpenWithSeed(t)
	defer cleanup()
	service := NewPostService(db)
	ctx := context.Background()

	colors := []analogdb.Color{
		{Hex: "#808080", Css: "client", Html: "client", Percent: 0.6},
		{Hex: "#C0392B", Css: "client", Html: "client", Percent: 0.3},
		{Hex: "008080", Percent: 0.1},
	}
	post := createColorPost(ctx, t, service, "derived", colors, false)

	wantNames := [][2]string{{"gray", "gray"}, {"firebrick", "red"}, {"teal", "teal"}}
	check := func(got []analogdb.Color) {
		t.Helper()
		for i, c := range got {
			if c.Css != wantNames[i][0] || c.Html != wantNames[i][1] {
				t.Errorf("color %d: expected %v, got %s %s", i, wantNames[i], c.Css, c.Html)
			}
		}
	}
	check(post.Colors)
	found, err := service.FindPostByID(ctx, post.Id)
	if err != nil {
		t.Fatal(err)
	}
	check(found.Colors)

	var l, a, b, chroma, hue float32
	if err := db.db.QueryRow("SELECT oklab_l, oklab_a, oklab_b, chroma, hue FROM colors WHERE post_id = $1 AND hex = '#C0392B'", post.Id).Scan(&l, &a, &b, &chroma, &hue); err != nil {
		t.Fatal(err)
	}
	d, _ := colorspace.Derive("c0392b")
	if l != float32(d.L) || a != float32(d.A) || b != float32(d.B) || chroma != float32(d.Chroma) || hue != float32(d.Hue) {
		t.Errorf("unexpected derived columns %v %v %v %v %v", l, a, b, chroma, hue)
	}

	row := mustPostColor(t, db, post.Id)
	if !row.hue.Valid || float32(row.hue.Float64) != float32(d.Hue) || float32(row.chroma.Float64) != float32(d.Chroma) {
		t.Fatalf("expected the firebrick cluster as representative, got %+v", row)
	}
	if want := colorspace.RainbowKey(post.Id, d.Hue); row.rainbow.Float64 != want {
		t.Errorf("expected rainbow key %v, got %v", want, row.rainbow.Float64)
	}

	t.Run("patch colors", func(t *testing.T) {
		patch := []analogdb.Color{{Hex: "#0000ff", Css: "client", Html: "red", Percent: 0.9}}
		if err := service.PatchPost(ctx, &analogdb.PatchPost{Colors: &patch}, post.Id); err != nil {
			t.Fatal(err)
		}
		found, err := service.FindPostByID(ctx, post.Id)
		if err != nil {
			t.Fatal(err)
		}
		if len(found.Colors) != 1 || found.Colors[0].Html != "navy" || found.Colors[0].Css != "blue" {
			t.Errorf("expected a derived navy color, got %+v", found.Colors)
		}
		blue, _ := colorspace.Derive("0000ff")
		if row := mustPostColor(t, db, post.Id); float32(row.hue.Float64) != float32(blue.Hue) || row.rainbow.Float64 != colorspace.RainbowKey(post.Id, blue.Hue) {
			t.Errorf("expected the blue hue, got %+v", row)
		}
	})

	t.Run("grayscale clears the representative", func(t *testing.T) {
		yes, no := true, false
		if err := service.PatchPost(ctx, &analogdb.PatchPost{Grayscale: &yes}, post.Id); err != nil {
			t.Fatal(err)
		}
		if row := mustPostColor(t, db, post.Id); row.hue.Valid || row.chroma.Valid || row.rainbow.Valid {
			t.Errorf("expected no representative for grayscale, got %+v", row)
		}
		if err := service.PatchPost(ctx, &analogdb.PatchPost{Grayscale: &no}, post.Id); err != nil {
			t.Fatal(err)
		}
		if row := mustPostColor(t, db, post.Id); !row.hue.Valid || !row.rainbow.Valid {
			t.Errorf("expected the representative back, got %+v", row)
		}
	})

	t.Run("neutral and grayscale posts", func(t *testing.T) {
		neutral := createColorPost(ctx, t, service, "neutral", []analogdb.Color{{Hex: "#808080", Percent: 0.7}, {Hex: "#000000", Percent: 0.3}}, false)
		if row := mustPostColor(t, db, neutral.Id); row.hue.Valid || row.rainbow.Valid {
			t.Errorf("expected no representative for a neutral palette, got %+v", row)
		}
		gray := createColorPost(ctx, t, service, "grayscale", []analogdb.Color{{Hex: "#ff0000", Percent: 1}}, true)
		if row := mustPostColor(t, db, gray.Id); row.hue.Valid || row.rainbow.Valid {
			t.Errorf("expected no representative for a grayscale post, got %+v", row)
		}
		empty := []analogdb.Color{}
		if err := service.PatchPost(ctx, &analogdb.PatchPost{Colors: &empty}, post.Id); err != nil {
			t.Fatal(err)
		}
		if row := mustPostColor(t, db, post.Id); row.hue.Valid {
			t.Errorf("expected no representative without colors, got %+v", row)
		}
	})

	t.Run("invalid colors", func(t *testing.T) {
		bad := map[string][]analogdb.Color{
			"hex":      {{Hex: "#ff00", Percent: 0.5}},
			"empty":    {{Hex: "", Percent: 0.5}},
			"negative": {{Hex: "#ff0000", Percent: -0.1}},
			"sum":      {{Hex: "#ff0000", Percent: 0.7}, {Hex: "#00ff00", Percent: 0.4}},
		}
		for name, colors := range bad {
			_, err := service.CreatePost(ctx, &analogdb.CreatePost{Permalink: "bad_" + name, Images: make([]analogdb.Image, 4), Colors: colors})
			if code := analogdb.ErrorCode(err); code != analogdb.ERRUNPROCESSABLE {
				t.Errorf("create %s: expected %s, got %v", name, analogdb.ERRUNPROCESSABLE, err)
			}
			err = service.PatchPost(ctx, &analogdb.PatchPost{Colors: &colors}, post.Id)
			if code := analogdb.ErrorCode(err); code != analogdb.ERRUNPROCESSABLE {
				t.Errorf("patch %s: expected %s, got %v", name, analogdb.ERRUNPROCESSABLE, err)
			}
		}
	})
}

type colorTestPost struct {
	id        int
	grayscale bool
	palette   []colorspace.Swatch
}

// share8 rounds a share like a NUMERIC(9, 8) sum cast to float8
func share8(x float64) float64 {
	return math.Round(x*1e8) / 1e8
}

// expectedColorOrder ranks posts in Go the way sort=color should
func expectedColorOrder(t *testing.T, posts []colorTestPost, colors []string) []int {
	t.Helper()
	type ranked struct {
		id   int
		rank float64
	}
	out := []ranked{}
	for _, p := range posts {
		buckets, err := colorspace.BucketShares(p.palette)
		if err != nil {
			t.Fatal(err)
		}
		rank, ok := math.Inf(1), true
		for _, c := range colors {
			share := buckets[c]
			if !colorspace.IsBucket(c) {
				if share, err = colorspace.HexShare(p.palette, c, colorspace.MatchRadius); err != nil {
					t.Fatal(err)
				}
			}
			share = share8(share)
			if share <= colorspace.DefaultThreshold(c, len(colors)) {
				ok = false
			}
			rank = math.Min(rank, share)
		}
		if ok {
			out = append(out, ranked{p.id, rank})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].rank != out[j].rank {
			return out[i].rank > out[j].rank
		}
		return out[i].id > out[j].id
	})
	ids := []int{}
	for _, r := range out {
		ids = append(ids, r.id)
	}
	return ids
}

func expectedRainbowOrder(t *testing.T, posts []colorTestPost) []int {
	t.Helper()
	type keyed struct {
		id  int
		key float64
	}
	out := []keyed{}
	for _, p := range posts {
		hue, _, ok, err := colorspace.PaletteRepresentative(p.palette)
		if err != nil {
			t.Fatal(err)
		}
		if ok && !p.grayscale {
			out = append(out, keyed{p.id, colorspace.RainbowKey(p.id, hue)})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].key != out[j].key {
			return out[i].key < out[j].key
		}
		return out[i].id < out[j].id
	})
	ids := []int{}
	for _, k := range out {
		ids = append(ids, k.id)
	}
	return ids
}

func TestPostService_ColorSort(t *testing.T) {
	db, cleanup := mustOpen(t)
	defer cleanup()
	service := NewPostService(db)
	ctx := context.Background()

	posts := []colorTestPost{}
	for i := range 40 {
		colors := colorTestPalette(i)
		grayscale := i%9 == 0
		post := createColorPost(ctx, t, service, fmt.Sprintf("color_%d", i), colors, grayscale)
		palette := []colorspace.Swatch{}
		for _, c := range colors {
			palette = append(palette, colorspace.Swatch{Hex: c.Hex, Percent: c.Percent})
		}
		posts = append(posts, colorTestPost{id: post.Id, grayscale: grayscale, palette: palette})
	}

	colorSort := analogdb.PostSortColor
	query := func(colors []string, limit int, cursor *analogdb.Cursor) ([]*analogdb.Post, int) {
		t.Helper()
		filter := analogdb.NewPostFilter(&limit, &colorSort, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		if colors != nil {
			filter.Colors = &colors
			filter.SetMinColorPercent()
		}
		filter.Cursor = cursor
		got, count, err := service.FindPosts(ctx, filter)
		if err != nil {
			t.Fatal(err)
		}
		return got, count
	}
	ids := func(posts []*analogdb.Post) []int {
		out := []int{}
		for _, p := range posts {
			out = append(out, p.Id)
		}
		return out
	}

	cases := []struct {
		name   string
		colors []string
	}{
		{"named", []string{"red"}},
		{"named teal", []string{"teal"}},
		{"hex", []string{"bb3333"}},
		{"hex white", []string{"ffffff"}},
		{"two names", []string{"red", "teal"}},
		{"name and hex", []string{"gray", "ff0000"}},
		{"three", []string{"brown", "red", "teal"}},
		{"rainbow", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var want []int
			if tc.colors == nil {
				want = expectedRainbowOrder(t, posts)
			} else {
				want = expectedColorOrder(t, posts, tc.colors)
			}
			if len(want) < 3 {
				t.Fatalf("expected at least 3 matches to page through, got %v", want)
			}

			full, count := query(tc.colors, 1000, nil)
			if got := ids(full); !reflect.DeepEqual(got, want) {
				t.Fatalf("expected order %v, got %v", want, got)
			}
			if count != len(want) {
				t.Errorf("expected count %d, got %d", len(want), count)
			}
			for i := 1; i < len(full); i++ {
				prev, cur := full[i-1].SortRank, full[i].SortRank
				if tc.colors == nil && prev > cur || tc.colors != nil && prev < cur {
					t.Fatalf("sort rank out of order at %d: %v then %v", i, prev, cur)
				}
			}

			for _, size := range []int{1, 2, 7} {
				walked := []int{}
				var cursor *analogdb.Cursor
				for page := 0; ; page++ {
					if page > len(want)+1 {
						t.Fatal("walk did not end")
					}
					got, pageCount := query(tc.colors, size, cursor)
					if pageCount != count {
						t.Fatalf("count changed from %d to %d", count, pageCount)
					}
					walked = append(walked, ids(got)...)
					if len(got) < size {
						break
					}
					last := got[len(got)-1]
					cursor = &analogdb.Cursor{Rank: last.SortRank, ID: last.Id}
				}
				if !reflect.DeepEqual(walked, want) {
					t.Errorf("size %d: walk %v differs from %v", size, walked, want)
				}
			}
		})
	}

	t.Run("min_color override", func(t *testing.T) {
		limit := 1000
		colors, min := []string{"red"}, []float64{0.6}
		filter := analogdb.NewPostFilter(&limit, &colorSort, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		filter.Colors, filter.ColorPercents = &colors, &min
		filter.SetMinColorPercent()
		got, _, err := service.FindPosts(ctx, filter)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range got {
			if p.SortRank <= 0.6 {
				t.Errorf("post %d has rank %v under the override", p.Id, p.SortRank)
			}
		}
		if len(got) == 0 {
			t.Error("expected posts over the override")
		}
	})

	t.Run("time sort with a color", func(t *testing.T) {
		limit := 1000
		timeSort := analogdb.PostSortTime
		colors := []string{"red"}
		filter := analogdb.NewPostFilter(&limit, &timeSort, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		filter.Colors = &colors
		filter.SetMinColorPercent()
		got, count, err := service.FindPosts(ctx, filter)
		if err != nil {
			t.Fatal(err)
		}
		want := expectedColorOrder(t, posts, colors)
		if count != len(want) || len(got) != len(want) {
			t.Errorf("expected %d posts, got %d and count %d", len(want), len(got), count)
		}
		if got[0].Id < got[len(got)-1].Id {
			t.Error("expected newest first")
		}
	})
}
