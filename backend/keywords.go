package analogdb

import "context"

// Keyword represents a single word/tag for a post
type Keyword struct {
	Word   string  `json:"word" example:"baseball"`
	Weight float64 `json:"weight" example:"0.43"`
}

type KeywordService interface {
	// FindKeywords(ctx context.Context, filter *KeywordFilter) ([]string, error)
	GetKeywordSummary(ctx context.Context, filter *KeywordFilter) (*[]KeywordSummary, error)
	// TagCounts returns the number of posts per tag and the number of posts with any tag
	TagCounts(ctx context.Context) (map[string]int, int, error)
}

type KeywordFilter struct {
	Limit *int
	// Days counts only tags of posts created in the last n days
	Days *int
}

type KeywordSummary struct {
	Word  string `json:"word" example:"baseball"`
	Count int    `json:"count" example:"207"`
}
