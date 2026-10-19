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
	hue, chroma, strength, rainbow sql.NullFloat64
}

func mustPostColor(t *testing.T, db *DB, id int) postColorRow {
	t.Helper()
	var row postColorRow
	if err := db.db.QueryRow("SELECT color_hue, color_chroma, color_strength, rainbow_key FROM pictures WHERE id = $1", id).Scan(&row.hue, &row.chroma, &row.strength, &row.rainbow); err != nil {
		t.Fatal(err)
	}
	return row
}

// checkPostColor compares the stored color columns of a post with colorspace
func checkPostColor(t *testing.T, db *DB, id int, palette []colorspace.Swatch, grayscale bool) postColorRow {
	t.Helper()
	row := mustPostColor(t, db, id)
	c, ok, err := colorspace.PalettePostColor(palette, grayscale)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		if row.hue.Valid || row.chroma.Valid || row.strength.Valid || row.rainbow.Valid {
			t.Errorf("post %d: expected no rainbow color, got %+v", id, row)
		}
		return row
	}
	if !row.strength.Valid || float32(row.strength.Float64) != float32(c.Strength) || !row.rainbow.Valid || row.rainbow.Float64 != colorspace.RainbowKey(id, c) {
		t.Errorf("post %d: expected strength %v and key %v, got %+v", id, float32(c.Strength), colorspace.RainbowKey(id, c), row)
	}
	if c.Neutral {
		if row.hue.Valid || row.chroma.Valid {
			t.Errorf("post %d: expected no hue for a neutral post, got %+v", id, row)
		}
		// black ends its loop where the next one starts
		w := colorspace.RainbowHueWidth + colorspace.RainbowNeutralWidth
		if loop := int(math.Ceil(row.rainbow.Float64/w - 1)); loop < 0 || loop >= colorspace.RainbowNeutralLoops(c.Bucket) {
			t.Errorf("post %d: loop %d out of the %d loops of %s", id, loop, colorspace.RainbowNeutralLoops(c.Bucket), c.Bucket)
		}
	} else if float32(row.hue.Float64) != float32(c.Hue) || float32(row.chroma.Float64) != float32(c.Chroma) {
		t.Errorf("post %d: expected hue %v chroma %v, got %+v", id, c.Hue, c.Chroma, row)
	}
	return row
}

