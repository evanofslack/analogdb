package analogdb

import (
	"fmt"
	"math"
	"strconv"
)

// DuplicatePost is a post whose image matches an older post
type DuplicatePost struct {
	KeepID       int    `json:"keep_id" example:"9156"`
	DuplicateID  int    `json:"duplicate_id" example:"36395"`
	Distance     int    `json:"distance" example:"0"`
	SameAuthor   bool   `json:"same_author" example:"true"`
	KeepURL      string `json:"keep_url" example:"https://i.analogdb.com/keep.jpeg"`
	DuplicateURL string `json:"duplicate_url" example:"https://i.analogdb.com/dupe.jpeg"`
}

// The duplicate rule, tuned against a hand check of 584 pairs within 4 bits.
// Matches up to 2 bits were mostly real, while 3 and 4 bits were mostly
// different photos unless posted by the same author. Every match is reported
// for a person to check, so the rule catches nearly all reposts and accepts
// about one false report in four. Plain images such as fog or sky land near
// many others, and a hash with that many neighbours is ignored.
const (
	DuplicateAnyAuthorBits   = 2
	DuplicateSameAuthorBits  = 4
	DuplicateAspectTolerance = 0.05
	DuplicateCrowdBits       = 4
	DuplicateCrowdSize       = 8
)

// ImageFacts are the fields of a post the duplicate rule compares
type ImageFacts struct {
	ID     int
	Author string
	Width  int
	Height int
}

// ImageMatchQuery is a new post to check against existing posts
type ImageMatchQuery struct {
	Hash      int64
	Grayscale bool
	Author    string
	Width     int
	Height    int
}

// ImageMatch is the oldest existing post a new post duplicates
type ImageMatch struct {
	ID         int
	Distance   int
	SameAuthor bool
}

// IsDuplicateImage applies the duplicate rule to two posts whose hashes are
// distance bits apart and whose greyscale flags already match
func IsDuplicateImage(keep, dup ImageFacts, distance int) bool {
	if keep.Width <= 0 || keep.Height <= 0 || dup.Width <= 0 || dup.Height <= 0 {
		return false
	}
	ratio := float64(keep.Width*dup.Height) / float64(keep.Height*dup.Width)
	if math.Abs(ratio-1) >= DuplicateAspectTolerance {
		return false
	}
	if distance <= DuplicateAnyAuthorBits {
		return true
	}
	return distance <= DuplicateSameAuthorBits && keep.Author == dup.Author
}

// DuplicateReportMessage is the message on a report filed for a duplicate,
// such as "duplicate of post 9156, 1 bit apart, other author"
func DuplicateReportMessage(d DuplicatePost) string {
	bits := "bits"
	if d.Distance == 1 {
		bits = "bit"
	}
	author := "other author"
	if d.SameAuthor {
		author = "same author"
	}
	return fmt.Sprintf("duplicate of post %d, %d %s apart, %s", d.KeepID, d.Distance, bits, author)
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
