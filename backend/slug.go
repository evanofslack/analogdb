package analogdb

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Slug joins the parts into a lowercase, url safe string with accents removed
func Slug(parts ...string) string {
	decomposed := norm.NFKD.String(strings.Join(parts, " "))
	var b strings.Builder
	dash := false
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		r = unicode.ToLower(r)
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
			continue
		}
		if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}
