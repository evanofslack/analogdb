package server

import (
	"regexp"
	"slices"
	"strings"
	"unicode"
)

// The keep list, skipped endings and monochrome aliases mirror the tagger in
// scrape/packages/scrape/src/scrape/tagging.py, so queries match stored tags.
var singularKeep = map[string]struct{}{
	"alps": {}, "angeles": {}, "athens": {}, "atlas": {}, "canvas": {}, "christmas": {},
	"clothes": {}, "glasses": {}, "jeans": {}, "news": {}, "people": {}, "series": {},
	"shorts": {}, "species": {}, "stairs": {}, "sunglasses": {}, "texas": {}, "vegas": {},
	"wales": {},
}

var singularSkipEndings = []string{"ss", "us", "is", "ics"}

// singularIrregular holds plurals the suffix rules get wrong, matching the
// tagger's inflect output
var singularIrregular = map[string]string{
	"buses": "bus", "calves": "calf", "children": "child", "cookies": "cookie",
	"echoes": "echo", "elves": "elf", "feet": "foot", "gases": "gas", "geese": "goose",
	"halves": "half", "heroes": "hero", "hooves": "hoof", "knives": "knife",
	"leaves": "leaf", "lenses": "lens", "lives": "life", "loaves": "loaf",
	"mice": "mouse", "movies": "movie", "oxen": "ox", "potatoes": "potato",
	"scarves": "scarf", "selfies": "selfie", "shelves": "shelf", "teeth": "tooth",
	"thieves": "thief", "tomatoes": "tomato", "volcanoes": "volcano", "wives": "wife",
	"wolves": "wolf", "women": "woman", "zombies": "zombie",
}

const monochrome = "monochrome"

var monochromeAliases = []string{
	"black-and-white",
	"black and white",
	"black white",
	"greyscale",
	"grayscale",
	"b w",
	"bw",
}

var (
	nonWord         = regexp.MustCompile(`[^\p{L}\p{N}\p{M}\s-]|_`)
	monochromeAlias = regexp.MustCompile(`(^| )(` + strings.Join(monochromeAliases, "|") + `)( |$)`)
)

type searchQuery struct {
	// terms are the normalized words, as they appear in tags
	terms []string
	// expanded is the cleaned words plus any normalized words not already there
	expanded string
}

// cleanQuery lowercases and replaces non-word characters except - with spaces
func cleanQuery(q string) []string {
	q = nonWord.ReplaceAllString(strings.ToLower(q), " ")
	words := []string{}
	for _, w := range strings.Fields(q) {
		if w = strings.Trim(w, "-"); w != "" {
			words = append(words, w)
		}
	}
	return words
}

func replaceMonochrome(s string) string {
	for {
		replaced := monochromeAlias.ReplaceAllString(s, "${1}"+monochrome+"${3}")
		if replaced == s {
			return s
		}
		s = replaced
	}
}

func singularWord(w string) string {
	if len(w) <= 3 {
		return w
	}
	if _, ok := singularKeep[w]; ok {
		return w
	}
	for _, end := range singularSkipEndings {
		if strings.HasSuffix(w, end) {
			return w
		}
	}
	for _, r := range w {
		if !unicode.IsLetter(r) {
			return w
		}
	}
	if single, ok := singularIrregular[w]; ok {
		return single
	}
	switch {
	case !strings.HasSuffix(w, "s"):
		return w
	case strings.HasSuffix(w, "ies") && len(w) > 4:
		return strings.TrimSuffix(w, "ies") + "y"
	case strings.HasSuffix(w, "ches"), strings.HasSuffix(w, "shes"), strings.HasSuffix(w, "sses"),
		strings.HasSuffix(w, "xes"), strings.HasSuffix(w, "zzes"):
		return strings.TrimSuffix(w, "es")
	}
	return strings.TrimSuffix(w, "s")
}

func normalizeQuery(q string) searchQuery {
	words := cleanQuery(q)
	cleaned := strings.Join(words, " ")

	terms := []string{}
	for _, w := range strings.Fields(replaceMonochrome(cleaned)) {
		terms = append(terms, singularWord(w))
	}

	expanded := append([]string{}, words...)
	for _, t := range terms {
		if !slices.Contains(expanded, t) {
			expanded = append(expanded, t)
		}
	}
	return searchQuery{terms: terms, expanded: strings.Join(expanded, " ")}
}
