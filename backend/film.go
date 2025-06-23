package analogdb

import (
	"context"
	"time"
)

// Film represents the source info for a film
type Film struct {
	Id          int       `json:"id"`
	Make        string    `json:"make"`
	Type        string    `json:"type"`
	Speed       int       `json:"speed"`
	ColorType   string    `json:"color_type"`
	Description string    `json:"description"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
}

type FilmService interface {
	AllFilms(ctx context.Context) ([]*Film, error)
	CreateFilm(ctx context.Context, film *Film) (*Film, error)
}
