// Package stats builds the stats page document from posts, colors, subject
// keywords, image vectors and the curated seed. It does no IO.
package stats

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/evanofslack/analogdb"
)

// post is one input post with its derived values
type post struct {
	analogdb.StatsInputPost
	colors []analogdb.StatsInputColor
	words  map[string]bool
	// unit vector, nil when the post has none
	vec []float32
	// percent weighted mean of its colors, hasColor false without colors
	hasColor  bool
	lightness float64
	chroma    float64
	theme     int
}

type builder struct {
	now time.Time
	// first day of the window
	start  time.Time
	seed   *analogdb.StatsSeed
	posts  []*post
	films  []*filmGroup
	themes []themeGroup
	// mean unit vector over every post with a vector
	globalMean []float32
	// z scored style per post, by prompt key
	styles map[string][]float64
}

// Build computes the stats document. vecs maps post ids to image vectors, it
// may hold posts outside the input, they are ignored.
func Build(in *analogdb.StatsInput, vecs map[int][]float32, seed *analogdb.StatsSeed, now time.Time) (*analogdb.Stats, error) {
	if in == nil || len(in.Posts) == 0 {
		return nil, errors.New("no posts in the window")
	}
	if seed == nil || len(seed.Themes) == 0 {
		return nil, errors.New("no stats seed")
	}
	if !slices.ContainsFunc(seed.Prompts, func(p analogdb.StatsPrompt) bool { return p.Key == analogdb.StatsPromptBase }) {
		return nil, fmt.Errorf("seed has no %q prompt", analogdb.StatsPromptBase)
	}

	b := &builder{now: now.UTC(), seed: seed}
	b.start = windowStart(in.Posts, b.now)
	b.index(in, vecs)
	b.computeStyles()
	b.assignThemes()
	b.groupFilms()

	stats := &analogdb.Stats{
		Version:     SnapshotVersion,
		Generated:   b.now.Truncate(time.Second),
		SeedVersion: seed.Version,
		Window:      analogdb.StatsWindow{From: b.start.Format(time.DateOnly), To: b.now.Format(time.DateOnly)},
		Totals:      b.totals(),
		Films:       b.filmLooks(),
		Families:    families(),
		Subjects:    b.subjects(),
		Trends:      b.trends(),
		Themes:      b.themeEntries(),
		Popularity:  b.popularity(),
		Posting:     b.posting(),
		Method: analogdb.StatsMethod{
			MinFilmPosts:   MinFilmPosts,
			MinCameraPosts: MinCameraPosts,
			AuthorCap:      AuthorCap,
			Model:          seed.Model,
		},
	}
	return stats, nil
}

func (b *builder) index(in *analogdb.StatsInput, vecs map[int][]float32) {
	start := b.start.Unix()
	byID := make(map[int]*post, len(in.Posts))
	for _, p := range in.Posts {
		if p.Time < start {
			continue
		}
		pp := &post{StatsInputPost: p, words: map[string]bool{}, theme: -1}
		if v, ok := vecs[p.ID]; ok && len(v) == analogdb.StatsVectorDims {
			pp.vec = unit(v)
		}
		byID[p.ID] = pp
		b.posts = append(b.posts, pp)
	}
	slices.SortFunc(b.posts, func(x, y *post) int { return x.ID - y.ID })

	for _, c := range in.Colors {
		if p, ok := byID[c.PostID]; ok {
			p.colors = append(p.colors, c)
		}
	}
	for _, k := range in.Keywords {
		if p, ok := byID[k.PostID]; ok {
			p.words[k.Word] = true
		}
	}

	withVec := [][]float32{}
	for _, p := range b.posts {
		p.lightness, p.chroma, p.hasColor = postColor(p.colors)
		if p.vec != nil {
			withVec = append(withVec, p.vec)
		}
	}
	b.globalMean = meanVector(withVec)
}

func monthOf(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// windowStart is the first month after which every complete month has at
// least MinMonthPosts posts, so thin early months stay out as the archive
// grows backwards. Without such a month it is the month of the first post.
func windowStart(posts []analogdb.StatsInputPost, now time.Time) time.Time {
	current := monthOf(now)
	if len(posts) == 0 {
		return current
	}
	counts := map[time.Time]int{}
	first := current
	for _, p := range posts {
		m := monthOf(time.Unix(p.Time, 0))
		counts[m]++
		if m.Before(first) {
			first = m
		}
	}
	start := first
	for m := first; m.Before(current); m = m.AddDate(0, 1, 0) {
		if counts[m] < MinMonthPosts {
			start = m.AddDate(0, 1, 0)
		}
	}
	if !start.Before(current) {
		return first
	}
	return start
}

// postColor is the percent weighted mean lightness and chroma of a post's colors
func postColor(colors []analogdb.StatsInputColor) (lightness, chroma float64, ok bool) {
	var weight float64
	for _, c := range colors {
		weight += c.Percent
		lightness += c.Percent * c.L
		chroma += c.Percent * c.Chroma
	}
	if weight == 0 {
		return 0, 0, false
	}
	return lightness / weight, chroma / weight, true
}

func (b *builder) totals() analogdb.StatsTotals {
	authors, films, cameras := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, p := range b.posts {
		if p.Author != "" {
			authors[p.Author] = true
		}
		if p.FilmSlug != "" {
			films[p.FilmSlug] = true
		}
		if p.CameraSlug != "" {
			cameras[p.CameraSlug] = true
		}
	}
	return analogdb.StatsTotals{
		Photos:        len(b.posts),
		Photographers: len(authors),
		Films:         len(films),
		Cameras:       len(cameras),
	}
}

func statsPost(p *post) analogdb.StatsPost {
	return analogdb.StatsPost{ID: p.ID, Title: p.Title, Score: p.Score, Nsfw: p.Nsfw, Images: p.Images}
}

// titleMake capitalizes each word of a make, the catalog stores them lowercase
func titleMake(make string) string {
	words := strings.Fields(make)
	for i, w := range words {
		r := []rune(w)
		r[0] = unicode.ToUpper(r[0])
		words[i] = string(r)
	}
	return strings.Join(words, " ")
}

// nearestPosts returns up to n posts by cosine to target, at most one per
// author, optionally skipping NSFW posts
func nearestPosts(posts []*post, target []float32, n int, skipNsfw bool) []analogdb.StatsPost {
	type scored struct {
		p   *post
		sim float64
	}
	candidates := []scored{}
	for _, p := range posts {
		if p.vec == nil || (skipNsfw && p.Nsfw) {
			continue
		}
		candidates = append(candidates, scored{p, dot(p.vec, target)})
	}
	slices.SortStableFunc(candidates, func(x, y scored) int {
		if x.sim != y.sim {
			if x.sim > y.sim {
				return -1
			}
			return 1
		}
		return x.p.ID - y.p.ID
	})
	out := []analogdb.StatsPost{}
	authors := map[string]bool{}
	for _, c := range candidates {
		if len(out) == n {
			break
		}
		if c.p.Author != "" && authors[c.p.Author] {
			continue
		}
		authors[c.p.Author] = true
		out = append(out, statsPost(c.p))
	}
	return out
}
