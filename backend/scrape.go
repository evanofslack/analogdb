package analogdb

import "context"

type ScrapeService interface {
	KeywordUpdatedPostIDs(ctx context.Context) ([]int, error)
	// CaptionMissingPostIDs returns posts with no caption, or with a
	// caption of another version when version is set
	CaptionMissingPostIDs(ctx context.Context, version *string) ([]int, error)
	// ImageHashMissingPosts returns posts with no image hash, oldest id first
	ImageHashMissingPosts(ctx context.Context) ([]ImageHashMissing, error)
	// SetImageHashes stores image hashes without touching the posts' updated
	// time, and returns how many posts were written
	SetImageHashes(ctx context.Context, hashes []ImageHash) (int, error)
	// FindImageMatch returns the oldest post a new image duplicates
	FindImageMatch(ctx context.Context, q ImageMatchQuery) (ImageMatch, bool, error)
	// DuplicatePosts returns every post that duplicates an older post, paired
	// with the oldest match
	DuplicatePosts(ctx context.Context) ([]DuplicatePost, error)
	// ReportDuplicates files a duplicate report on each duplicate post without
	// an open one, and returns how many were filed
	ReportDuplicates(ctx context.Context, dupes []DuplicatePost) (int, error)
}
