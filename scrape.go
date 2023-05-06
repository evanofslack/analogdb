package analogdb

import "context"

type ScrapeService interface {
	KeywordUpdatedPostIDs(ctx context.Context) ([]int, error)
}
