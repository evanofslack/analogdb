package analogdb

import "context"

// Keyword represents a single word/tag for a post
type Keyword struct {
    Word   string  `json:"word" example:"baseball"`
    Weight float64 `json:"weight" example:"0.43"`
}

type KeywordService interface {
	// FindKeywords(ctx context.Context, filter *KeywordFilter) ([]string, error)
	GetKeywordSummary(ctx context.Context, limit int) (*[]KeywordSummary, error)
}

type KeywordFilter struct {
	Limit *int
}

type KeywordSummary struct {
    Word  string `json:"word" example:"baseball"`
    Count int    `json:"count" example: "207"`
}
