package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evanofslack/analogdb"
)

type mockFilmService struct {
	films     []*analogdb.Film
	deleted   []int
	deleteErr error
}

func (m *mockFilmService) FindFilms(ctx context.Context, filter *analogdb.FilmFilter) ([]*analogdb.Film, error) {
	return m.films, nil
}

func (m *mockFilmService) CreateFilm(ctx context.Context, film *analogdb.CreateFilm) (*analogdb.CreateFilm, error) {
	return film, nil
}

func (m *mockFilmService) DeleteFilm(ctx context.Context, id int) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deleted = append(m.deleted, id)
	return nil
}

type mockCameraService struct {
	cameras   []*analogdb.Camera
	deleted   []int
	deleteErr error
}

func (m *mockCameraService) FindCameras(ctx context.Context, filter *analogdb.CameraFilter) ([]*analogdb.Camera, error) {
	return m.cameras, nil
}

func (m *mockCameraService) CreateCamera(ctx context.Context, camera *analogdb.CreateCamera) (*analogdb.CreateCamera, error) {
	return camera, nil
}

func (m *mockCameraService) DeleteCamera(ctx context.Context, id int) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deleted = append(m.deleted, id)
	return nil
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

func deleteCatalog(s *Server, path string, authed bool) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodDelete, path, nil)
	if authed {
		r.SetBasicAuth("admin", "secret")
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)
	return w
}

func TestDeleteCatalogEntries(t *testing.T) {
	notFound := &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "camera not found"}
	inUse := &analogdb.Error{Code: analogdb.ERRCONFLICT, Message: "camera nikon f90s is used by 2 posts"}
	tests := []struct {
		name   string
		path   string
		authed bool
		err    error
		want   int
	}{
		{"camera deleted", "/v1/camera/7", true, nil, http.StatusOK},
		{"film deleted", "/v1/film/7", true, nil, http.StatusOK},
		{"camera missing", "/v1/camera/7", true, notFound, http.StatusNotFound},
		{"film in use", "/v1/film/7", true, inUse, http.StatusConflict},
		{"camera in use", "/v1/camera/7", true, inUse, http.StatusConflict},
		{"bad id", "/v1/camera/abc", true, nil, http.StatusBadRequest},
		{"camera needs auth", "/v1/camera/7", false, nil, http.StatusUnauthorized},
		{"film needs auth", "/v1/film/7", false, nil, http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := mustOpenAdmin(t)
			defer mustClose(t, s)
			cameras := &mockCameraService{deleteErr: tt.err}
			films := &mockFilmService{deleteErr: tt.err}
			s.CameraService = cameras
			s.FilmService = films

			w := deleteCatalog(s, tt.path, tt.authed)
			if w.Code != tt.want {
				t.Fatalf("want %d, got %d: %s", tt.want, w.Code, w.Body.String())
			}
			deleted := len(cameras.deleted) + len(films.deleted)
			if tt.want == http.StatusOK && deleted != 1 {
				t.Errorf("want one delete, got %d", deleted)
			}
			if tt.want != http.StatusOK && deleted != 0 {
				t.Errorf("want no delete, got %d", deleted)
			}
		})
	}
}
