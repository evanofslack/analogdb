package stats

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"slices"
	"testing"
	"time"

	"github.com/evanofslack/analogdb"
)

var (
	testNow   = time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	testStart = time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
)

func basis(i int) []float32 {
	v := make([]float32, analogdb.StatsVectorDims)
	v[i] = 1
	return v
}

// noisy is a unit vector near axis i, varied by seed so films have spread
func noisy(i int, rng *rand.Rand) []float32 {
	v := basis(i)
	for j := range 8 {
		v[100+j] = float32(rng.NormFloat64() * 0.2)
	}
	return unit(v)
}

func testSeed(hidden ...string) *analogdb.StatsSeed {
	border := analogdb.StatsRoleBorder
	seed := &analogdb.StatsSeed{Version: "v1", Model: "test"}
	for i := range 4 {
		key := fmt.Sprintf("t%02d", i)
		seed.Themes = append(seed.Themes, analogdb.StatsTheme{
			Key: key, Name: "theme " + key, Hidden: slices.Contains(hidden, key), Center: basis(i),
		})
	}
	seed.Themes[3].Role = &border
	for i, key := range analogdb.StatsPromptKeys {
		seed.Prompts = append(seed.Prompts, analogdb.StatsPrompt{Key: key, Label: key, Text: key, Vector: basis(100 + i)})
	}
	return seed
}

type testFilm struct {
	make, typ, colorType string
	theme                int
	hue                  float64
	posts                int
}

var testFilms = []testFilm{
	{"kodak", "portra 400", "color", 0, 60, 120},
	{"kodak", "gold 200", "color", 0, 80, 110},
	{"cinestill", "800t", "color", 1, 240, 100},
	{"kodak", "ektar 100", "color", 2, 200, 90},
	{"ilford", "hp5 plus", "bw", 2, 0, 100},
}

// testInput makes posts for the test films, spread over the window, with
// colors, keywords and vectors. Every film has one prolific author.
func testInput(rng *rand.Rand) (*analogdb.StatsInput, map[int][]float32) {
	in := &analogdb.StatsInput{}
	vecs := map[int][]float32{}
	id := 0
	span := testNow.Unix() - testStart.Unix()
	for fi, f := range testFilms {
		for i := range f.posts {
			id++
			author := fmt.Sprintf("author-%d-%d", fi, i)
			if i%4 == 0 {
				author = fmt.Sprintf("prolific-%d", fi)
			}
			grey := f.colorType == "bw"
			p := analogdb.StatsInputPost{
				ID:            id,
				Author:        author,
				Time:          testStart.Unix() + rng.Int63n(span),
				Score:         rng.Intn(1000),
				Nsfw:          i%10 == 0,
				Greyscale:     grey,
				FilmSlug:      analogdb.Slug(f.make, f.typ),
				FilmMake:      f.make,
				FilmType:      f.typ,
				FilmColorType: f.colorType,
				CameraSlug:    "nikon-fm2",
				CameraMake:    "nikon",
				CameraModel:   "fm2",
				FocalLength:   50,
				Aperture:      "f/2",
				Title:         "post",
				Popularity:    rng.Float64(),
			}
			if grey {
				p.Popularity *= 0.5
			}
			in.Posts = append(in.Posts, p)
			chroma := 0.08
			if grey {
				chroma = 0.002
			}
			in.Colors = append(in.Colors,
				analogdb.StatsInputColor{PostID: id, Percent: 0.6, L: 0.5, A: 0.05, B: 0.05, Chroma: chroma, Hue: f.hue},
				analogdb.StatsInputColor{PostID: id, Percent: 0.4, L: 0.7, A: 0.01, B: 0.01, Chroma: 0.01, Hue: f.hue},
			)
			if f.typ == "800t" || i%5 == 0 {
				in.Keywords = append(in.Keywords, analogdb.StatsInputKeyword{PostID: id, Word: "neon"})
			}
			vecs[id] = noisy(f.theme, rng)
		}
	}
	return in, vecs
}

