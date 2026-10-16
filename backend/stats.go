package analogdb

import (
	"context"
	"encoding/json"
	"time"
)

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

// StatsInputPost is one post in the stats window with what the builder needs.
// Film and camera fields are empty when the post doesn't match the catalog.
type StatsInputPost struct {
	ID            int
	Author        string
	Time          int64
	Score         int
	Nsfw          bool
	Greyscale     bool
	FilmSlug      string
	FilmMake      string
	FilmType      string
	FilmColorType string
	CameraSlug    string
	CameraMake    string
	CameraModel   string
	FocalLength   int
	Aperture      string
	Title         string
	Images        []Image
	// percent rank of the score within the post's UTC month, 0 to 1
	Popularity float64
}

type StatsInputColor struct {
	PostID  int
	Percent float64
	L       float64
	A       float64
	B       float64
	Chroma  float64
	Hue     float64
}

type StatsInputKeyword struct {
	PostID int
	Word   string
}

type StatsInput struct {
	Posts    []StatsInputPost
	Colors   []StatsInputColor
	Keywords []StatsInputKeyword
}

// StatsSnapshot is one stored build, Body is the marshalled Stats
type StatsSnapshot struct {
	ID          int64           `json:"id"`
	Created     time.Time       `json:"created"`
	SeedVersion string          `json:"seed_version"`
	BuildMs     int             `json:"build_ms"`
	Size        int             `json:"size"`
	Body        json.RawMessage `json:"-"`
}

type StatsService interface {
	ReplaceSeed(ctx context.Context, seed *StatsSeed) error
	FindSeed(ctx context.Context) (*StatsSeed, error)
	LoadStatsInput(ctx context.Context, since time.Time, subjects []string) (*StatsInput, error)
	CreateSnapshot(ctx context.Context, snapshot *StatsSnapshot) error
	FindLatestSnapshot(ctx context.Context) (*StatsSnapshot, error)
	FindSnapshots(ctx context.Context) ([]*StatsSnapshot, error)
}

// VectorScanner pages through every stored image vector
type VectorScanner interface {
	ScanVectors(ctx context.Context, fn func(postID int, vector []float32) error) error
}

// Stats is the document behind the stats page
type Stats struct {
	Version     int               `json:"version" example:"1"`
	Generated   time.Time         `json:"generated"`
	SeedVersion string            `json:"seed_version" example:"2026-10-16T20:11:27+00:00"`
	Window      StatsWindow       `json:"window"`
	Totals      StatsTotals       `json:"totals"`
	Films       []StatsFilm       `json:"films"`
	Families    []StatsFamily     `json:"families"`
	Subjects    StatsSubjects     `json:"subjects"`
	Trends      StatsTrends       `json:"trends"`
	Themes      []StatsThemeEntry `json:"themes"`
	Popularity  StatsPopularity   `json:"popularity"`
	Posting     StatsPosting      `json:"posting"`
	Method      StatsMethod       `json:"method"`
}

type StatsWindow struct {
	From string `json:"from" example:"2022-01-01"`
	To   string `json:"to" example:"2026-10-17"`
}

type StatsTotals struct {
	Photos        int `json:"photos" example:"24531"`
	Photographers int `json:"photographers" example:"9080"`
	Films         int `json:"films" example:"212"`
	Cameras       int `json:"cameras" example:"640"`
}

// StatsPost is a post shown on the stats page
type StatsPost struct {
	ID     int     `json:"id" example:"1"`
	Title  string  `json:"title" example:"Sunset over the lake"`
	Score  int     `json:"score" example:"150"`
	Nsfw   bool    `json:"nsfw" example:"false"`
	Images []Image `json:"images"`
}

type StatsBands struct {
	Warm    float64 `json:"warm" example:"0.59"`
	Green   float64 `json:"green" example:"0.13"`
	Cool    float64 `json:"cool" example:"0.25"`
	Magenta float64 `json:"magenta" example:"0.03"`
}

type StatsSimilar struct {
	Slug       string  `json:"slug" example:"kodak-gold-200"`
	Similarity float64 `json:"similarity" example:"0.83"`
}

type StatsPosition struct {
	X float64 `json:"x" example:"0.21"`
	Y float64 `json:"y" example:"-0.44"`
}

type StatsFilm struct {
	Slug       string             `json:"slug" example:"kodak-portra-400"`
	Make       string             `json:"make" example:"kodak"`
	Type       string             `json:"type" example:"portra 400"`
	ColorType  string             `json:"color_type" example:"color"`
	Family     string             `json:"family" example:"portrait"`
	PostCount  int                `json:"post_count" example:"4084"`
	Popularity float64            `json:"popularity" example:"57.1"`
	Lightness  float64            `json:"lightness" example:"0.549"`
	Chroma     float64            `json:"chroma" example:"0.034"`
	Hue        []float64          `json:"hue"`
	Bands      *StatsBands        `json:"bands"`
	Palette    []string           `json:"palette" example:"#c9a27e"`
	Styles     map[string]float64 `json:"styles"`
	Similar    []StatsSimilar     `json:"similar"`
	Position   StatsPosition      `json:"position"`
	Typical    []StatsPost        `json:"typical"`
}

