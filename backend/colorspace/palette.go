package colorspace

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

// Representative is the index of the cluster with the highest percent times
// chroma among clusters with chroma at least NeutralChroma, or -1 when none
func Representative(chromas, percents []float64) int {
	best, bestScore := -1, 0.0
	for i, c := range chromas {
		if c < NeutralChroma {
			continue
		}
		if score := percents[i] * c; best == -1 || score > bestScore {
			best, bestScore = i, score
		}
	}
	return best
}

// PaletteRepresentative is the hue and chroma of a palette's representative
// color, ok is false when it has none. Grayscale posts have none either.
func PaletteRepresentative(palette []Swatch) (hue, chroma float64, ok bool, err error) {
	chromas, percents, hues := make([]float64, len(palette)), make([]float64, len(palette)), make([]float64, len(palette))
	for i, s := range palette {
		d, err := Derive(s.Hex)
		if err != nil {
			return 0, 0, false, err
		}
		chromas[i], percents[i], hues[i] = d.Chroma, s.Percent, d.Hue
	}
	i := Representative(chromas, percents)
	if i < 0 {
		return 0, 0, false, nil
	}
	return hues[i], chromas[i], true, nil
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

// RainbowKey matches the rainbow_key column on pictures for a post id and its
// representative hue. The hue is rounded to float32 first, as the column is REAL.
func RainbowKey(id int, hue float64) float64 {
	h := float64(float32(hue)) - RainbowHueOffset + 360
	if h >= 360 {
		h -= 360
	}
	cycle := (int64(id) * RainbowHash) % RainbowCycles
	return float64(cycle*360) + h
}
