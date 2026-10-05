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
	// FindKeyword returns the post count and top posts of one tag
	FindKeyword(ctx context.Context, word string, topPosts int) (*KeywordDetail, error)
	// CoOccurring returns the number of posts each other tag shares with the word
	CoOccurring(ctx context.Context, word string) (map[string]int, error)
}

type KeywordFilter struct {
	Limit *int
	// Days counts only tags of posts created in the last n days
	Days *int
	// TopPosts attaches this many top scoring posts to each keyword
	TopPosts *int
	// MinCount keeps only keywords on at least this many posts
	MinCount *int
}

type KeywordSummary struct {
	Word     string        `json:"word" example:"baseball"`
	Count    int           `json:"count" example:"207"`
	TopPosts []CatalogPost `json:"top_posts,omitempty"`
}

type KeywordDetail struct {
	Word     string
	Count    int
	TopPosts []CatalogPost
}
