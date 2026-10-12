package analogdb

import "testing"

func TestParseImageHash(t *testing.T) {
	for _, s := range []string{"0000000000000000", "c3a1f0e0d8b8b0f0", "ffffffffffffffff", "7fffffffffffffff"} {
		h, err := ParseImageHash(s)
		if err != nil {
			t.Fatalf("%s: %v", s, err)
		}
		if got := FormatImageHash(h); got != s {
			t.Errorf("want %s round trip, got %s", s, got)
		}
	}
	if h, _ := ParseImageHash("ffffffffffffffff"); h != -1 {
		t.Errorf("want top bit kept as -1, got %d", h)
	}
	if h, _ := ParseImageHash("C3A1F0E0D8B8B0F0"); FormatImageHash(h) != "c3a1f0e0d8b8b0f0" {
		t.Error("want upper case accepted")
	}
	for _, s := range []string{"", "abc", "c3a1f0e0d8b8b0f", "c3a1f0e0d8b8b0f00", "g3a1f0e0d8b8b0f0", "-3a1f0e0d8b8b0f0"} {
		if _, err := ParseImageHash(s); ErrorCode(err) != ERRBADREQUEST {
			t.Errorf("%q: want bad request, got %v", s, err)
		}
	}
}

func TestIsDuplicateImage(t *testing.T) {
	keep := ImageFacts{ID: 1, Author: "u/a", Width: 2000, Height: 3000}
	other := func(author string, w, h int) ImageFacts {
		return ImageFacts{ID: 2, Author: author, Width: w, Height: h}
	}
	tests := []struct {
		name     string
		dup      ImageFacts
		distance int
		want     bool
	}{
		{"exact, other author", other("u/b", 1000, 1500), 0, true},
		{"1 bit, other author", other("u/b", 1000, 1500), 1, false},
		{"2 bits, same author", other("u/a", 1000, 1500), 2, true},
		{"3 bits, same author", other("u/a", 1000, 1500), 3, false},
		{"aspect within 3%", other("u/a", 1000, 1530), 0, true},
		{"aspect off by 5%", other("u/a", 1000, 1580), 0, false},
		{"rotated", other("u/a", 3000, 2000), 0, false},
		{"no size", other("u/a", 0, 0), 0, false},
	}
	for _, tt := range tests {
		if got := IsDuplicateImage(keep, tt.dup, tt.distance); got != tt.want {
			t.Errorf("%s: want %v, got %v", tt.name, tt.want, got)
		}
	}
}
