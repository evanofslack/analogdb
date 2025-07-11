package analogdb

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// CreateFilm is model for creating a film in database
type CreateFilm struct {
	Id          int       `json:"id"`
	Make        string    `json:"make"`
	Type        string    `json:"type"`
	Speed       int       `json:"speed"`
	ColorType   string    `json:"color_type"`
	Description string    `json:"description"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
}

// Film represents a specific film type with post count
type Film struct {
	Id          int       `json:"id"`
	Make        string    `json:"make"`
	Type        string    `json:"type"`
	Speed       int       `json:"speed"`
	ColorType   string    `json:"color_type"`
	Description string    `json:"description"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
	PostCount   int       `json:"post_count"`
}

type FilmSort int

const (
	FilmSortUnknown FilmSort = iota
	FilmSortAlphabetically
	FilmSortCounts
)

func (s FilmSort) String() string {
	switch s {
	case FilmSortAlphabetically:
		return "alphabetically"
	case FilmSortCounts:
		return "counts"
	default:
		return "unknown"
	}
}

func FilmSortFromString(s string) FilmSort {
	switch strings.ToLower(s) {
	case "alphabetically":
		return FilmSortAlphabetically
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

type FilmService interface {
	AllFilms(ctx context.Context, filter *FilmFilter) ([]*Film, error)
	CreateFilm(ctx context.Context, film *CreateFilm) (*CreateFilm, error)
}
