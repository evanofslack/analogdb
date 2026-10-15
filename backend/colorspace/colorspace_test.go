package colorspace

import (
	"math"
	"testing"
)

func TestBucket(t *testing.T) {
	tests := []struct {
		hex  string
		want string
	}{
		{"#ff0000", Red},
		{"#800000", Red},
		{"#c0392b", Red},
		{"#0000ff", Navy},
		{"#4682b4", Navy},
		{"#808080", Gray},
		{"#cccccc", Gray},
		{"#000", ""},
		{"#000000", Black},
		{"#2c2c2c", Black},
		{"#ffffff", White},
		{"#d2b48c", Tan},
		{"#f5deb3", Tan},
		{"#8b4513", Brown},
		{"#a0522d", Brown},
		{"#808000", Olive},
		{"#556b2f", Olive},
		{"#008080", Teal},
		{"#00ffff", Teal},
		{"#ffa500", Orange},
		{"#ffff00", Yellow},
		{"#ffd700", Yellow},
		{"#008000", Green},
		{"#006400", Green},
		{"#800080", Purple},
		{"#ff00ff", Pink},
		{"#ffc0cb", Pink},
		{"#ff69b4", Pink},
		{"#c71585", Pink},
		{"#db7093", Red},
		{"#cc5500", Orange},
		{"#b5651d", Orange},
		{"#160113", Black},
		{"#140a1f", Black},
		{"#4a5560", Gray},
	}
	for _, tt := range tests {
		t.Run(tt.hex, func(t *testing.T) {
			d, err := Derive(tt.hex)
			if tt.want == "" {
				if err == nil {
					t.Fatalf("expected an error for %s", tt.hex)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if d.Html != tt.want {
				lch := d.ToLCh()
				t.Errorf("expected %s, got %s (L=%.3f C=%.3f H=%.1f)", tt.want, d.Html, lch.L, lch.C, lch.H)
			}
		})
	}
}

func TestBucketsHaveThresholds(t *testing.T) {
	if len(Buckets) != len(BucketThresholds) {
		t.Fatalf("expected %d thresholds, got %d", len(Buckets), len(BucketThresholds))
	}
	for _, b := range Buckets {
		if !IsBucket(b) {
			t.Errorf("missing threshold for %s", b)
		}
	}
}

func TestLabRoundTrip(t *testing.T) {
	for _, hex := range []string{"000000", "ffffff", "ff0000", "00ff00", "0000ff", "837d5c", "c0392b", "123456"} {
		c, err := ParseHex(hex)
		if err != nil {
			t.Fatal(err)
		}
		if got := c.ToLab().ToRGB(); got != c {
			t.Errorf("%s: round trip gave %s", hex, got.Hex())
		}
		lab := c.ToLab()
		if back := lab.ToLCh().ToLab(); DeltaE(lab, back) > 1e-9 {
			t.Errorf("%s: LCh round trip moved %g", hex, DeltaE(lab, back))
		}
	}
}

func TestLabReference(t *testing.T) {
	// reference values from Ottosson's OKLab post
	c, _ := ParseHex("ffffff")
	if lab := c.ToLab(); math.Abs(lab.L-1) > 1e-4 || math.Abs(lab.A) > 1e-4 || math.Abs(lab.B) > 1e-4 {
		t.Errorf("white: %+v", lab)
	}
	c, _ = ParseHex("ff0000")
	lab := c.ToLab()
	if math.Abs(lab.L-0.62796) > 1e-3 || math.Abs(lab.A-0.22486) > 1e-3 || math.Abs(lab.B-0.12585) > 1e-3 {
		t.Errorf("red: %+v", lab)
	}
}

func TestDeltaE(t *testing.T) {
	if d := DeltaE(Lab{0.5, 0.1, 0.1}, Lab{0.5, 0.1, 0.1}); d != 0 {
		t.Errorf("expected 0, got %g", d)
	}
	if d := DeltaE(Lab{0, 0, 0}, Lab{0.3, 0, 0.4}); math.Abs(d-0.5) > 1e-12 {
		t.Errorf("expected 0.5, got %g", d)
	}
}

func TestNormalizeQuery(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"red", "red", false},
		{"RED", "red", false},
		{" teal ", "teal", false},
		{"c0392b", "bb3333", false},
		{"#C0392B", "bb3333", false},
		{"#ffffff", "ffffff", false},
		{"000000", "000000", false},
		{"090909", "111111", false},
		{"080808", "000000", false},
		{"fff", "", true},
		{"blue", "", true},
		{"zzzzzz", "", true},
		{"#12345", "", true},
		{"+12345", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := NormalizeQuery(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error %t, got %v", tt.wantErr, err)
			}
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestSnapValues(t *testing.T) {
	seen := map[uint8]bool{}
	for v := 0; v < 256; v++ {
		s := RGB{R: uint8(v)}.Snap().R
		if s%17 != 0 || math.Abs(float64(s)-float64(v)) > 8.5 {
			t.Fatalf("%d snapped to %d", v, s)
		}
		seen[s] = true
	}
	if len(seen) != 16 {
		t.Errorf("expected 16 levels, got %d", len(seen))
	}
}

func TestDefaultThreshold(t *testing.T) {
	if got := DefaultThreshold(Red, 1); got != BucketThresholds[Red] {
		t.Errorf("expected %g, got %g", BucketThresholds[Red], got)
	}
	if got := DefaultThreshold("cc3333", 1); got != HexThreshold {
		t.Errorf("expected %g, got %g", HexThreshold, got)
	}
	if got := DefaultThreshold(Gray, 3); got != BucketThresholds[Gray]/3 {
		t.Errorf("expected %g, got %g", BucketThresholds[Gray]/3, got)
	}
}

func TestNearestCSS(t *testing.T) {
	tests := map[string]string{
		"ff0000": "red",
		"000000": "black",
		"8b4513": "saddlebrown",
		"d2b48c": "tan",
		"00ffff": "aqua",
		"808080": "gray",
	}
	for hex, want := range tests {
		c, _ := ParseHex(hex)
		if got := NearestCSS(c.ToLab()); got != want {
			t.Errorf("%s: expected %s, got %s", hex, want, got)
		}
	}
}

func TestRepresentative(t *testing.T) {
	tests := []struct {
		name    string
		palette []Swatch
		want    int
	}{
		{"empty", nil, -1},
		{"all neutral", []Swatch{{"808080", 0.5}, {"000000", 0.4}}, -1},
		{"largest share times chroma", []Swatch{{"c0392b", 0.1}, {"0000ff", 0.2}, {"008080", 0.3}}, 1},
		{"neutral never wins", []Swatch{{"7f8080", 0.9}, {"008080", 0.05}}, 1},
		{"tinted shadow skipped", []Swatch{{"160113", 0.15}, {"f9491e", 0.1}}, 1},
		{"first on a tie", []Swatch{{"ff0000", 0.2}, {"ff0000", 0.2}}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			derived, percents, err := derivePalette(tt.palette)
			if err != nil {
				t.Fatal(err)
			}
			if got := Representative(derived, percents); got != tt.want {
				t.Errorf("expected %d, got %d", tt.want, got)
			}
		})
	}
}

func TestPalettePostColor(t *testing.T) {
	red, _ := Derive("f9491e")
	shadow, _ := Derive("160113")
	if shadow.Html != Black || shadow.Chroma < NeutralChroma {
		t.Fatalf("expected a tinted black shadow, got %s %g", shadow.Html, shadow.Chroma)
	}
	white, _ := Derive("ffffff")
	gray, _ := Derive("808080")
	black, _ := Derive("000000")

	tests := []struct {
		name      string
		palette   []Swatch
		grayscale bool
		ok        bool
		want      PostColor
	}{
		{"chromatic", []Swatch{{"160113", 0.15}, {"f9491e", 0.1}}, false, true, PostColor{Hue: red.Hue, Chroma: red.Chroma, Strength: 0.1 * red.Chroma}},
		{"white", []Swatch{{"ffffff", 0.5}, {"808080", 0.3}}, false, true, PostColor{Neutral: true, Bucket: White, Strength: 0.5, Lightness: (0.5*white.L + 0.3*gray.L) / 0.8}},
		{"gray", []Swatch{{"808080", 0.7}, {"000000", 0.3}}, false, true, PostColor{Neutral: true, Bucket: Gray, Strength: 0.7, Lightness: 0.7*gray.L + 0.3*black.L}},
		{"black", []Swatch{{"000000", 0.6}}, false, true, PostColor{Neutral: true, Bucket: Black, Strength: 0.6, Lightness: black.L}},
		{"white below minimum", []Swatch{{"ffffff", 0.39}, {"808080", 0.3}, {"000000", 0.3}}, false, false, PostColor{}},
		{"gray below minimum", []Swatch{{"808080", 0.69}, {"000000", 0.31}}, false, false, PostColor{}},
		{"black below minimum", []Swatch{{"000000", 0.49}, {"808080", 0.48}}, false, false, PostColor{}},
		{"grayscale never chromatic", []Swatch{{"f9491e", 0.4}, {"000000", 0.6}}, true, true, PostColor{Neutral: true, Bucket: Black, Strength: 0.6, Lightness: black.L}},
		{"grayscale below minimum", []Swatch{{"f9491e", 0.6}, {"000000", 0.4}}, true, false, PostColor{}},
		{"empty", nil, false, false, PostColor{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok, err := PalettePostColor(tt.palette, tt.grayscale)
			if err != nil {
				t.Fatal(err)
			}
			if ok != tt.ok || math.Abs(got.Hue-tt.want.Hue) > 1e-9 || math.Abs(got.Chroma-tt.want.Chroma) > 1e-9 ||
				math.Abs(got.Strength-tt.want.Strength) > 1e-9 || got.Neutral != tt.want.Neutral || got.Bucket != tt.want.Bucket || math.Abs(got.Lightness-tt.want.Lightness) > 1e-9 {
				t.Errorf("expected %+v %t, got %+v %t", tt.want, tt.ok, got, ok)
			}
		})
	}
	if _, _, err := PalettePostColor([]Swatch{{Hex: "nope"}}, false); err == nil {
		t.Error("expected an error for a bad hex")
	}
}

func TestPalette(t *testing.T) {
	palette := []Swatch{
		{Hex: "#c0392b", Percent: 0.3},
		{Hex: "#ff0000", Percent: 0.1},
		{Hex: "#808080", Percent: 0.4},
		{Hex: "#008080", Percent: 0.2},
	}
	shares, err := BucketShares(palette)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(shares[Red]-0.4) > 1e-9 || shares[Gray] != 0.4 || shares[Teal] != 0.2 {
		t.Errorf("unexpected shares %v", shares)
	}

	share, err := HexShare(palette, "bb3333", MatchRadius)
	if err != nil {
		t.Fatal(err)
	}
	if share != 0.3 {
		t.Errorf("expected hex share 0.3, got %g", share)
	}

	hue, chroma, ok, err := PaletteRepresentative(palette)
	if err != nil || !ok {
		t.Fatalf("expected a representative, got %t %v", ok, err)
	}
	d, _ := Derive("c0392b")
	if hue != d.Hue || chroma != d.Chroma {
		t.Errorf("expected the firebrick cluster, got hue %g chroma %g", hue, chroma)
	}

	if _, _, ok, _ := PaletteRepresentative([]Swatch{{Hex: "000000", Percent: 1}}); ok {
		t.Error("expected no representative for a neutral palette")
	}
	if _, err := BucketShares([]Swatch{{Hex: "nope"}}); err == nil {
		t.Error("expected an error for a bad hex")
	}
}

func TestRainbowKey(t *testing.T) {
	if RainbowTotalLoops() != 252 {
		t.Fatalf("expected 252 loops, got %d", RainbowTotalLoops())
	}
	hash := func(id, loops int) float64 {
		return float64((uint64(id) * 2654435761) % uint64(loops))
	}
	const w = 390
	tests := []struct {
		name string
		id   int
		c    PostColor
		want float64
	}{
		{"strongest tier at the loop start", 1, PostColor{Hue: 15, Strength: 0.05}, hash(1, 9) * w},
		{"hue wraps to the loop end", 1, PostColor{Hue: 0, Strength: 0.03}, hash(1, 9)*w + 345},
		{"second tier", 2, PostColor{Hue: 100, Strength: 0.0299}, (9+hash(2, 20))*w + 85},
		{"third tier at its minimum", 3, PostColor{Hue: 15, Strength: 0.015}, (29 + hash(3, 31)) * w},
		{"fourth tier", 4, PostColor{Hue: 15, Strength: 0.012}, (60 + hash(4, 43)) * w},
		{"fifth tier", 5, PostColor{Hue: 15, Strength: 0.009}, (103 + hash(5, 45)) * w},
		{"float32 tier minimum", 6, PostColor{Hue: 15, Strength: float64(float32(0.02))}, (9 + hash(6, 20)) * w},
		{"weakest tier", 6, PostColor{Hue: 15, Strength: 0.0089}, (148 + hash(6, 104)) * w},
		{"zero strength", 7, PostColor{Hue: 15}, (148 + hash(7, 104)) * w},
		{"float32 hue", 40211, PostColor{Hue: 359.5, Strength: 0.1}, hash(40211, 9)*w + float64(float32(359.5)) - 15},
		{"white tail", 8, PostColor{Neutral: true, Bucket: White, Lightness: 1}, hash(8, 23)*w + 360},
		{"black tail", 8, PostColor{Neutral: true, Bucket: Black, Lightness: 0}, hash(8, 172)*w + 390},
		{"gray tail", 8, PostColor{Neutral: true, Bucket: Gray, Lightness: 0.5}, hash(8, 186)*w + 375},
		{"lightness clamped", 8, PostColor{Neutral: true, Bucket: White, Lightness: 1.2}, hash(8, 23)*w + 360},
	}
	for _, tt := range tests {
		if got := RainbowKey(tt.id, tt.c); got != tt.want {
			t.Errorf("%s: expected %g, got %g", tt.name, tt.want, got)
		}
	}

	for _, bucket := range []string{White, Gray, Black} {
		if n := RainbowNeutralLoops(bucket); n < 1 || n > RainbowTotalLoops() {
			t.Errorf("%s: %d loops out of range", bucket, n)
		}
	}
	for id := 1; id < 2000; id++ {
		for _, c := range []PostColor{{Hue: 359.99, Strength: 1}, {Hue: 14.99}, {Neutral: true, Bucket: White, Lightness: 1}, {Neutral: true, Bucket: Gray, Lightness: 0.5}, {Neutral: true, Bucket: Black, Lightness: 0.01}} {
			key := RainbowKey(id, c)
			loop := math.Floor(key / w)
			loops := 252.0
			if c.Neutral {
				loops = float64(RainbowNeutralLoops(c.Bucket))
			}
			if loop < 0 || loop >= loops {
				t.Fatalf("id %d %+v: loop %g out of range", id, c, loop)
			}
			within := key - loop*w
			if c.Neutral && (within < 360 || within > 390) || !c.Neutral && within >= 360 {
				t.Fatalf("id %d %+v: unexpected place %g in its loop", id, c, within)
			}
		}
	}
}
