package analogdb

import "context"

type AdminCounts struct {
	Posts    int `json:"posts"`
	Authors  int `json:"authors"`
	Cameras  int `json:"cameras"`
	Films    int `json:"films"`
	Keywords int `json:"keywords"`
}

type AdminPostedCounts struct {
	Day   int `json:"day"`
	Week  int `json:"week"`
	Month int `json:"month"`
}

// AdminFreshness holds unix seconds of the latest pipeline activity, nil when never seen
type AdminFreshness struct {
	NewestPost     *int64 `json:"newest_post"`
	ScoreUpdate    *int64 `json:"score_update"`
	KeywordsUpdate *int64 `json:"keywords_update"`
	ColorsUpdate   *int64 `json:"colors_update"`
}

type AdminTableSize struct {
	Name  string `json:"name"`
	Bytes int64  `json:"bytes"`
}

type AdminDatabase struct {
	Bytes            int64            `json:"bytes"`
	Tables           []AdminTableSize `json:"tables"`
	MigrationVersion int              `json:"migration_version"`
	MigrationDirty   bool             `json:"migration_dirty"`
}

type AdminStats struct {
	Counts    AdminCounts       `json:"counts"`
	Posted    AdminPostedCounts `json:"posted"`
	Freshness AdminFreshness    `json:"freshness"`
	Database  AdminDatabase     `json:"database"`
}

type AdminCoverage struct {
	Total       int `json:"total"`
	Camera      int `json:"camera"`
	Film        int `json:"film"`
	Description int `json:"description"`
	Keywords    int `json:"keywords"`
	Colors      int `json:"colors"`
	FocalLength int `json:"focal_length"`
	Aperture    int `json:"aperture"`
}

type AdminUnmatchedCamera struct {
	CameraMake   string `json:"camera_make"`
	CameraModel  string `json:"camera_model"`
	PostCount    int    `json:"post_count"`
	SamplePostID int    `json:"sample_post_id"`
}

type AdminUnmatchedFilm struct {
	FilmMake     string `json:"film_make"`
	FilmType     string `json:"film_type"`
	FilmSpeed    *int64 `json:"film_speed"`
	PostCount    int    `json:"post_count"`
	SamplePostID int    `json:"sample_post_id"`
}

type AdminQuality struct {
	Coverage         AdminCoverage          `json:"coverage"`
	UnmatchedCameras []AdminUnmatchedCamera `json:"unmatched_cameras"`
	UnmatchedFilms   []AdminUnmatchedFilm   `json:"unmatched_films"`
}

type MissingField string

const (
	MissingCamera      MissingField = "camera"
	MissingFilm        MissingField = "film"
	MissingDescription MissingField = "description"
	MissingKeywords    MissingField = "keywords"
	MissingColors      MissingField = "colors"
)

func (f MissingField) Valid() bool {
	switch f {
	case MissingCamera, MissingFilm, MissingDescription, MissingKeywords, MissingColors:
		return true
	}
	return false
}

type MissingPostsFilter struct {
	Field    MissingField
	Limit    int
	BeforeID *int
}

type AdminPost struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Author      string  `json:"author"`
	Time        int64   `json:"time"`
	LowURL      string  `json:"low_url"`
	CameraMake  *string `json:"camera_make"`
	CameraModel *string `json:"camera_model"`
	FilmMake    *string `json:"film_make"`
	FilmType    *string `json:"film_type"`
	FilmSpeed   *int64  `json:"film_speed"`
}

type AdminService interface {
	Stats(ctx context.Context) (*AdminStats, error)
	Quality(ctx context.Context) (*AdminQuality, error)
	MissingPosts(ctx context.Context, filter *MissingPostsFilter) ([]*AdminPost, error)
}

// VectorCounter reports how many objects the vector database holds
type VectorCounter interface {
	CountObjects(ctx context.Context) (int, error)
}
