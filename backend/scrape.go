package analogdb

import "context"

type ScrapeService interface {
	KeywordUpdatedPostIDs(ctx context.Context) ([]int, error)
	// CaptionMissingPostIDs returns posts with no caption, or with a
	// caption of another version when version is set
	CaptionMissingPostIDs(ctx context.Context, version *string) ([]int, error)
}
