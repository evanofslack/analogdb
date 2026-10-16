package stats

import (
	"math"
	"slices"
	"strings"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/colorspace"
)

type filmGroup struct {
	slug      string
	make      string
	typ       string
	colorType string
	posts     []*post
	// mean unit vector of its posts, and that minus the global mean
	centroid []float32
	centered []float32
	family   string
}

func (f *filmGroup) bw() bool { return f.colorType == "bw" }

// groupFilms collects the films with enough posts, by post count then slug
func (b *builder) groupFilms() {
	bySlug := map[string]*filmGroup{}
	for _, p := range b.posts {
		if p.FilmSlug == "" {
			continue
		}
		f, ok := bySlug[p.FilmSlug]
		if !ok {
			f = &filmGroup{slug: p.FilmSlug, make: p.FilmMake, typ: p.FilmType, colorType: p.FilmColorType}
			bySlug[p.FilmSlug] = f
		}
		f.posts = append(f.posts, p)
	}
	for _, f := range bySlug {
		if len(f.posts) < MinFilmPosts {
			continue
		}
		vecs := [][]float32{}
		for _, p := range f.posts {
			if p.vec != nil {
				vecs = append(vecs, p.vec)
			}
		}
		if f.centroid = meanVector(vecs); f.centroid != nil {
			f.centered = sub(f.centroid, b.globalMean)
		}
		b.films = append(b.films, f)
	}
	slices.SortFunc(b.films, func(x, y *filmGroup) int {
		if len(x.posts) != len(y.posts) {
			return len(y.posts) - len(x.posts)
		}
		return strings.Compare(x.slug, y.slug)
	})
	b.assignFamilies()
}

// computeStyles scores each post against every style prompt relative to the
// base prompt and z scores each style over all posts with a vector
func (b *builder) computeStyles() {
	var base []float32
	for _, p := range b.seed.Prompts {
		if p.Key == analogdb.StatsPromptBase {
			base = unit(p.Vector)
		}
	}
	b.styles = map[string][]float64{}
	for _, prompt := range b.seed.Prompts {
		if prompt.Key == analogdb.StatsPromptBase {
			continue
		}
		vec := unit(prompt.Vector)
		raw := make([]float64, len(b.posts))
		values := []float64{}
		for i, p := range b.posts {
			if p.vec == nil {
				raw[i] = math.NaN()
				continue
			}
			raw[i] = dot(p.vec, vec) - dot(p.vec, base)
			values = append(values, raw[i])
		}
		mu := mean(values)
		var variance float64
		for _, v := range values {
			variance += (v - mu) * (v - mu)
		}
		sd := 0.0
		if len(values) > 0 {
			sd = math.Sqrt(variance / float64(len(values)))
		}
		for i, v := range raw {
			if math.IsNaN(v) || sd == 0 {
				continue
			}
			raw[i] = (v - mu) / sd
		}
		b.styles[prompt.Key] = raw
	}
}

func (b *builder) filmLooks() []analogdb.StatsFilm {
	position := b.positions()
	postIndex := make(map[*post]int, len(b.posts))
	for i, p := range b.posts {
		postIndex[p] = i
	}

	films := []analogdb.StatsFilm{}
	for _, f := range b.films {
		pops := make([]float64, 0, len(f.posts))
		for _, p := range f.posts {
			pops = append(pops, p.Popularity)
		}
		film := analogdb.StatsFilm{
			Slug:       f.slug,
			Make:       f.make,
			Type:       f.typ,
			ColorType:  f.colorType,
			Family:     f.family,
			PostCount:  len(f.posts),
			Popularity: pop(mean(pops)),
			Hue:        []float64{},
			Palette:    []string{},
			Styles:     map[string]float64{},
			Similar:    b.similar(f),
			Position:   position[f.slug],
			Typical:    []analogdb.StatsPost{},
		}

		lightness, chroma := []float64{}, []float64{}
		for _, p := range f.posts {
			if !p.hasColor || (!f.bw() && p.Greyscale) {
				continue
			}
			lightness = append(lightness, p.lightness)
			chroma = append(chroma, p.chroma)
		}
		film.Lightness = r3(mean(lightness))
		film.Chroma = r3(mean(chroma))

		if !f.bw() {
			film.Hue, film.Bands, film.Palette = filmHues(f.posts)
		}

		for key, values := range b.styles {
			zs := []float64{}
			for _, p := range f.posts {
				if z := values[postIndex[p]]; !math.IsNaN(z) {
					zs = append(zs, z)
				}
			}
			if len(zs) > 0 {
				film.Styles[key] = r3(mean(zs))
			}
		}

		if f.centroid != nil {
			film.Typical = nearestPosts(f.posts, unit(f.centroid), TypicalPosts, true)
		}
		films = append(films, film)
	}
	return films
}

