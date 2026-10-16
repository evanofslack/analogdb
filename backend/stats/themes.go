package stats

import (
	"cmp"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/evanofslack/analogdb"
)

type themeGroup struct {
	analogdb.StatsTheme
	center []float32
	posts  []*post
}

// assignThemes puts each post with a vector in the theme with the nearest
// center. Hidden themes take posts too, they are only left out of the output.
func (b *builder) assignThemes() {
	b.themes = make([]themeGroup, len(b.seed.Themes))
	for i, t := range b.seed.Themes {
		b.themes[i] = themeGroup{StatsTheme: t, center: unit(t.Center)}
	}
	for _, p := range b.posts {
		if p.vec == nil {
			continue
		}
		best, bestSim := -1, math.Inf(-1)
		for i, t := range b.themes {
			if t.center == nil {
				continue
			}
			if sim := dot(p.vec, t.center); sim > bestSim {
				best, bestSim = i, sim
			}
		}
		if best >= 0 {
			p.theme = best
			b.themes[best].posts = append(b.themes[best].posts, p)
		}
	}
}

func popularityGroup(posts []*post) analogdb.StatsGroup {
	pops := make([]float64, len(posts))
	for i, p := range posts {
		pops[i] = p.Popularity
	}
	return analogdb.StatsGroup{Count: len(posts), Popularity: pop(mean(pops))}
}

// themeEntries lists the visible themes by popularity
func (b *builder) themeEntries() []analogdb.StatsThemeEntry {
	out := []analogdb.StatsThemeEntry{}
	for _, t := range b.themes {
		if t.Hidden || len(t.posts) == 0 {
			continue
		}
		color, bw := []*post{}, []*post{}
		nsfw := 0
		for _, p := range t.posts {
			if p.Greyscale {
				bw = append(bw, p)
			} else {
				color = append(color, p)
			}
			if p.Nsfw {
				nsfw++
			}
		}
		all := popularityGroup(t.posts)
		entry := analogdb.StatsThemeEntry{
			Key:        t.Key,
			Name:       t.Name,
			Count:      all.Count,
			Popularity: all.Popularity,
			NsfwShare:  r3(float64(nsfw) / float64(len(t.posts))),
			Posts:      nearestPosts(t.posts, t.center, ThemePosts, false),
		}
		if len(color) > 0 {
			g := popularityGroup(color)
			entry.Color = &g
		}
		if len(bw) >= MinBWPosts {
			g := popularityGroup(bw)
			entry.BW = &g
		}
		out = append(out, entry)
	}
	slices.SortStableFunc(out, func(x, y analogdb.StatsThemeEntry) int {
		if c := cmp.Compare(y.Popularity, x.Popularity); c != 0 {
			return c
		}
		return strings.Compare(x.Key, y.Key)
	})
	return out
}

func (b *builder) popularity() analogdb.StatsPopularity {
	color, bw := []*post{}, []*post{}
	saturated := []*post{}
	apertures := map[string][]*post{}
	focals := map[int][]*post{}
	for _, p := range b.posts {
		if p.Greyscale {
			bw = append(bw, p)
		} else {
			color = append(color, p)
			if p.hasColor {
				saturated = append(saturated, p)
			}
		}
		if p.Aperture != "" {
			apertures[p.Aperture] = append(apertures[p.Aperture], p)
		}
		if p.FocalLength > 0 {
			focals[p.FocalLength] = append(focals[p.FocalLength], p)
		}
	}

	out := analogdb.StatsPopularity{
		Color:      popularityGroup(color).Popularity,
		BW:         popularityGroup(bw).Popularity,
		Saturation: []float64{},
		Aperture:   []analogdb.StatsValueGroup{},
		Focal:      []analogdb.StatsFocalGroup{},
	}

	slices.SortStableFunc(saturated, func(x, y *post) int { return cmp.Compare(x.chroma, y.chroma) })
	if len(saturated) >= 10 {
		for d := range 10 {
			lo, hi := d*len(saturated)/10, (d+1)*len(saturated)/10
			out.Saturation = append(out.Saturation, popularityGroup(saturated[lo:hi]).Popularity)
		}
	}

	for value, posts := range apertures {
		if len(posts) >= MinGroupPosts {
			g := popularityGroup(posts)
			out.Aperture = append(out.Aperture, analogdb.StatsValueGroup{Value: value, Count: g.Count, Popularity: g.Popularity})
		}
	}
	slices.SortFunc(out.Aperture, func(x, y analogdb.StatsValueGroup) int {
		if c := cmp.Compare(apertureValue(x.Value), apertureValue(y.Value)); c != 0 {
			return c
		}
		return strings.Compare(x.Value, y.Value)
	})

	for value, posts := range focals {
		if len(posts) >= MinGroupPosts {
			g := popularityGroup(posts)
			out.Focal = append(out.Focal, analogdb.StatsFocalGroup{Value: value, Count: g.Count, Popularity: g.Popularity})
		}
	}
	slices.SortFunc(out.Focal, func(x, y analogdb.StatsFocalGroup) int { return x.Value - y.Value })

	for _, t := range b.themes {
		if t.Role != nil && *t.Role == analogdb.StatsRoleBorder && len(t.posts) > 0 {
			out.Border = &analogdb.StatsBorder{Theme: t.Key, Popularity: popularityGroup(t.posts).Popularity}
		}
	}
	return out
}

// apertureValue reads the f-number of an aperture like f/1.4, unknown ones sort last
func apertureValue(s string) float64 {
	v, err := strconv.ParseFloat(strings.TrimPrefix(strings.ToLower(s), "f/"), 64)
	if err != nil {
		return math.Inf(1)
	}
	return v
}
