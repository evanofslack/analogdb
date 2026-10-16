package stats

import (
	"math"
	"slices"
	"strings"

	"github.com/evanofslack/analogdb"
)

func families() []analogdb.StatsFamily {
	out := make([]analogdb.StatsFamily, len(Families))
	for i, f := range Families {
		out[i] = analogdb.StatsFamily{Key: f.Key, Name: f.Name}
	}
	return out
}

// assignFamilies gives listed films their family, B&W films bw, and every
// other film the family of its nearest listed film
func (b *builder) assignFamilies() {
	listed := map[string]string{}
	for _, f := range Families {
		for _, slug := range f.Films {
			listed[slug] = f.Key
		}
	}
	anchors := []*filmGroup{}
	for _, f := range b.films {
		if key, ok := listed[f.slug]; ok {
			f.family = key
			if f.centered != nil {
				anchors = append(anchors, f)
			}
		}
	}
	for _, f := range b.films {
		if f.family != "" {
			continue
		}
		if f.bw() {
			f.family = familyBW
			continue
		}
		f.family = "consumer"
		best := math.Inf(-1)
		if f.centered == nil {
			continue
		}
		for _, a := range anchors {
			if sim := cosine(f.centered, a.centered); sim > best {
				best, f.family = sim, a.family
			}
		}
	}
}

// similar is the films with the closest centered centroids
func (b *builder) similar(f *filmGroup) []analogdb.StatsSimilar {
	out := []analogdb.StatsSimilar{}
	if f.centered == nil {
		return out
	}
	for _, other := range b.films {
		if other == f || other.centered == nil {
			continue
		}
		out = append(out, analogdb.StatsSimilar{Slug: other.slug, Similarity: cosine(f.centered, other.centered)})
	}
	slices.SortFunc(out, func(x, y analogdb.StatsSimilar) int {
		switch {
		case x.Similarity > y.Similarity:
			return -1
		case x.Similarity < y.Similarity:
			return 1
		}
		return strings.Compare(x.Slug, y.Slug)
	})
	if len(out) > SimilarFilms {
		out = out[:SimilarFilms]
	}
	for i := range out {
		out[i].Similarity = r3(out[i].Similarity)
	}
	return out
}

// positions places each film on a 2D map from the first two principal
// components of the centered centroids, both axes scaled to [-1, 1]
func (b *builder) positions() map[string]analogdb.StatsPosition {
	films := []*filmGroup{}
	for _, f := range b.films {
		if f.centered != nil {
			films = append(films, f)
		}
	}
	slices.SortFunc(films, func(x, y *filmGroup) int { return strings.Compare(x.slug, y.slug) })

	rows := make([][]float32, len(films))
	anchor := -1
	for i, f := range films {
		rows[i] = f.centered
		if f.slug == positionAnchor {
			anchor = i
		}
	}
	xs, ys := pca2(rows)
	orientAxis(xs, anchor)
	orientAxis(ys, anchor)

	out := map[string]analogdb.StatsPosition{}
	for i, f := range films {
		out[f.slug] = analogdb.StatsPosition{X: r3(xs[i]), Y: r3(ys[i])}
	}
	return out
}

// orientAxis flips the axis so the anchor is non-negative, or the largest
// value without an anchor, then scales it to [-1, 1]
func orientAxis(v []float64, anchor int) {
	if len(v) == 0 {
		return
	}
	peak := 0
	for i := range v {
		if math.Abs(v[i]) > math.Abs(v[peak]) {
			peak = i
		}
	}
	ref := peak
	if anchor >= 0 && v[anchor] != 0 {
		ref = anchor
	}
	scale := math.Abs(v[peak])
	if scale == 0 {
		return
	}
	if v[ref] < 0 {
		scale = -scale
	}
	for i := range v {
		v[i] /= scale
	}
}
