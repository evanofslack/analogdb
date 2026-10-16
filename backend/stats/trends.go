package stats

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/evanofslack/analogdb"
)

// item names what a post is counted under, ok false when it has none
type item func(p *post) (key, name string, ok bool)

func filmItem(p *post) (string, string, bool) {
	return p.FilmSlug, titleMake(p.FilmMake) + " " + p.FilmType, p.FilmSlug != ""
}

func cameraItem(p *post) (string, string, bool) {
	return p.CameraSlug, titleMake(p.CameraMake) + " " + p.CameraModel, p.CameraSlug != ""
}

func filmBrandItem(p *post) (string, string, bool) {
	return p.FilmMake, titleMake(p.FilmMake), p.FilmSlug != ""
}

func cameraBrandItem(p *post) (string, string, bool) {
	return p.CameraMake, titleMake(p.CameraMake), p.CameraSlug != ""
}

// tally holds counts per item and period where one author counts at most
// AuthorCap posts per item per period
type tally struct {
	names  map[string]string
	counts map[string][]float64
	raw    map[string]int
	first  map[string]int64
	totals []float64
}

func (t *tally) total(key string) float64 {
	var sum float64
	for _, c := range t.counts[key] {
		sum += c
	}
	return sum
}

func (t *tally) share(key string, period int) float64 {
	if t.totals[period] == 0 {
		return 0
	}
	return 100 * t.counts[key][period] / t.totals[period]
}

// ranked returns the items with at least min posts by capped total
func (t *tally) ranked(min int) []string {
	keys := []string{}
	for key, n := range t.raw {
		if n >= min {
			keys = append(keys, key)
		}
	}
	slices.SortFunc(keys, func(x, y string) int {
		if c := cmp.Compare(t.total(y), t.total(x)); c != 0 {
			return c
		}
		return strings.Compare(x, y)
	})
	return keys
}

func (b *builder) tally(of item, periodOf func(p *post) int, periods int) *tally {
	t := &tally{
		names:  map[string]string{},
		counts: map[string][]float64{},
		raw:    map[string]int{},
		first:  map[string]int64{},
		totals: make([]float64, periods),
	}
	type capKey struct {
		item   string
		period int
		author string
	}
	seen := map[capKey]int{}
	for _, p := range b.posts {
		key, name, ok := of(p)
		if !ok {
			continue
		}
		period := periodOf(p)
		if period < 0 || period >= periods {
			continue
		}
		if _, ok := t.names[key]; !ok {
			t.names[key] = name
			t.counts[key] = make([]float64, periods)
		}
		t.raw[key]++
		if first, ok := t.first[key]; !ok || p.Time < first {
			t.first[key] = p.Time
		}
		if p.Author != "" {
			k := capKey{key, period, p.Author}
			if seen[k] >= AuthorCap {
				continue
			}
			seen[k]++
		}
		t.counts[key][period]++
		t.totals[period]++
	}
	return t
}

// halfYear is the index of a time's half year counted from the window start
func halfYear(t time.Time) int {
	t = t.UTC()
	return (t.Year()-Start.Year())*2 + (int(t.Month())-1)/6
}

func periodName(i int) string {
	return fmt.Sprintf("%dH%d", Start.Year()+i/2, i%2+1)
}

func (b *builder) trends() analogdb.StatsTrends {
	periods := halfYear(b.now) + 1
	byHalf := func(p *post) int { return halfYear(time.Unix(p.Time, 0)) }

	names := make([]string, periods)
	for i := range names {
		names[i] = periodName(i)
	}
	films := b.tally(filmItem, byHalf, periods)
	cameras := b.tally(cameraItem, byHalf, periods)

	return analogdb.StatsTrends{
		Periods:      names,
		Partial:      []string{names[periods-1]},
		Films:        shareLines(films, films.ranked(MinTrendPosts), TrendTop, true),
		Cameras:      shareLines(cameras, cameras.ranked(MinTrendPosts), TrendTop, true),
		FilmBrands:   brandLines(b.tally(filmBrandItem, byHalf, periods)),
		CameraBrands: brandLines(b.tally(cameraBrandItem, byHalf, periods)),
		Movers: analogdb.StatsMoverGroups{
			Films:   b.movers(filmItem, MinFilmPosts),
			Cameras: b.movers(cameraItem, MinCameraPosts),
		},
		Arrivals: b.arrivals(films, cameras),
		Themes:   b.themeTrends(periods),
	}
}

