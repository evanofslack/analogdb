package analogdb

import "context"

const (
	StatsVectorDims  = 512
	StatsRoleBorder  = "border"
	StatsPromptBase  = "base"
	StatsMinThemes   = 2
	StatsMaxThemes   = 64
	StatsMaxThemeLen = 60
)

// StatsPromptKeys are the style prompts the stats builder scores against,
// base is the plain "a photo" reference the others are measured from
var StatsPromptKeys = []string{StatsPromptBase, "vivid", "teal_orange", "red_night", "grain", "crisp"}

// StatsSeed holds the inputs to the stats builder that are fit offline and
// curated by hand: visual theme centers and style prompt vectors
type StatsSeed struct {
	Version string        `json:"version"`
	Model   string        `json:"model"`
	Themes  []StatsTheme  `json:"themes"`
	Prompts []StatsPrompt `json:"prompts"`
}

type StatsTheme struct {
	Key    string    `json:"key"`
	Name   string    `json:"name"`
	Hidden bool      `json:"hidden"`
	Role   *string   `json:"role"`
	Center []float32 `json:"center"`
}

type StatsPrompt struct {
	Key    string    `json:"key"`
	Label  string    `json:"label"`
	Text   string    `json:"text"`
	Vector []float32 `json:"vector"`
}

type StatsService interface {
	ReplaceSeed(ctx context.Context, seed *StatsSeed) error
	FindSeed(ctx context.Context) (*StatsSeed, error)
}