func build(t *testing.T, seed *analogdb.StatsSeed) *analogdb.Stats {
	t.Helper()
	in, vecs := testInput(rand.New(rand.NewSource(1)))
	stats, err := Build(in, vecs, seed, testNow)
	if err != nil {
		t.Fatal(err)
	}
	return stats
}

func film(t *testing.T, stats *analogdb.Stats, slug string) analogdb.StatsFilm {
	t.Helper()
	for _, f := range stats.Films {
		if f.Slug == slug {
			return f
		}
	}
	t.Fatalf("film %s missing", slug)
	return analogdb.StatsFilm{}
}

func TestBuildErrors(t *testing.T) {
	in, vecs := testInput(rand.New(rand.NewSource(1)))
	if _, err := Build(&analogdb.StatsInput{}, vecs, testSeed(), testNow); err == nil {
		t.Error("want an error without posts")
	}
	if _, err := Build(in, vecs, nil, testNow); err == nil {
		t.Error("want an error without a seed")
	}
	seed := testSeed()
	seed.Prompts = seed.Prompts[1:]
	if _, err := Build(in, vecs, seed, testNow); err == nil {
		t.Error("want an error without the base prompt")
	}
}

func TestFilmLooks(t *testing.T) {
	stats := build(t, testSeed())

	if len(stats.Films) != len(testFilms) {
		t.Fatalf("want %d films, got %d", len(testFilms), len(stats.Films))
	}
	if stats.Films[0].Slug != "kodak-portra-400" {
		t.Errorf("want films by post count, got %s first", stats.Films[0].Slug)
	}

	for _, f := range stats.Films {
		if f.ColorType == "bw" {
			if len(f.Hue) != 0 || f.Bands != nil || len(f.Palette) != 0 {
				t.Errorf("%s: want empty color fields for B&W, got %v %v %v", f.Slug, f.Hue, f.Bands, f.Palette)
			}
			continue
		}
		var sum float64
		for _, h := range f.Hue {
			sum += h
		}
		if len(f.Hue) != HueBins || math.Abs(sum-1) > 0.01 {
			t.Errorf("%s: want %d hue bins summing to 1, got %d summing to %v", f.Slug, HueBins, len(f.Hue), sum)
		}
		b := f.Bands
		if total := b.Warm + b.Green + b.Cool + b.Magenta; math.Abs(total-1) > 0.01 {
			t.Errorf("%s: want bands summing to 1, got %v", f.Slug, total)
		}
		if len(f.Palette) == 0 || f.Palette[0][0] != '#' {
			t.Errorf("%s: want a hex palette, got %v", f.Slug, f.Palette)
		}
		if len(f.Typical) == 0 {
			t.Errorf("%s: want typical posts", f.Slug)
		}
		for _, p := range f.Typical {
			if p.Nsfw {
				t.Errorf("%s: typical post %d is NSFW", f.Slug, p.ID)
			}
		}
	}

	var styles analogdb.StatsStyles
	for _, f := range stats.Films {
		s := f.Styles
		styles.Vivid += math.Abs(s.Vivid)
		styles.TealOrange += math.Abs(s.TealOrange)
		styles.RedNight += math.Abs(s.RedNight)
		styles.Grain += math.Abs(s.Grain)
		styles.Crisp += math.Abs(s.Crisp)
	}
	if s := styles; s.Vivid == 0 || s.TealOrange == 0 || s.RedNight == 0 || s.Grain == 0 || s.Crisp == 0 {
		t.Errorf("want every style scored for some film, got %+v", s)
	}

	if b := film(t, stats, "cinestill-800t").Bands; b.Cool < 0.99 {
		t.Errorf("want 800t all cool, got %+v", b)
	}
	if b := film(t, stats, "kodak-gold-200").Bands; b.Warm < 0.99 {
		t.Errorf("want gold all warm, got %+v", b)
	}
	if fam := film(t, stats, "kodak-portra-400").Family; fam != "portrait" {
		t.Errorf("want portra 400 in portrait, got %s", fam)
	}
	if fam := film(t, stats, "ilford-hp5-plus").Family; fam != familyBW {
		t.Errorf("want hp5 in bw, got %s", fam)
	}
	if sim := film(t, stats, "kodak-portra-400").Similar; len(sim) == 0 || sim[0].Slug != "kodak-gold-200" {
		t.Errorf("want gold nearest portra 400, got %+v", sim)
	}
}