// chromatic is one color row that counts toward hue, weighted by its percent
type chromatic struct {
	lab    colorspace.Lab
	hue    float64
	weight float64
}

func chromaticRows(posts []*post) []chromatic {
	rows := []chromatic{}
	for _, p := range posts {
		if p.Greyscale {
			continue
		}
		for _, c := range p.colors {
			if c.Chroma > ChromaThreshold && c.Percent > 0 {
				rows = append(rows, chromatic{lab: colorspace.Lab{L: c.L, A: c.A, B: c.B}, hue: c.Hue, weight: c.Percent})
			}
		}
	}
	return rows
}

func hueBin(hue float64) int {
	bin := int(math.Floor(math.Mod(hue, 360) / (360.0 / HueBins)))
	if bin < 0 {
		bin += HueBins
	}
	return bin % HueBins
}

// hueBand names the band of an OKLCH hue: warm 20 to 110, green to 180, cool
// to 290, magenta the rest
func hueBand(hue float64) string {
	h := math.Mod(hue, 360)
	if h < 0 {
		h += 360
	}
	switch {
	case h >= 20 && h < 110:
		return "warm"
	case h >= 110 && h < 180:
		return "green"
	case h >= 180 && h < 290:
		return "cool"
	default:
		return "magenta"
	}
}

// filmHues is the chromatic mass share per hue bin and band, and a palette
func filmHues(posts []*post) ([]float64, *analogdb.StatsBands, []string) {
	rows := chromaticRows(posts)
	var total float64
	bins := make([]float64, HueBins)
	bands := map[string]float64{}
	for _, r := range rows {
		total += r.weight
		bins[hueBin(r.hue)] += r.weight
		bands[hueBand(r.hue)] += r.weight
	}
	if total == 0 {
		return []float64{}, nil, []string{}
	}
	for i := range bins {
		bins[i] = r3(bins[i] / total)
	}
	return bins, &analogdb.StatsBands{
		Warm:    r3(bands["warm"] / total),
		Green:   r3(bands["green"] / total),
		Cool:    r3(bands["cool"] / total),
		Magenta: r3(bands["magenta"] / total),
	}, palette(rows)
}

// palette runs a weighted k-means in OKLab seeded from the largest hue bins
// and returns the centers as hex, ordered by hue
func palette(rows []chromatic) []string {
	byBin := make([][]chromatic, HueBins)
	weights := make([]float64, HueBins)
	for _, r := range rows {
		bin := hueBin(r.hue)
		byBin[bin] = append(byBin[bin], r)
		weights[bin] += r.weight
	}
	order := make([]int, HueBins)
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(x, y int) int {
		switch {
		case weights[x] > weights[y]:
			return -1
		case weights[x] < weights[y]:
			return 1
		}
		return x - y
	})

	centers := []colorspace.Lab{}
	for _, bin := range order {
		if len(centers) == PaletteSize || weights[bin] == 0 {
			break
		}
		centers = append(centers, weightedMean(byBin[bin]))
	}
	if len(centers) == 0 {
		return []string{}
	}

	for range PaletteIterations {
		groups := make([][]chromatic, len(centers))
		for _, r := range rows {
			best, bestDist := 0, math.Inf(1)
			for i, c := range centers {
				if d := colorspace.DeltaE(r.lab, c); d < bestDist {
					best, bestDist = i, d
				}
			}
			groups[best] = append(groups[best], r)
		}
		for i, g := range groups {
			if len(g) > 0 {
				centers[i] = weightedMean(g)
			}
		}
	}

	slices.SortFunc(centers, func(x, y colorspace.Lab) int {
		hx, hy := x.ToLCh().H, y.ToLCh().H
		switch {
		case hx < hy:
			return -1
		case hx > hy:
			return 1
		}
		return 0
	})
	hexes := make([]string, len(centers))
	for i, c := range centers {
		hexes[i] = "#" + c.ToRGB().Hex()
	}
	return hexes
}

func weightedMean(rows []chromatic) colorspace.Lab {
	var l, a, bb, w float64
	for _, r := range rows {
		l += r.weight * r.lab.L
		a += r.weight * r.lab.A
		bb += r.weight * r.lab.B
		w += r.weight
	}
	return colorspace.Lab{L: l / w, A: a / w, B: bb / w}
}
