package server

import (
	"encoding/json"
	"net/http"

	"github.com/evanofslack/analogdb"
	"github.com/go-chi/chi/v5"
)

type CamerasResponse struct {
	Cameras []analogdb.Camera `json:"cameras"`
}

type CreateCameraResponse struct {
	Message string          `json:"message"`
	Camera  analogdb.Camera `json:"camera"`
}

const (
	camerasPath = "/cameras"
)

func (s *Server) mountCameraHandlers() {
	s.router.Route(camerasPath, func(r chi.Router) {
		r.Get("/", s.getCameras)
		r.With(s.auth).Put("/", s.createCamera)
		r.With(s.auth).Post("/", s.createCamera)
	})
}

func (s *Server) getCameras(w http.ResponseWriter, r *http.Request) {
	resp, err := s.makeCameraResponse(r)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	err = encodeResponse(w, r, http.StatusOK, resp)
	if err != nil {
		s.writeError(w, r, err)
	}
}

func (s *Server) makeCameraResponse(r *http.Request) (CamerasResponse, error) {
	cameras, err := s.CameraService.AllCameras(r.Context())
	resp := CamerasResponse{}
	if err != nil {
		return resp, err
	}
	for _, c := range cameras {
		resp.Cameras = append(resp.Cameras, *c)
	}
	return resp, nil
}

func (s *Server) createCamera(w http.ResponseWriter, r *http.Request) {
	var createCamera analogdb.Camera
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
