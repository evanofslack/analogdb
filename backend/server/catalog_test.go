package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evanofslack/analogdb"
)

type mockFilmService struct {
	films []*analogdb.Film
}

func (m *mockFilmService) FindFilms(ctx context.Context, filter *analogdb.FilmFilter) ([]*analogdb.Film, error) {
	return m.films, nil
}

func (m *mockFilmService) CreateFilm(ctx context.Context, film *analogdb.CreateFilm) (*analogdb.CreateFilm, error) {
	return film, nil
}

type mockCameraService struct {
	cameras []*analogdb.Camera
}

func (m *mockCameraService) FindCameras(ctx context.Context, filter *analogdb.CameraFilter) ([]*analogdb.Camera, error) {
	return m.cameras, nil
}

func (m *mockCameraService) CreateCamera(ctx context.Context, camera *analogdb.CreateCamera) (*analogdb.CreateCamera, error) {
	return camera, nil
}

func TestMakeFilmResponseSlug(t *testing.T) {
	s := &Server{FilmService: &mockFilmService{films: []*analogdb.Film{
		{Make: "kodak", Type: "portra 400"},
		{Make: "rollei", Type: "rolleiflex 2.8f"},
	}}}
	r := httptest.NewRequest(http.MethodGet, "/films", nil)
	resp, err := s.makeFilmResponse(r, &analogdb.FilmFilter{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"kodak-portra-400", "rollei-rolleiflex-2-8f"}
	for i, f := range resp.Films {
		if f.Slug != want[i] {
			t.Errorf("want slug %q, got %q", want[i], f.Slug)
		}
	}
}

func TestMakeCameraResponseSlug(t *testing.T) {
	s := &Server{CameraService: &mockCameraService{cameras: []*analogdb.Camera{
		{Make: "voigtländer", Model: "bessa r2"},
		{Make: "hasselblad", Model: "500c/m"},
	}}}
	r := httptest.NewRequest(http.MethodGet, "/cameras", nil)
	resp, err := s.makeCameraResponse(r, &analogdb.CameraFilter{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"voigtlander-bessa-r2", "hasselblad-500c-m"}
	for i, c := range resp.Cameras {
		if c.Slug != want[i] {
			t.Errorf("want slug %q, got %q", want[i], c.Slug)
		}
	}
}
