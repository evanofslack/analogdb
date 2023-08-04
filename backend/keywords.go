package analogdb

import "context"


// Keyword represents a single word/tag for a post
type Keyword struct {
	Word   string  `json:"word"`
	Weight float64 `json:"weight"`
}

type KeywordService interface {
	// FindKeywords(ctx context.Context, filter *KeywordFilter) ([]string, error)
	GetKeywordSummary(ctx context.Context, limit int) (*[]KeywordSummary, error)
}

type KeywordFilter struct {
	Limit *int
}

type KeywordSummary struct {
	Word  string `json:"word"`
	Count int    `json:"count"`
}
