package analogdb

import "context"

type KeywordService interface {
	// FindKeywords(ctx context.Context, filter *KeywordFilter) ([]string, error)
	GetKeywordSummary(ctx context.Context, limit int) (*[]KeywordSummary, error)
}

type KeywordFilter struct {
	Limit *int
}

type KeywordSummary struct {
	Word      string  `json:"word"`
	Count     int     `json:"count"`
	AvgWeight float64 `json:"avg_weight"`
}
