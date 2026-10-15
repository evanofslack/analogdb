package colorspace

import (
	"image/color"
	"math"
	"sort"

	"golang.org/x/image/colornames"
)

// Bucket names, the html colors a post can be filtered by
const (
	Red    = "red"
	Orange = "orange"
	Tan    = "tan"
	Yellow = "yellow"
	Green  = "green"
	Olive  = "olive"
	Teal   = "teal"
	Navy   = "navy"
	Purple = "purple"
	Gray   = "gray"
	Black  = "black"
	White  = "white"
	Brown  = "brown"
)

// Buckets lists every bucket name
var Buckets = []string{Red, Orange, Tan, Yellow, Green, Olive, Teal, Navy, Purple, Gray, Black, White, Brown}

// IsBucket reports whether name is a bucket name
func IsBucket(name string) bool {
	_, ok := BucketThresholds[name]
	return ok
}

func inHue(h, from, to float64) bool {
	return h >= from && h < to
}

// Bucket names an OKLCH color
func Bucket(l, c, h float64) string {
	switch {
	case c < NeutralChroma:
		if l < BlackMaxL {
			return Black
		}
		if l >= WhiteMinL {
			return White
		}
		return Gray
	case inHue(h, BrownHueMin, BrownHueMax) && l < BrownMaxL:
		return Brown
	case inHue(h, BrownHueMin, BrownHueMax) && c < TanMaxChroma:
		return Tan
	case inHue(h, OliveHueMin, OliveHueMax) && (l < OliveMaxL || c < OliveLowC && l < OliveLowCMax):
		return Olive
	}
	name := hueRanges[0].Name
	for _, r := range hueRanges {
		if h >= r.From {
			name = r.Name
		}
	}
	return name
}

type cssName struct {
	name string
	lab  Lab
}

var cssNames = func() []cssName {
	names := make([]string, 0, len(colornames.Map))
	for name := range colornames.Map {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]cssName, 0, len(names))
	for _, name := range names {
		c := colornames.Map[name]
		out = append(out, cssName{name: name, lab: rgba(c).ToLab()})
	}
	return out
}()

func rgba(c color.RGBA) RGB {
	return RGB{R: c.R, G: c.G, B: c.B}
}

// NearestCSS is the CSS color name closest in OKLab, the first alphabetically on a tie
func NearestCSS(lab Lab) string {
	best, bestDist := "", math.Inf(1)
	for _, c := range cssNames {
		if d := DeltaE(lab, c.lab); d < bestDist {
			best, bestDist = c.name, d
		}
	}
	return best
}
