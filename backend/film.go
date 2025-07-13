package analogdb

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// CreateFilm is model for creating a film in database
type CreateFilm struct {
	Id          int       `json:"id" example:"1" swaggerignore:"true"`
	Make        string    `json:"make" example:"kodak"`
	Type        string    `json:"type" example:"portra 400"`
	Speed       int       `json:"speed" example:"400"`
	ColorType   string    `json:"color_type" example:"color"`
	Description string    `json:"description" example:"Kodak Portra 400 is professional color negative film with fine grain"`
	Created     time.Time `json:"created" example:"2025-07-11T12:00:00Z" swaggerignore:"true"`
	Updated     time.Time `json:"updated" example:"2025-07-11T12:00:00Z" swaggerignore:"true"`
}

// Film represents a specific film type with post count
type Film struct {
	Id          int       `json:"id" example:"1"`
	Make        string    `json:"make" example:"kodak"`
	Type        string    `json:"type" example:"portra 400"`
	Speed       int       `json:"speed" example:"400"`
	ColorType   string    `json:"color_type" exawple:"color"`
	Description string    `json:"description" example:"Kodak Portra 400 is professional color negative film with fine grain"`
	Created     time.Time `json:"created" example:"2025-07-11T12:00:00Z"`
	Updated     time.Time `json:"updated" example:"2025-07-11T12:00:00Z"`
	PostCount   int       `json:"post_count" example:"25"`
}

type FilmSort int

const (
	FilmSortUnknown FilmSort = iota
	FilmSortAlphabetical
	FilmSortCounts
)

func (s FilmSort) String() string {
	switch s {
	case FilmSortAlphabetical:
		return "alphabetical"
	case FilmSortCounts:
		return "counts"
	default:
		return "unknown"
	}
}

func FilmSortFromString(s string) FilmSort {
	switch strings.ToLower(s) {
	case "alphabetical":
		return FilmSortAlphabetical
	case "counts":
		return FilmSortCounts
	default:
		return FilmSortUnknown
	}
}

// FilmFilter are options used for querying films
type FilmFilter struct {
	Limit             *int
	Sort              *FilmSort
	IDs               *[]int
	Make              *string
	Type              *string
	Speed             *int
	ColorType         *string
	IncludeCounts     *bool
	ExcludeZeroCounts *bool
}

func NewFilmFilter(limit *int, sort *FilmSort, ids *[]int, make *string, ty *string, speed *int, colortype *string, includeCounts *bool, excludeZeroCounts *bool) *FilmFilter {
	return &FilmFilter{
		Limit:             limit,
		Sort:              sort,
		IDs:               ids,
		Make:              make,
		Type:              ty,
		Speed:             speed,
		ColorType:         colortype,
		IncludeCounts:     includeCounts,
		ExcludeZeroCounts: excludeZeroCounts,
	}
}

func (filter *FilmFilter) String() string {
	out := []string{}
	if limit := filter.Limit; limit != nil {
		out = append(out, fmt.Sprintf("limit: %d", *limit))
	}
	if sort := filter.Sort; sort != nil {
		out = append(out, fmt.Sprintf("sort: %q", *sort))
	}
	if ids := filter.IDs; ids != nil {
		out = append(out, fmt.Sprintf("ids: %v", *ids))
	}
	if make := filter.Make; make != nil {
		out = append(out, fmt.Sprintf("make: %s", *make))
	}
	if ty := filter.Type; ty != nil {
		out = append(out, fmt.Sprintf("type: %s", *ty))
	}
	if speed := filter.Speed; speed != nil {
		out = append(out, fmt.Sprintf("speed: %d", *speed))
	}
	if color := filter.ColorType; color != nil {
		out = append(out, fmt.Sprintf("color_type: %s", *color))
	}
	if includeCounts := filter.IncludeCounts; includeCounts != nil {
		out = append(out, fmt.Sprintf("include_counts: %t", *includeCounts))
	}
	if excludeZeros := filter.ExcludeZeroCounts; excludeZeros != nil {
		out = append(out, fmt.Sprintf("exclude_zero_counts: %t", *excludeZeros))
	}
	return strings.Join(out, ", ")
}

type FilmService interface {
	FindFilms(ctx context.Context, filter *FilmFilter) ([]*Film, error)
	CreateFilm(ctx context.Context, film *CreateFilm) (*CreateFilm, error)
}