func swatches(colors []analogdb.Color) []colorspace.Swatch {
	out := []colorspace.Swatch{}
	for _, c := range colors {
		out = append(out, colorspace.Swatch{Hex: c.Hex, Percent: c.Percent})
	}
	return out
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

	row := checkPostColor(t, db, post.Id, swatches(colors), false)
	if !row.hue.Valid || float32(row.hue.Float64) != float32(d.Hue) || float32(row.strength.Float64) != float32(0.3*d.Chroma) {
		t.Fatalf("expected the firebrick cluster as representative, got %+v", row)
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
		if row := checkPostColor(t, db, post.Id, swatches(patch), false); float32(row.hue.Float64) != float32(blue.Hue) {
			t.Errorf("expected the blue hue, got %+v", row)
		}
	})

	t.Run("grayscale clears the representative", func(t *testing.T) {
		yes, no := true, false
		if err := service.PatchPost(ctx, &analogdb.PatchPost{Grayscale: &yes}, post.Id); err != nil {
			t.Fatal(err)
		}
		if row := mustPostColor(t, db, post.Id); row.hue.Valid || row.chroma.Valid || row.strength.Valid || row.rainbow.Valid {
			t.Errorf("expected no rainbow color for a grayscale blue post, got %+v", row)
		}
		if err := service.PatchPost(ctx, &analogdb.PatchPost{Grayscale: &no}, post.Id); err != nil {
			t.Fatal(err)
		}
		if row := mustPostColor(t, db, post.Id); !row.hue.Valid || !row.rainbow.Valid {
			t.Errorf("expected the representative back, got %+v", row)
		}
	})

	t.Run("neutral and grayscale posts", func(t *testing.T) {
		cases := []struct {
			name      string
			colors    []analogdb.Color
			grayscale bool
			ok        bool
		}{
			{"neutral white", []analogdb.Color{{Hex: "#ffffff", Percent: 0.5}, {Hex: "#808080", Percent: 0.5}}, false, true},
			{"neutral gray", []analogdb.Color{{Hex: "#808080", Percent: 0.7}, {Hex: "#000000", Percent: 0.3}}, false, true},
			{"neutral black", []analogdb.Color{{Hex: "#000000", Percent: 0.6}, {Hex: "#808080", Percent: 0.4}}, false, true},
			{"neutral below minimum", []analogdb.Color{{Hex: "#808080", Percent: 0.6}, {Hex: "#000000", Percent: 0.4}}, false, false},
			{"grayscale black", []analogdb.Color{{Hex: "#ff0000", Percent: 0.4}, {Hex: "#000000", Percent: 0.6}}, true, true},
			{"grayscale red", []analogdb.Color{{Hex: "#ff0000", Percent: 1}}, true, false},
		}
		tails := map[string]float64{}
		for _, tc := range cases {
			p := createColorPost(ctx, t, service, tc.name, tc.colors, tc.grayscale)
			row := checkPostColor(t, db, p.Id, swatches(tc.colors), tc.grayscale)
			if row.rainbow.Valid != tc.ok {
				t.Errorf("%s: expected in the rainbow %t, got %+v", tc.name, tc.ok, row)
			}
			if tc.ok {
				// black ends its loop where the next one starts
				w := colorspace.RainbowHueWidth + colorspace.RainbowNeutralWidth
				place := row.rainbow.Float64 - math.Ceil(row.rainbow.Float64/w-1)*w
				if place < colorspace.RainbowHueWidth {
					t.Errorf("%s: expected a place in the neutral tail, got %v", tc.name, place)
				}
				tails[tc.name] = place
			}
		}
		if !(tails["neutral white"] < tails["neutral gray"] && tails["neutral gray"] < tails["neutral black"]) {
			t.Errorf("expected white before gray before black in the tail, got %v", tails)
		}

		empty := []analogdb.Color{}
		if err := service.PatchPost(ctx, &analogdb.PatchPost{Colors: &empty}, post.Id); err != nil {
			t.Fatal(err)
		}
		if row := mustPostColor(t, db, post.Id); row.hue.Valid || row.strength.Valid || row.rainbow.Valid {
			t.Errorf("expected no rainbow color without colors, got %+v", row)
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
		c, ok, err := colorspace.PalettePostColor(p.palette, p.grayscale)
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			out = append(out, keyed{p.id, colorspace.RainbowKey(p.id, c)})
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
	neutrals := [][]analogdb.Color{
		{{Hex: "#ffffff", Percent: 0.6}, {Hex: "#808080", Percent: 0.4}},
		{{Hex: "#808080", Percent: 0.8}, {Hex: "#ffffff", Percent: 0.2}},
		{{Hex: "#000000", Percent: 0.7}, {Hex: "#808080", Percent: 0.3}},
		{{Hex: "#808080", Percent: 0.5}, {Hex: "#000000", Percent: 0.45}},
	}
	for i := range 48 {
		colors := colorTestPalette(i)
		if i >= 40 {
			colors = neutrals[i%len(neutrals)]
		}
		grayscale := i%9 == 0
		post := createColorPost(ctx, t, service, fmt.Sprintf("color_%d", i), colors, grayscale)
		posts = append(posts, colorTestPost{id: post.Id, grayscale: grayscale, palette: swatches(colors)})
		checkPostColor(t, db, post.Id, swatches(colors), grayscale)
	}

	colorSort := analogdb.PostSortColor
	query := func(colors []string, limit int, cursor *analogdb.Cursor) ([]*analogdb.Post, int) {
		t.Helper()
		filter := analogdb.NewPostFilter(&limit, &colorSort, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
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
		filter := analogdb.NewPostFilter(&limit, &colorSort, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
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
		filter := analogdb.NewPostFilter(&limit, &timeSort, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
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

type colorSnapshot struct {
	colors map[int][7]any
	posts  map[int][4]any
}

func mustColorSnapshot(t *testing.T, db *DB) colorSnapshot {
	t.Helper()
	snap := colorSnapshot{colors: map[int][7]any{}, posts: map[int][4]any{}}
	mustScanRows(t, db, "SELECT id, html, css, oklab_l, oklab_a, oklab_b, chroma, hue FROM colors", func(rows *sql.Rows) error {
		var id int
		var html, css sql.NullString
		var l, a, b, chroma, hue sql.NullFloat64
		err := rows.Scan(&id, &html, &css, &l, &a, &b, &chroma, &hue)
		snap.colors[id] = [7]any{html, css, l, a, b, chroma, hue}
		return err
	})
	mustScanRows(t, db, "SELECT id, color_hue, color_chroma, color_strength, rainbow_key FROM pictures", func(rows *sql.Rows) error {
		var id int
		var hue, chroma, strength, key sql.NullFloat64
		err := rows.Scan(&id, &hue, &chroma, &strength, &key)
		snap.posts[id] = [4]any{hue, chroma, strength, key}
		return err
	})
	return snap
}

func mustScanRows(t *testing.T, db *DB, query string, scan func(*sql.Rows) error) {
	t.Helper()
	rows, err := db.db.Query(query)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := scan(rows); err != nil {
			t.Fatal(err)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestColorService_Rederive(t *testing.T) {
	db, cleanup := mustOpen(t)
	defer cleanup()
	service := NewPostService(db)
	colors := NewColorService(db)
	ctx := context.Background()

	type created struct {
		id        int
		grayscale bool
		palette   []colorspace.Swatch
	}
	posts := []created{}
	create := func(i int) {
		c := colorTestPalette(i)
		grayscale := i%5 == 0
		p := createColorPost(ctx, t, service, fmt.Sprintf("rederive_%d", i), c, grayscale)
		posts = append(posts, created{p.Id, grayscale, swatches(c)})
	}
	for i := range 20 {
		create(i)
	}
	// later posts fall in other batches
	if _, err := db.db.Exec("SELECT setval(pg_get_serial_sequence('pictures', 'id'), 1700)"); err != nil {
		t.Fatal(err)
	}
	for i := 20; i < 25; i++ {
		create(i)
	}
	want := mustColorSnapshot(t, db)

	result, err := colors.RederiveColors(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.Posts != len(posts) || result.Colors != 3*len(posts) || result.PostsUpdated != 0 || result.ColorsUpdated != 0 {
		t.Fatalf("expected nothing to change on fresh posts, got %+v", result)
	}

	corrupt := []string{
		"UPDATE colors SET html = 'red', css = 'nope', oklab_l = NULL, chroma = 0, hue = NULL",
		"UPDATE pictures SET color_hue = NULL, color_chroma = 1, color_strength = NULL, rainbow_key = NULL",
	}
	for _, q := range corrupt {
		if _, err := db.db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	result, err = colors.RederiveColors(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.PostsUpdated != len(posts) || result.ColorsUpdated != 3*len(posts) {
		t.Errorf("expected every row updated, got %+v", result)
	}
	if got := mustColorSnapshot(t, db); !reflect.DeepEqual(got, want) {
		t.Errorf("expected the derived columns restored\nwant %v\ngot  %v", want, got)
	}
	keys := 0
	for _, p := range posts {
		if checkPostColor(t, db, p.id, p.palette, p.grayscale).rainbow.Valid {
			keys++
		}
	}
	if keys == 0 {
		t.Error("expected rainbow keys")
	}

	// a grayscale change made outside the API is picked up
	if _, err := db.db.Exec("UPDATE pictures SET greyscale = NOT greyscale WHERE id = $1", posts[1].id); err != nil {
		t.Fatal(err)
	}
	if _, err := colors.RederiveColors(ctx); err != nil {
		t.Fatal(err)
	}
	checkPostColor(t, db, posts[1].id, posts[1].palette, !posts[1].grayscale)
}
