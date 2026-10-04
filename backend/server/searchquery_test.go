package server

import (
	"slices"
	"testing"
)

func TestNormalizeQuery(t *testing.T) {
	tests := []struct {
		q        string
		terms    []string
		expanded string
	}{
		{"flowers", []string{"flower"}, "flowers flower"},
		{"flower", []string{"flower"}, "flower"},
		{"glass", []string{"glass"}, "glass"},
		{"paris", []string{"paris"}, "paris"},
		{"bus", []string{"bus"}, "bus"},
		{"people", []string{"people"}, "people"},
		{"physics", []string{"physics"}, "physics"},
		{"dogs on beaches", []string{"dog", "on", "beach"}, "dogs on beaches dog beach"},
		{"black and white portrait", []string{"monochrome", "portrait"}, "black and white portrait monochrome"},
		{"Black-and-White", []string{"monochrome"}, "black-and-white monochrome"},
		{"B&W", []string{"monochrome"}, "b w monochrome"},
		{"bw street", []string{"monochrome", "street"}, "bw street monochrome"},
		{"greyscale", []string{"monochrome"}, "greyscale monochrome"},
		{"self-portrait", []string{"self-portrait"}, "self-portrait"},
		{"self-portraits", []string{"self-portraits"}, "self-portraits"},
		{"--rainy-- days!", []string{"rainy", "day"}, "rainy days day"},
		{"  New   York  ", []string{"new", "york"}, "new york"},
		{"35mm cars", []string{"35mm", "car"}, "35mm cars car"},
		{"bwa", []string{"bwa"}, "bwa"},
		{"!!!", []string{}, ""},
	}
	for _, tt := range tests {
		got := normalizeQuery(tt.q)
		if !slices.Equal(got.terms, tt.terms) {
			t.Errorf("%q: want terms %v, got %v", tt.q, tt.terms, got.terms)
		}
		if got.expanded != tt.expanded {
			t.Errorf("%q: want expanded %q, got %q", tt.q, tt.expanded, got.expanded)
		}
	}
}

// expected values are the tagger's inflect singular_noun output
func TestSingularWord(t *testing.T) {
	want := map[string]string{
		"waves": "wave", "leaves": "leaf", "knives": "knife", "wolves": "wolf", "caves": "cave",
		"gloves": "glove", "lives": "life", "shelves": "shelf", "curves": "curve", "hooves": "hoof",
		"cliffs": "cliff", "roofs": "roof", "scarves": "scarf", "dunes": "dune", "horses": "horse",
		"houses": "house", "buses": "bus", "churches": "church", "boxes": "box", "cherries": "cherry",
		"movies": "movie", "shoes": "shoe", "heroes": "hero", "potatoes": "potato", "photos": "photo",
		"children": "child", "mice": "mouse", "geese": "goose", "feet": "foot", "teeth": "tooth",
		"women": "woman", "men": "men", "cacti": "cacti", "trees": "tree", "skies": "sky",
		"cities": "city", "ladies": "lady", "pies": "pie", "sheep": "sheep", "fish": "fish",
		"kisses": "kiss", "lenses": "lens", "vases": "vase", "statues": "statue", "breezes": "breeze",
		"mountains": "mountain", "kids": "kid", "cars": "car", "dishes": "dish", "glass": "glass",
		"cactus": "cactus", "tennis": "tennis", "physics": "physics", "jeans": "jeans", "sunset": "sunset",
	}
	for plural, single := range want {
		if got := singularWord(plural); got != single {
			t.Errorf("%s: want %s, got %s", plural, single, got)
		}
	}
}
