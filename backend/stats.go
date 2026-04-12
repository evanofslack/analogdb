package analogdb

import (
	"context"
	"time"
)

type StatsOverview struct {
	TotalPosts    int     `json:"total_posts" example:"42000"`
	TotalAuthors  int     `json:"total_authors" example:"8206"`
	TotalCameras  int     `json:"total_cameras" example:"380"`
	TotalFilms    int     `json:"total_films" example:"210"`
	TotalKeywords int     `json:"total_keywords" example:"41492"`
	AvgScore      float64 `json:"avg_score" example:"853.2"`
	MedianScore   float64 `json:"median_score" example:"720.0"`
	MinScore      float64 `json:"min_score" example:"1.0"`
	MaxScore      float64 `json:"max_score" example:"98423.0"`
	StdDevScore   float64 `json:"std_dev_score" example:"1240.5"`
}

type StatsPeriod struct {
	Period   string  `json:"period" example:"2024-03"`
	Count    int     `json:"count" example:"412"`
	AvgScore float64 `json:"avg_score" example:"731.5"`
}

type StatsFilm struct {
	FilmMake  string  `json:"film_make" example:"kodak"`
	FilmType  string  `json:"film_type" example:"portra 400"`
	FilmSpeed int     `json:"film_speed" example:"400"`
	ColorType string  `json:"color_type" example:"color"`
	PostCount int     `json:"post_count" example:"1820"`
	AvgScore  float64 `json:"avg_score" example:"942.1"`
}

type StatsCamera struct {
	CameraMake  string  `json:"camera_make" example:"nikon"`
	CameraModel string  `json:"camera_model" example:"fm2"`
	PostCount   int     `json:"post_count" example:"2341"`
	AvgScore    float64 `json:"avg_score" example:"880.4"`
}

type StatsColor struct {
	HtmlName   string  `json:"html_name" example:"gray"`
	Hex        string  `json:"hex" example:"#837d5c"`
	PostCount  int     `json:"post_count" example:"5100"`
	AvgPercent float64 `json:"avg_percent" example:"0.38"`
	AvgScore   float64 `json:"avg_score" example:"820.5"`
}

type StatsKeyword struct {
	Word      string  `json:"word" example:"35mm"`
	PostCount int     `json:"post_count" example:"3200"`
	AvgScore  float64 `json:"avg_score" example:"760.3"`
}

type StatsMeta struct {
	Total       int       `json:"total"`
	GeneratedAt time.Time `json:"generated_at"`
}

type StatsFilter struct {
	Limit       *int
	Start       *int
	End         *int
	Granularity *string
	Metric      *string
}

type StatsService interface {
	GetOverview(ctx context.Context, filter *StatsFilter) (*StatsOverview, error)
	GetPostsOverTime(ctx context.Context, filter *StatsFilter) ([]*StatsPeriod, error)
	GetFilmStats(ctx context.Context, filter *StatsFilter) ([]*StatsFilm, error)
	GetCameraStats(ctx context.Context, filter *StatsFilter) ([]*StatsCamera, error)
	GetColorStats(ctx context.Context, filter *StatsFilter) ([]*StatsColor, error)
	GetKeywordStats(ctx context.Context, filter *StatsFilter) ([]*StatsKeyword, error)
}
