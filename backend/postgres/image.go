package postgres

import "strings"

const defaultImageBase = "https://i.analogdb.com"

// imageHosts serve the same bucket, so their URLs are stored as bare keys
var imageHosts = []string{"https://i.analogdb.com/", "https://d3i73ktnzbi69i.cloudfront.net/"}

// imageURL builds the URL of a stored key. Full URLs pass through unchanged.
func imageURL(base, stored string) string {
	if stored == "" || strings.HasPrefix(stored, "https://") || strings.HasPrefix(stored, "http://") {
		return stored
	}
	return base + "/" + stored
}

// imageKey strips a known image host from url. Other URLs are kept whole.
func imageKey(base, url string) string {
	for _, prefix := range append([]string{base + "/"}, imageHosts...) {
		if key, ok := strings.CutPrefix(url, prefix); ok && key != "" {
			return key
		}
	}
	return url
}
