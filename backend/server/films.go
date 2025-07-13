package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

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

// default to sorting alphabetical
var defaultFilmsSort = analogdb.FilmSortAlphabetical

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
	films, err := s.FilmService.FindFilms(r.Context(), filter)
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
	filter := analogdb.NewFilmFilter(nil, &defaultFilmsSort, nil, nil, nil, nil, nil, nil, nil)

	values := r.URL.Query()

	if sort := values.Get("sort"); sort != "" {
		if sort == "alphabetical" || sort == "counts" {
			switch sort {
			case "alphabetical":
				alpha := analogdb.FilmSortAlphabetical
				filter.Sort = &alpha
			case "counts":
				counts := analogdb.FilmSortCounts
				filter.Sort = &counts
			}
		} else {
			return nil, fmt.Errorf("invalid sort parameter %s, valid options are 'alphabetical', or 'counts'", sort)
		}
	}

	if limit := values.Get("page_size"); limit != "" {
		if intLimit, err := stringToInt(limit); err != nil {
			return nil, err
		} else {
			filter.Limit = &intLimit
		}
	}

	if make := values.Get("make"); make != "" {
		filter.Make = &make
	}

	if ty := values.Get("type"); ty != "" {
		filter.Type = &ty
	}

	if speed := values.Get("speed"); speed != "" {
		if intSpeed, err := stringToInt(speed); err != nil {
			return nil, err
		} else {
			filter.Speed = &intSpeed
		}
	}

	if color := values.Get("colortype"); color != "" {
		filter.ColorType = &color
	}

	if id := values.Get("id"); id != "" {
		if identify, err := strconv.Atoi(id); err != nil {
			return nil, err
		} else {
			filter.IDs = &[]int{identify}
		}
	}

	if includeCounts := values.Get("include_counts"); includeCounts != "" {
		if val, err := stringToBool(includeCounts); err != nil {
			return nil, err
		} else {
			filter.IncludeCounts = &val
		}
	}

	if excludeZero := values.Get("exclude_zero_counts"); excludeZero != "" {
		if val, err := stringToBool(excludeZero); err != nil {
			return nil, err
		} else {
			filter.IncludeCounts = &val
		}
	}

	return filter, nil
}
