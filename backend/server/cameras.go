package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/evanofslack/analogdb"
	"github.com/go-chi/chi/v5"
)

type CamerasResponse struct {
	Cameras []analogdb.Camera `json:"cameras"`
}

type CreateCameraResponse struct {
	Message string                `json:"message" example:"Success, camera created"`
	Camera  analogdb.CreateCamera `json:"camera"`
}

// default to sorting alphabetical
var defaultCamerasSort = analogdb.CameraSortAlphabetical

const (
	camerasPath = "/cameras"
	cameraPath  = "/camera"
)

func (s *Server) mountCameraHandlers(r chi.Router) {
	r.Route(camerasPath, func(r chi.Router) {
		r.Get("/", s.getCameras)
	})
	r.Route(cameraPath, func(r chi.Router) {
		r.With(s.auth).Put("/", s.createCamera)
		r.With(s.auth).Post("/", s.createCamera)
	})
}

// @Summary Get cameras with optional filtering
// @Description Retrieve cameras with optional query parameters for filtering and sorting
// @Tags cameras
// @Accept json
// @Produce json
// @Param sort query string false "Sort order" Enums(alphabetical, counts)
// @Param page_size query int false "Number of results to return"
// @Param make query string false "Filter by camera make"
// @Param model query string false "Filter by camera model"
// @Param id query int false "Filter by specific camera ID"
// @Param include_counts query bool false "Include count data"
// @Param exclude_zero_counts query bool false "Exclude zero counts"
// @Success 200 {object} CamerasResponse
// @Failure 400 {object} analogdb.Error "Invalid query parameters"
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Router /cameras [get]
func (s *Server) getCameras(w http.ResponseWriter, r *http.Request) {
	filter, err := parseToCameraFilter(r)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	resp, err := s.makeCameraResponse(r, filter)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	err = encodeResponse(w, r, http.StatusOK, resp)
	if err != nil {
		s.writeError(w, r, err)
	}
}

func (s *Server) makeCameraResponse(r *http.Request, filter *analogdb.CameraFilter) (CamerasResponse, error) {
	cameras, err := s.CameraService.FindCameras(r.Context(), filter)
	resp := CamerasResponse{}
	if err != nil {
		return resp, err
	}
	for _, c := range cameras {
		resp.Cameras = append(resp.Cameras, *c)
	}
	return resp, nil
}

// @Summary Create a new camera
// @Description Create a new camera entry (requires authentication)
// @Tags camera
// @Accept json
// @Produce json
// @Param camera body analogdb.CreateCamera true "camera data to create"
// @Success 201 {object} CreateCameraResponse
// @Failure 400 {object} analogdb.Error "Invalid request body"
// @Failure 401 {object} analogdb.Error "Unauthorized"
// @Failure 422 {object} analogdb.Error "Unprocessable entity"
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Security BasicAuth
// @Router /camera [put]
// @Router /camera [post]
func (s *Server) createCamera(w http.ResponseWriter, r *http.Request) {
	var createCamera analogdb.CreateCamera
	if err := json.NewDecoder(r.Body).Decode(&createCamera); err != nil {
		err = &analogdb.Error{Code: analogdb.ERRUNPROCESSABLE, Message: "parse camera from request body"}
		s.writeError(w, r, err)
		return
	}

	created, err := s.CameraService.CreateCamera(r.Context(), &createCamera)
	if err != nil || created == nil {
		s.writeError(w, r, err)
		return
	}

	createdResponse := CreateCameraResponse{
		Message: "Success, camera created",
		Camera:  *created,
	}
	if err := encodeResponse(w, r, http.StatusCreated, createdResponse); err != nil {
		s.writeError(w, r, err)
	}
}

// parse URL for query parameters and convert to FilmFilter
func parseToCameraFilter(r *http.Request) (*analogdb.CameraFilter, error) {
	filter := analogdb.NewCameraFilter(nil, &defaultCamerasSort, nil, nil, nil, nil, nil, nil, nil)

	values := r.URL.Query()

	if sort := values.Get("sort"); sort != "" {
		if sort == "alphabetical" || sort == "counts" {
			switch sort {
			case "alphabetical":
				alpha := analogdb.CameraSortAlphabetical
				filter.Sort = &alpha
			case "counts":
				counts := analogdb.CameraSortCounts
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

	if model := values.Get("model"); model != "" {
		filter.Model = &model
	}

	if id := values.Get("id"); id != "" {
		if identify, err := strconv.Atoi(id); err != nil {
			return nil, err
		} else {
			filter.IDs = &[]int{identify}
		}
	}

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
