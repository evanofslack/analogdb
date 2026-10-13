package analogdb

import "context"

type ScrapeService interface {
	// CaptionMissingPostIDs returns posts with no caption, or with a
	// caption of another version when version is set
	CaptionMissingPostIDs(ctx context.Context, version *string) ([]int, error)
	// FindImageMatch returns the oldest post a new image duplicates
	FindImageMatch(ctx context.Context, q ImageMatchQuery) (ImageMatch, bool, error)
	// DuplicatePosts returns every post that duplicates an older post, paired
	// with the oldest match
	DuplicatePosts(ctx context.Context) ([]DuplicatePost, error)
	// ReportDuplicates files a duplicate report on each duplicate post without
	// an open one, and returns how many were filed
	ReportDuplicates(ctx context.Context, dupes []DuplicatePost) (int, error)
}