func TestPositionsStable(t *testing.T) {
	first := build(t, testSeed())
	again := build(t, testSeed())

	in, vecs := testInput(rand.New(rand.NewSource(1)))
	rand.New(rand.NewSource(2)).Shuffle(len(in.Posts), func(i, j int) { in.Posts[i], in.Posts[j] = in.Posts[j], in.Posts[i] })
	permuted, err := Build(in, vecs, testSeed(), testNow)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range first.Films {
		for name, other := range map[string]*analogdb.Stats{"again": again, "permuted": permuted} {
			if got := film(t, other, f.Slug).Position; got != f.Position {
				t.Errorf("%s %s: want position %+v, got %+v", name, f.Slug, f.Position, got)
			}
		}
		if math.Abs(f.Position.X) > 1 || math.Abs(f.Position.Y) > 1 {
			t.Errorf("%s: position out of range %+v", f.Slug, f.Position)
		}
	}
	if p := film(t, first, positionAnchor).Position; p.X < 0 || p.Y < 0 {
		t.Errorf("want the anchor non-negative, got %+v", p)
	}
}

func TestSubjectsLift(t *testing.T) {
	stats := build(t, testSeed())
	neon := slices.Index(stats.Subjects.Words, "neon")
	night := slices.Index(stats.Subjects.Words, "night")
	for _, row := range stats.Subjects.Rows {
		if row.Lift[night] != nil {
			t.Errorf("%s: want null lift without posts, got %v", row.Slug, *row.Lift[night])
		}
		if row.Count[neon] < MinLiftCount && row.Lift[neon] != nil {
			t.Errorf("%s: want null lift below %d posts, count %d", row.Slug, MinLiftCount, row.Count[neon])
		}
		if row.Slug == "cinestill-800t" && (row.Lift[neon] == nil || *row.Lift[neon] < 2) {
			t.Errorf("want 800t neon lift above 2, got %v", row.Lift[neon])
		}
	}
	if slices.ContainsFunc(stats.Subjects.Rows, func(r analogdb.StatsSubjectRow) bool { return r.Slug == "kodak-ektar-100" }) {
		t.Errorf("want films under %d posts left out of the heatmap", MinHeatmapPosts)
	}
}

func TestAuthorCap(t *testing.T) {
	in := &analogdb.StatsInput{}
	at := time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC).Unix()
	add := func(id int, author, slug string) {
		in.Posts = append(in.Posts, analogdb.StatsInputPost{ID: id, Author: author, Time: at, FilmSlug: slug, FilmMake: "kodak", FilmType: slug})
	}
	for i := range 20 {
		add(i+1, "one", "a")
	}
	for i := range 5 {
		add(100+i, fmt.Sprintf("other-%d", i), "b")
	}
	b := &builder{now: testNow, start: testStart}
	b.index(in, nil)
	tally := b.tally(filmItem, func(*post) int { return 0 }, 1)
	if got := tally.counts["a"][0]; got != AuthorCap {
		t.Errorf("want one author capped at %d, got %v", AuthorCap, got)
	}
	if got := tally.counts["b"][0]; got != 5 {
		t.Errorf("want five authors counted, got %v", got)
	}
	if got := tally.share("a", 0); got != 50 {
		t.Errorf("want a 50%% share after the cap, got %v", got)
	}
}

