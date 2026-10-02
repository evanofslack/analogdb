package analogdb

import "encoding/json"

// PostCaption is the vision caption of a post with the model and prompt
// version that wrote it. Only the caption text is shown on posts.
type PostCaption struct {
	Caption *string         `json:"caption,omitempty" example:"A woman on a beach at sunset"`
	Model   string          `json:"model" example:"google/gemini-2.5-flash-lite"`
	Version string          `json:"version" example:"v1"`
	Raw     json.RawMessage `json:"raw" swaggertype:"object"`
}
