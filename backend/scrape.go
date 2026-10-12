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
	// FindImageMatch returns the oldest post whose image hash is within
	// maxBits of hash
	FindImageMatch(ctx context.Context, hash int64, grayscale bool, maxBits int) (int, bool, error)
	// DuplicatePosts returns every post with an older image within maxBits,
	// paired with the oldest match
	DuplicatePosts(ctx context.Context, maxBits int) ([]DuplicatePost, error)
}