func shareLines(t *tally, keys []string, top int, slug bool) []analogdb.StatsTrendItem {
	if len(keys) > top {
		keys = keys[:top]
	}
	out := make([]analogdb.StatsTrendItem, 0, len(keys))
	for _, key := range keys {
		line := analogdb.StatsTrendItem{Name: t.names[key], Shares: make([]float64, len(t.totals))}
		if slug {
			line.Slug = key
		}
		for i := range t.totals {
			line.Shares[i] = r3(t.share(key, i))
		}
		out = append(out, line)
	}
	return out
}

func brandLines(t *tally) []analogdb.StatsTrendItem {
	return shareLines(t, t.ranked(MinTrendPosts), BrandTop, false)
}

// movers compares each item's share across the base years with its share in
// the last twelve complete months
func (b *builder) movers(of item, min int) analogdb.StatsMovers {
	recentEnd := time.Date(b.now.Year(), b.now.Month(), 1, 0, 0, 0, 0, time.UTC)
	recentStart := recentEnd.AddDate(-1, 0, 0)
	window := func(p *post) int {
		switch t := time.Unix(p.Time, 0).UTC(); {
		case t.Before(MoversFromEnd):
			return 0
		case !t.Before(recentStart) && t.Before(recentEnd):
			return 1
		}
		return -1
	}
	t := b.tally(of, window, 2)

	movers := []analogdb.StatsMover{}
	for _, key := range t.ranked(min) {
		from, to := t.share(key, 0), t.share(key, 1)
		if max(from, to) < MoverMinShare {
			continue
		}
		movers = append(movers, analogdb.StatsMover{Slug: key, Name: t.names[key], From: r3(from), To: r3(to), Change: r3(to - from)})
	}
	slices.SortStableFunc(movers, func(x, y analogdb.StatsMover) int { return cmp.Compare(y.Change, x.Change) })

	out := analogdb.StatsMovers{Up: []analogdb.StatsMover{}, Down: []analogdb.StatsMover{}}
	for _, m := range movers {
		if m.Change > 0 && len(out.Up) < MoverCount {
			out.Up = append(out.Up, m)
		}
	}
	for i := len(movers) - 1; i >= 0; i-- {
		if movers[i].Change < 0 && len(out.Down) < MoverCount {
			out.Down = append(out.Down, movers[i])
		}
	}
	return out
}

// arrivals is the films and cameras first seen late in the window, by count
func (b *builder) arrivals(films, cameras *tally) []analogdb.StatsArrival {
	after := ArrivalsAfter.Unix()
	out := []analogdb.StatsArrival{}
	for _, group := range []struct {
		kind string
		t    *tally
	}{{"film", films}, {"camera", cameras}} {
		for _, key := range group.t.ranked(MinTrendPosts) {
			first := group.t.first[key]
			if first < after {
				continue
			}
			out = append(out, analogdb.StatsArrival{
				Kind:  group.kind,
				Slug:  key,
				Name:  group.t.names[key],
				First: time.Unix(first, 0).UTC().Format("2006-01"),
				Count: int(group.t.total(key)),
			})
		}
	}
	slices.SortStableFunc(out, func(x, y analogdb.StatsArrival) int {
		if c := cmp.Compare(y.Count, x.Count); c != 0 {
			return c
		}
		return strings.Compare(x.Slug, y.Slug)
	})
	if len(out) > ArrivalsTop {
		out = out[:ArrivalsTop]
	}
	return out
}

// themeTrends is each visible theme's share of the posts with a vector per half year
func (b *builder) themeTrends(periods int) []analogdb.StatsTrendItem {
	totals := make([]float64, periods)
	counts := make([][]float64, len(b.themes))
	for i := range counts {
		counts[i] = make([]float64, periods)
	}
	for _, p := range b.posts {
		if p.theme < 0 {
			continue
		}
		period := halfYear(time.Unix(p.Time, 0))
		if period < 0 || period >= periods {
			continue
		}
		totals[period]++
		counts[p.theme][period]++
	}
	out := []analogdb.StatsTrendItem{}
	for i, t := range b.themes {
		if t.Hidden || len(t.posts) == 0 {
			continue
		}
		line := analogdb.StatsTrendItem{Key: t.Key, Name: t.Name, Shares: make([]float64, periods)}
		for j := range totals {
			if totals[j] > 0 {
				line.Shares[j] = r3(100 * counts[i][j] / totals[j])
			}
		}
		out = append(out, line)
	}
	return out
}
