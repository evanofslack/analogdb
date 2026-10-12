package analogdb

import (
	"fmt"
	"strconv"
)

// ImageHashMissing is a post with no image hash and its low res image
type ImageHashMissing struct {
	ID  int    `json:"id" example:"9156"`
	URL string `json:"url" example:"https://i.analogdb.com/low.jpeg"`
}

// ImageHash is a post's 64 bit dHash as 16 hex characters, kept as a string
// since JSON numbers lose precision past 53 bits
type ImageHash struct {
	ID   int    `json:"id" example:"9156"`
	Hash string `json:"image_hash" example:"c3a1f0e0d8b8b0f0"`
}

// DuplicatePost is a post whose image matches an older post
type DuplicatePost struct {
	KeepID       int    `json:"keep_id" example:"9156"`
	DuplicateID  int    `json:"duplicate_id" example:"36395"`
	Distance     int    `json:"distance" example:"0"`
	KeepURL      string `json:"keep_url" example:"https://i.analogdb.com/keep.jpeg"`
	DuplicateURL string `json:"duplicate_url" example:"https://i.analogdb.com/dupe.jpeg"`
}

// ParseImageHash reads 16 hex characters into the signed value stored in
// postgres, keeping the bits as they are
func ParseImageHash(s string) (int64, error) {
	if len(s) != 16 {
		return 0, &Error{Code: ERRBADREQUEST, Message: fmt.Sprintf("invalid image hash %q, want 16 hex characters", s)}
	}
	u, err := strconv.ParseUint(s, 16, 64)
	if err != nil {
		return 0, &Error{Code: ERRBADREQUEST, Message: fmt.Sprintf("invalid image hash %q, want 16 hex characters", s)}
	}
	return int64(u), nil
}

// FormatImageHash is the inverse of ParseImageHash
func FormatImageHash(h int64) string {
	return fmt.Sprintf("%016x", uint64(h))
}
