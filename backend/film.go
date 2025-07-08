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

// Film represents a film with its make, types and total count
type Film struct {
	Make      string     `json:"make"`
	Types     []FilmType `json:"film_types"`
	PostCount int        `json:"post_count"`
}

// FilmType represents a specific film type with post count
type FilmType struct {
	Id          int    `json:"id"`
	Make        string    `json:"make"`
	Type        string    `json:"type"`
	Speed       int       `json:"speed"`
	ColorType   string    `json:"color_type"`
	Description string    `json:"description"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
	PostCount   int       `json:"post_count"`
}

// FilmFilter are options used for querying films
type FilmFilter struct {
	IncludeCounts     *bool
	ExcludeZeroCounts *bool
}

func (filter *FilmFilter) String() string {
	out := []string{}
	if filter.IncludeCounts != nil {
		out = append(out, fmt.Sprintf("include_counts: %t", *filter.IncludeCounts))
	}
	if filter.ExcludeZeroCounts != nil {
		out = append(out, fmt.Sprintf("exclude_zero_counts: %t", *filter.ExcludeZeroCounts))
	}
	return strings.Join(out, ", ")
}

func NewFilmFilter(includeCounts *bool, excludeZeroCounts *bool) *FilmFilter {
	return &FilmFilter{
		IncludeCounts:     includeCounts,
		ExcludeZeroCounts: excludeZeroCounts,
	}
}

type FilmService interface {
	AllFilms(ctx context.Context, filter *FilmFilter) ([]*Film, error)
	CreateFilm(ctx context.Context, film *CreateFilm) (*CreateFilm, error)
}
