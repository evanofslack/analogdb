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
	Message string        `json:"message"`
	Film    analogdb.Film `json:"film"`
}

const (
	filmsPath = "/films"
)

func (s *Server) mountFilmHandlers() {
	s.router.Route(camerasPath, func(r chi.Router) {
		r.Get("/", s.getFilms)
	})
	s.router.Route(postPath, func(r chi.Router) {
		r.With(s.auth).Put("/", s.createFilm)
		r.With(s.auth).Post("/", s.createFilm)
	})
	s.router.Route(idsPath, func(r chi.Router) {
		r.Get("/", s.allPostIDs)
	})
}

func (s *Server) getFilms(w http.ResponseWriter, r *http.Request) {
	resp, err := s.makeFilmResponse(r)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	err = encodeResponse(w, r, http.StatusOK, resp)
	if err != nil {
		s.writeError(w, r, err)
	}
}

func (s *Server) makeFilmResponse(r *http.Request) (FilmsResponse, error) {
	films, err := s.FilmService.AllFilms(r.Context())
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
	var createFilm analogdb.Film
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
