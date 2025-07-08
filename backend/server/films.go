package server

import (
	"encoding/json"
	"net/http"

	"github.com/evanofslack/analogdb"
	"github.com/go-chi/chi/v5"
)

type FilmsResponse struct {
	Films []analogdb.Film `json:"films"`
}

type CreateFilmResponse struct {
	Message string              `json:"message"`
	Film    analogdb.CreateFilm `json:"film"`
}

const (
	filmsPath = "/films"
)

func (s *Server) mountFilmHandlers() {
	s.router.Route(filmsPath, func(r chi.Router) {
		r.Get("/", s.getFilms)
		r.With(s.auth).Put("/", s.createFilm)
		r.With(s.auth).Post("/", s.createFilm)
	})
}

func (s *Server) getFilms(w http.ResponseWriter, r *http.Request) {
	filter, err := parseToFilmFilter(r)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	resp, err := s.makeFilmResponse(r, filter)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	err = encodeResponse(w, r, http.StatusOK, resp)
	if err != nil {
		s.writeError(w, r, err)
	}
}

func (s *Server) makeFilmResponse(r *http.Request, filter *analogdb.FilmFilter) (FilmsResponse, error) {
	films, err := s.FilmService.AllFilms(r.Context(), filter)
	resp := FilmsResponse{}
	if err != nil {
		return resp, err
	}
	for _, f := range films {
		resp.Films = append(resp.Films, *f)
	}
	return resp, nil
}

func (s *Server) createFilm(w http.ResponseWriter, r *http.Request) {
	var createFilm analogdb.CreateFilm
	if err := json.NewDecoder(r.Body).Decode(&createFilm); err != nil {
		err = &analogdb.Error{Code: analogdb.ERRUNPROCESSABLE, Message: "parse film from request body"}
		s.writeError(w, r, err)
		return
	}

	created, err := s.FilmService.CreateFilm(r.Context(), &createFilm)
	if err != nil || created == nil {
		s.writeError(w, r, err)
		return
	}

	createdResponse := CreateFilmResponse{
		Message: "Success, film created",
		Film:    *created,
	}
	if err := encodeResponse(w, r, http.StatusCreated, createdResponse); err != nil {
		s.writeError(w, r, err)
	}
}

// parse URL for query parameters and convert to FilmFilter
func parseToFilmFilter(r *http.Request) (*analogdb.FilmFilter, error) {
	filter := &analogdb.FilmFilter{}

	if includeCounts := r.URL.Query().Get("include_counts"); includeCounts != "" {
		if val, err := stringToBool(includeCounts); err != nil {
			return nil, err
		} else {
			filter.IncludeCounts = &val
		}
	}

	if excludeZero := r.URL.Query().Get("exclude_zero_counts"); excludeZero != "" {
		if val, err := stringToBool(excludeZero); err != nil {
			return nil, err
		} else {
			filter.IncludeCounts = &val
		}
	}

	return filter, nil
}