func TestMovers(t *testing.T) {
	in := &analogdb.StatsInput{}
	id := 0
	add := func(at time.Time, slug string, n int) {
		for range n {
			id++
			in.Posts = append(in.Posts, analogdb.StatsInputPost{
				ID: id, Author: fmt.Sprint(id), Time: at.Unix(), FilmSlug: slug, FilmMake: "kodak", FilmType: slug,
			})
		}
	}
	early := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	late := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	add(early, "rising", 10)
	add(late, "rising", 60)
	add(early, "steady", 50)
	add(late, "steady", 50)
	add(early, "falling", 40)
	add(late, "falling", 10)

	b := &builder{now: testNow, start: testStart}
	b.index(in, nil)
	movers := b.movers(filmItem, 50)
	if len(movers.Up) != 1 || movers.Up[0].Slug != "rising" {
		t.Errorf("want rising up, got %+v", movers.Up)
	}
	if len(movers.Down) != 2 || movers.Down[0].Slug != "falling" || movers.Down[1].Slug != "steady" {
		t.Errorf("want falling then steady down, got %+v", movers.Down)
	}
	for _, m := range movers.Up {
		if m.Change != r3(m.To-m.From) {
			t.Errorf("want change to minus from, got %+v", m)
		}
	}
}

func TestThemes(t *testing.T) {
	stats := build(t, testSeed("t01"))
	for _, theme := range stats.Themes {
		if theme.Key == "t01" {
			t.Error("want the hidden theme left out")
		}
		if len(theme.Posts) == 0 || len(theme.Posts) > ThemePosts {
			t.Errorf("%s: want 1 to %d posts, got %d", theme.Key, ThemePosts, len(theme.Posts))
		}
	}
	for _, line := range stats.Trends.Themes {
		if line.Key == "t01" {
			t.Error("want the hidden theme left out of trends")
		}
	}
	if !slices.IsSortedFunc(stats.Themes, func(x, y analogdb.StatsThemeEntry) int {
		return int(math.Round(10 * (y.Popularity - x.Popularity)))
	}) {
		t.Error("want themes by popularity")
	}

	var mixed analogdb.StatsThemeEntry
	for _, theme := range stats.Themes {
		if theme.Key == "t02" {
			mixed = theme
		}
	}
	if mixed.Color == nil || mixed.BW == nil || mixed.Color.Count+mixed.BW.Count != mixed.Count {
		t.Errorf("want t02 split into color and B&W, got %+v %+v", mixed.Color, mixed.BW)
	}
	if mixed.BW.Popularity >= mixed.Color.Popularity {
		t.Errorf("want B&W below color in the test data, got %v vs %v", mixed.BW.Popularity, mixed.Color.Popularity)
	}
	if stats.Popularity.Border != nil {
		t.Errorf("want no border without posts in its theme, got %+v", stats.Popularity.Border)
	}
}

func TestPopularityAndPosting(t *testing.T) {
	stats := build(t, testSeed())
	if stats.Popularity.BW >= stats.Popularity.Color {
		t.Errorf("want B&W below color, got %v vs %v", stats.Popularity.BW, stats.Popularity.Color)
	}
	if len(stats.Popularity.Saturation) != 10 {
		t.Errorf("want 10 saturation deciles, got %d", len(stats.Popularity.Saturation))
	}
	if len(stats.Popularity.Aperture) != 1 || stats.Popularity.Aperture[0].Value != "f/2" {
		t.Errorf("want one aperture group, got %+v", stats.Popularity.Aperture)
	}
	cells := stats.Posting.Cells
	if len(cells) != 168 {
		t.Fatalf("want 168 posting cells, got %d", len(cells))
	}
	total := 0
	for _, c := range cells {
		total += c.Count
		if c.Weekday < 1 || c.Weekday > 7 {
			t.Errorf("weekday %d out of range", c.Weekday)
		}
	}
	if total != stats.Totals.Photos {
		t.Errorf("want every post in a cell, got %d of %d", total, stats.Totals.Photos)
	}
}

