package colorspace

import "math"

// Swatch is one cluster of a palette, percent is its share of the whole image
type Swatch struct {
	Hex     string  `json:"hex"`
	Percent float64 `json:"percent"`
}

// Derived holds everything computed from a cluster's hex
type Derived struct {
	Lab
	Chroma float64
	Hue    float64
	Html   string
	Css    string
}

// Derive computes the OKLab and OKLCH values, bucket and CSS name of a hex color
func Derive(hex string) (Derived, error) {
	c, err := ParseHex(hex)
	if err != nil {
		return Derived{}, err
	}
	lab := c.ToLab()
	lch := lab.ToLCh()
	return Derived{
		Lab:    lab,
		Chroma: lch.C,
		Hue:    lch.H,
		Html:   Bucket(lch.L, lch.C, lch.H),
		Css:    NearestCSS(lab),
	}, nil
}

// IsNeutral reports whether a bucket is black, gray or white
func IsNeutral(bucket string) bool {
	return bucket == Black || bucket == Gray || bucket == White
}

// Representative is the index of the cluster with the highest percent times
// chroma among clusters outside the black, gray and white buckets, or -1 when none
func Representative(derived []Derived, percents []float64) int {
	best, bestScore := -1, 0.0
	for i, d := range derived {
		if IsNeutral(d.Html) {
			continue
		}
		if score := percents[i] * d.Chroma; best == -1 || score > bestScore {
			best, bestScore = i, score
		}
	}
	return best
}

// PaletteRepresentative is the hue and chroma of a palette's representative
// color, ok is false when it has none
func PaletteRepresentative(palette []Swatch) (hue, chroma float64, ok bool, err error) {
	derived, percents, err := derivePalette(palette)
	if err != nil {
		return 0, 0, false, err
	}
	i := Representative(derived, percents)
	if i < 0 {
		return 0, 0, false, nil
	}
	return derived[i].Hue, derived[i].Chroma, true, nil
}

func derivePalette(palette []Swatch) ([]Derived, []float64, error) {
	derived, percents := make([]Derived, len(palette)), make([]float64, len(palette))
	for i, s := range palette {
		d, err := Derive(s.Hex)
		if err != nil {
			return nil, nil, err
		}
		derived[i], percents[i] = d, s.Percent
	}
	return derived, percents, nil
}

// PostColor places a post in the rainbow. A chromatic post has the hue and
// chroma of its representative color and a strength of its percent times
// chroma. A neutral post has no hue or chroma, its strength is the share of its
// dominant neutral bucket, Bucket that bucket and Lightness the share weighted
// OKLab lightness of its neutral clusters.
type PostColor struct {
	Hue       float64
	Chroma    float64
	Strength  float64
	Neutral   bool
	Bucket    string
	Lightness float64
}

var neutralMins = map[string]float64{White: NeutralWhiteMin, Gray: NeutralGrayMin, Black: NeutralBlackMin}

var neutralLoops = map[string]int{White: NeutralWhiteLoops, Gray: NeutralGrayLoops, Black: NeutralBlackLoops}

// NewPostColor picks a post's rainbow color from its derived clusters. Grayscale
// posts are always neutral. ok is false when the post is not in the rainbow.
func NewPostColor(derived []Derived, percents []float64, grayscale bool) (PostColor, bool) {
	if !grayscale {
		if i := Representative(derived, percents); i >= 0 {
			return PostColor{Hue: derived[i].Hue, Chroma: derived[i].Chroma, Strength: percents[i] * derived[i].Chroma}, true
		}
	}
	shares := map[string]float64{}
	total, lightness := 0.0, 0.0
	for i, d := range derived {
		if !IsNeutral(d.Html) {
			continue
		}
		shares[d.Html] += percents[i]
		total += percents[i]
		lightness += percents[i] * d.L
	}
	dominant := ""
	for _, name := range []string{White, Gray, Black} {
		if dominant == "" || shares[name] > shares[dominant] {
			dominant = name
		}
	}
	share := shares[dominant]
	if share < neutralMins[dominant] {
		return PostColor{}, false
	}
	return PostColor{Strength: share, Neutral: true, Bucket: dominant, Lightness: math.Min(1, math.Max(0, lightness/total))}, true
}

// PalettePostColor is NewPostColor for a palette of hex colors
func PalettePostColor(palette []Swatch, grayscale bool) (PostColor, bool, error) {
	derived, percents, err := derivePalette(palette)
	if err != nil {
		return PostColor{}, false, err
	}
	c, ok := NewPostColor(derived, percents, grayscale)
	return c, ok, nil
}

// BucketShares sums each bucket's share of a palette
func BucketShares(palette []Swatch) (map[string]float64, error) {
	shares := map[string]float64{}
	for _, s := range palette {
		d, err := Derive(s.Hex)
		if err != nil {
			return nil, err
		}
		shares[d.Html] += s.Percent
	}
	return shares, nil
}

// HexShare sums the share of clusters within radius of the query hex
func HexShare(palette []Swatch, query string, radius float64) (float64, error) {
	q, err := ParseHex(query)
	if err != nil {
		return 0, err
	}
	qlab := q.ToLab()
	share := 0.0
	for _, s := range palette {
		c, err := ParseHex(s.Hex)
		if err != nil {
			return 0, err
		}
		if DeltaE(c.ToLab(), qlab) < radius {
			share += s.Percent
		}
	}
	return share, nil
}

// RainbowTotalLoops is the number of loops in the rainbow
func RainbowTotalLoops() int {
	total := 0
	for _, n := range RainbowTierLoops {
		total += n
	}
	return total
}

// RainbowTier is the index of the strength tier a chromatic post falls in. It
// compares in float32, like the REAL column strength is stored in.
func RainbowTier(strength float64) int {
	for i, min := range RainbowTierStrengths {
		if float32(strength) >= float32(min) {
			return i
		}
	}
	return len(RainbowTierStrengths) - 1
}

// RainbowNeutralLoops is the number of loops a neutral bucket is spread over
func RainbowNeutralLoops(bucket string) int {
	return max(1, min(neutralLoops[bucket], RainbowTotalLoops()))
}

func rainbowHash(id int, loops int) int {
	return int((uint64(id) * RainbowHash) % uint64(loops))
}

// RainbowKey is the rainbow_key of a post. Each loop holds the hues from red
// to purple, then neutral posts from white to black. Hue and strength are
// rounded to float32, like the REAL columns they are stored in.
func RainbowKey(id int, c PostColor) float64 {
	width := RainbowHueWidth + RainbowNeutralWidth
	if c.Neutral {
		cycle := rainbowHash(id, RainbowNeutralLoops(c.Bucket))
		l := math.Min(1, math.Max(0, c.Lightness))
		return float64(cycle)*width + RainbowHueWidth + (1-l)*RainbowNeutralWidth
	}
	tier := RainbowTier(c.Strength)
	offset := 0
	for _, n := range RainbowTierLoops[:tier] {
		offset += n
	}
	cycle := offset + rainbowHash(id, RainbowTierLoops[tier])
	h := float64(float32(c.Hue)) - RainbowHueOffset + 360
	if h >= 360 {
		h -= 360
	}
	return float64(cycle)*width + h
}
