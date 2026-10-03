package analogdb

import "context"

type SearchFilter struct {
	Query     string
	Expanded  string
	Image     []byte
	Nsfw      *bool
	Grayscale *bool
	Sprocket  *bool
	Limit     int
	Offset    int
}

type SearchHit struct {
	PostID int      `json:"post_id"`
	Tags   []string `json:"tags"`
}

type SearchService interface {
	SearchText(ctx context.Context, filter *SearchFilter) (hits []SearchHit, hasMore bool, err error)
	SearchImage(ctx context.Context, filter *SearchFilter) ([]SearchHit, error)
}