type StatsFamily struct {
	Key  string `json:"key" example:"landscape"`
	Name string `json:"name" example:"Landscape and slide"`
}

type StatsSubjectRow struct {
	Slug  string     `json:"slug" example:"cinestill-800t"`
	Lift  []*float64 `json:"lift"`
	Count []int      `json:"count"`
}

type StatsSubjects struct {
	Words []string          `json:"words" example:"neon"`
	Rows  []StatsSubjectRow `json:"rows"`
}

type StatsTrendItem struct {
	Slug   string    `json:"slug,omitempty" example:"kodak-gold-200"`
	Key    string    `json:"key,omitempty" example:""`
	Name   string    `json:"name" example:"Kodak gold 200"`
	Shares []float64 `json:"shares"`
}

type StatsMover struct {
	Slug   string  `json:"slug" example:"kodak-gold-200"`
	Name   string  `json:"name" example:"Kodak gold 200"`
	From   float64 `json:"from" example:"9.1"`
	To     float64 `json:"to" example:"14.6"`
	Change float64 `json:"change" example:"5.5"`
}

type StatsMovers struct {
	Up   []StatsMover `json:"up"`
	Down []StatsMover `json:"down"`
}

type StatsMoverGroups struct {
	Films   StatsMovers `json:"films"`
	Cameras StatsMovers `json:"cameras"`
}

type StatsArrival struct {
	Kind  string `json:"kind" example:"camera"`
	Slug  string `json:"slug" example:"pentax-17"`
	Name  string `json:"name" example:"Pentax 17"`
	First string `json:"first" example:"2024-07"`
	Count int    `json:"count" example:"112"`
}

type StatsTrends struct {
	Periods      []string         `json:"periods" example:"2022H1"`
	Partial      []string         `json:"partial" example:"2026H2"`
	Films        []StatsTrendItem `json:"films"`
	Cameras      []StatsTrendItem `json:"cameras"`
	FilmBrands   []StatsTrendItem `json:"film_brands"`
	CameraBrands []StatsTrendItem `json:"camera_brands"`
	Movers       StatsMoverGroups `json:"movers"`
	Arrivals     []StatsArrival   `json:"arrivals"`
	Themes       []StatsTrendItem `json:"themes"`
}

type StatsGroup struct {
	Count      int     `json:"count" example:"1009"`
	Popularity float64 `json:"popularity" example:"67.2"`
}

type StatsThemeEntry struct {
	Key        string      `json:"key" example:"t02"`
	Name       string      `json:"name" example:"Color figure and nude"`
	Count      int         `json:"count" example:"1009"`
	Popularity float64     `json:"popularity" example:"67.2"`
	Color      *StatsGroup `json:"color"`
	BW         *StatsGroup `json:"bw"`
	NsfwShare  float64     `json:"nsfw_share" example:"0.75"`
	Posts      []StatsPost `json:"posts"`
}

type StatsValueGroup struct {
	Value      string  `json:"value" example:"f/1.4"`
	Count      int     `json:"count" example:"1346"`
	Popularity float64 `json:"popularity" example:"52"`
}

type StatsFocalGroup struct {
	Value      int     `json:"value" example:"50"`
	Count      int     `json:"count" example:"4447"`
	Popularity float64 `json:"popularity" example:"51"`
}

type StatsBorder struct {
	Theme      string  `json:"theme" example:"t34"`
	Popularity float64 `json:"popularity" example:"35"`
}

type StatsPopularity struct {
	Color      float64           `json:"color" example:"53.1"`
	BW         float64           `json:"bw" example:"33.2"`
	Saturation []float64         `json:"saturation"`
	Aperture   []StatsValueGroup `json:"aperture"`
	Focal      []StatsFocalGroup `json:"focal"`
	Border     *StatsBorder      `json:"border,omitempty"`
}

type StatsPostingCell struct {
	Weekday    int     `json:"weekday" example:"1"`
	Hour       int     `json:"hour" example:"0"`
	Count      int     `json:"count" example:"120"`
	Popularity float64 `json:"popularity" example:"46"`
}

type StatsPosting struct {
	Cells []StatsPostingCell `json:"cells"`
}

type StatsMethod struct {
	MinFilmPosts   int    `json:"min_film_posts" example:"80"`
	MinCameraPosts int    `json:"min_camera_posts" example:"100"`
	AuthorCap      int    `json:"author_cap" example:"5"`
	Model          string `json:"model" example:"ViT-B-16-laion2b_s34b_b88k"`
}
