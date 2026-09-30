package analogdb

import (
	"context"
	"encoding/json"
	"time"
)

// PostExtraction is the stored metadata extraction for one post: the text the
// LLM saw, its raw output, and the camera and film mentions not in the catalog.
// Keeping the raw output lets posts be matched again without new LLM calls.
type PostExtraction struct {
	PostID           int             `json:"post_id" example:"1234"`
	ExtractorVersion string          `json:"extractor_version" example:"2026-10-a"`
	Model            string          `json:"model" example:"google/gemini-2.5-flash-lite"`
	Input            string          `json:"input,omitempty" example:"title: Dusk [Nikon FM, Portra 400]"`
	InputHash        string          `json:"input_hash" example:"9f2c..."`
	Raw              json.RawMessage `json:"raw,omitempty" swaggertype:"object"`
	Unmatched        json.RawMessage `json:"unmatched" swaggertype:"array,object"`
	Created          time.Time       `json:"created"`
	Updated          time.Time       `json:"updated"`
}

type ExtractionFilter struct {
	HasUnmatched *bool
	// Kind limits to extractions with an unmatched mention of this kind
	Kind *string
	// Key limits to extractions with an unmatched mention with this normalized key
	Key *string
	// Full includes the input text and raw output
	Full     bool
	BeforeID *int
	Limit    int
}

type ExtractionService interface {
	// UpsertExtractions writes extractions, returning how many were written
	// and the post ids skipped because the post doesn't exist
	UpsertExtractions(ctx context.Context, extractions []*PostExtraction) (int, []int, error)
	FindExtractions(ctx context.Context, filter *ExtractionFilter) ([]*PostExtraction, error)
}
