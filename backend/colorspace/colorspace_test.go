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
		{"#ff00ff", Purple},
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
		name     string
		chromas  []float64
		percents []float64
		want     int
	}{
		{"empty", nil, nil, -1},
		{"all neutral", []float64{0.01, 0.03}, []float64{0.5, 0.4}, -1},
		{"largest share times chroma", []float64{0.05, 0.2, 0.1}, []float64{0.6, 0.2, 0.3}, 1},
		{"neutral never wins", []float64{0.039, 0.05}, []float64{0.9, 0.05}, 1},
		{"first on a tie", []float64{0.1, 0.2}, []float64{0.2, 0.1}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Representative(tt.chromas, tt.percents); got != tt.want {
				t.Errorf("expected %d, got %d", tt.want, got)
			}
		})
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
	tests := []struct {
		id   int
		hue  float64
		want float64
	}{
		{1, 15, 49 * 360},
		{1, 0, 49*360 + 345},
		{2, 100, 98*360 + 85},
		{40211, 359.5, float64((int64(40211)*2654435761)%128*360) + float64(float32(359.5)) - 15},
	}
	for _, tt := range tests {
		if got := RainbowKey(tt.id, tt.hue); got != tt.want {
			t.Errorf("RainbowKey(%d, %g): expected %g, got %g", tt.id, tt.hue, tt.want, got)
		}
	}
}