func TestTrendPeriods(t *testing.T) {
	stats := build(t, testSeed())
	tr := stats.Trends
	if tr.Periods[0] != "2022H1" || tr.Periods[len(tr.Periods)-1] != "2026H2" || len(tr.Periods) != 10 {
		t.Errorf("want 2022H1 to 2026H2, got %v", tr.Periods)
	}
	if !slices.Equal(tr.Partial, []string{"2026H2"}) {
		t.Errorf("want the current period partial, got %v", tr.Partial)
	}
	for _, line := range tr.Films {
		if len(line.Shares) != len(tr.Periods) {
			t.Errorf("%s: want a share per period", line.Slug)
		}
	}
	if len(tr.FilmBrands) == 0 || tr.FilmBrands[0].Name != "Kodak" {
		t.Errorf("want Kodak the top film brand, got %+v", tr.FilmBrands)
	}
}

func TestRounding(t *testing.T) {
	body, err := json.Marshal(build(t, testSeed()))
	if err != nil {
		t.Fatal(err)
	}
	var walk func(v any)
	walk = func(v any) {
		switch x := v.(type) {
		case float64:
			if r3(x) != x {
				t.Errorf("unrounded float %v", x)
			}
		case []any:
			for _, e := range x {
				walk(e)
			}
		case map[string]any:
			for _, e := range x {
				walk(e)
			}
		}
	}
	var doc any
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatal(err)
	}
	walk(doc)
}

func TestWindowStart(t *testing.T) {
	posts := []analogdb.StatsInputPost{}
	add := func(month time.Time, n int) {
		for i := range n {
			posts = append(posts, analogdb.StatsInputPost{ID: len(posts) + 1, Time: month.Add(time.Duration(i) * time.Minute).Unix()})
		}
	}
	add(time.Date(2018, 6, 1, 0, 0, 0, 0, time.UTC), 5)
	add(time.Date(2021, 12, 1, 0, 0, 0, 0, time.UTC), 30)
	for m := testStart; m.Before(monthOf(testNow)); m = m.AddDate(0, 1, 0) {
		add(m, MinMonthPosts)
	}
	add(monthOf(testNow), 10)

	if got := windowStart(posts, testNow); !got.Equal(testStart) {
		t.Errorf("want the window from %v, got %v", testStart, got)
	}

	gap := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	thinned := slices.DeleteFunc(slices.Clone(posts), func(p analogdb.StatsInputPost) bool {
		return monthOf(time.Unix(p.Time, 0)).Equal(gap) && p.ID%2 == 0
	})
	if got := windowStart(thinned, testNow); !got.Equal(gap.AddDate(0, 1, 0)) {
		t.Errorf("want the window after a thin month, got %v", got)
	}

	thin := posts[:35]
	if got := windowStart(thin, testNow); !got.Equal(time.Date(2018, 6, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("want the first post's month without a full month, got %v", got)
	}
}

func TestRelativeRanges(t *testing.T) {
	stats := build(t, testSeed())
	tr := stats.Trends
	if stats.Window.From != "2022-01-01" {
		t.Errorf("want the window from the first post's month, got %s", stats.Window.From)
	}
	if tr.ArrivalsAfter != "2023-07" {
		t.Errorf("want arrivals after 18 months, got %s", tr.ArrivalsAfter)
	}
	if tr.Movers.Recent != (analogdb.StatsMonthRange{From: "2025-10", To: "2026-09"}) {
		t.Errorf("want the last 12 complete months, got %+v", tr.Movers.Recent)
	}
	if tr.Movers.Base != (analogdb.StatsMonthRange{From: "2023-10", To: "2025-09"}) {
		t.Errorf("want the 24 months before, got %+v", tr.Movers.Base)
	}

	b := &builder{now: testNow, start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
	if base, _ := b.moverRanges(); !base.from.Equal(b.start) {
		t.Errorf("want the base clipped to the window, got %v", base.from)
	}
}
