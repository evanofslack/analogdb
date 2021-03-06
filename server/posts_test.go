package server

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evanofslack/analogdb"
	"github.com/go-chi/chi/v5"
)

type testInfo struct {
	name       string
	method     string
	target     string
	wantBody   any
	wantStatus int
}

func TestPosts(t *testing.T) {
	t1 := testInfo{
		name:   "latest",
		method: http.MethodGet,
		target: "/latest?page_size=20",
		wantBody: Response{
			Meta: Meta{
				TotalPosts: 51,
				PageSize:   20,
				PageID:     1646884084,
				PageURL:    "/latest?page_size=20&page_id=1646884084",
				Seed:       0,
			},
			Posts: []analogdb.Post{},
		},
		wantStatus: http.StatusOK,
	}
	t2 := testInfo{
		name:   "top",
		method: http.MethodGet,
		target: "/top?page_size=10",
		wantBody: Response{
			Meta: Meta{
				TotalPosts: 51,
				PageSize:   10,
				PageID:     730,
				PageURL:    "/top?page_size=10&page_id=730",
				Seed:       0,
			},
			Posts: []analogdb.Post{},
		},
		wantStatus: http.StatusOK,
	}
	t3 := testInfo{
		name:   "random",
		method: http.MethodGet,
		target: "/random?page_size=2",
		wantBody: Response{
			Meta: Meta{
				TotalPosts: 51,
				PageSize:   2,
				PageID:     0,
				PageURL:    "",
				Seed:       0,
			},
			Posts: []analogdb.Post{},
		},
		wantStatus: http.StatusOK,
	}
	t4 := testInfo{
		name:   "nsfw",
		method: http.MethodGet,
		target: "/latest?nsfw=true",
		wantBody: Response{
			Meta: Meta{
				TotalPosts: 4,
				PageSize:   20,
				PageID:     0,
				PageURL:    "",
				Seed:       0,
			},
			Posts: []analogdb.Post{},
		},
		wantStatus: http.StatusOK,
	}

	t5 := testInfo{
		name:   "inverse nsfw",
		method: http.MethodGet,
		target: "/latest?nsfw=false",
		wantBody: Response{
			Meta: Meta{
				TotalPosts: 47,
				PageSize:   20,
				PageID:     1646854637,
				PageURL:    "/latest?page_size=20&page_id=1646854637&nsfw=false",
				Seed:       0,
			},
			Posts: []analogdb.Post{},
		},
		wantStatus: http.StatusOK,
	}

	t6 := testInfo{
		name:   "title",
		method: http.MethodGet,
		target: "/latest?title=portra&page_size=10",
		wantBody: Response{
			Meta: Meta{
				TotalPosts: 17,
				PageSize:   10,
				PageID:     1646797974,
				PageURL:    "/latest?page_size=10&page_id=1646797974&title=portra",
				Seed:       0,
			},
			Posts: []analogdb.Post{},
		},
		wantStatus: http.StatusOK,
	}

	t7 := testInfo{
		name:   "title next page",
		method: http.MethodGet,
		target: "/latest?page_size=10&page_id=1646797974&title=portra",
		wantBody: Response{
			Meta: Meta{
				TotalPosts: 7,
				PageSize:   10,
				PageID:     0,
				PageURL:    "",
				Seed:       0,
			},
			Posts: []analogdb.Post{},
		},
		wantStatus: http.StatusOK,
	}
	tt := []testInfo{t1, t2, t3, t4, t5, t6, t7}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			s, db := mustOpen(t)
			defer mustClose(t, s, db)
			r := httptest.NewRequest(tc.method, tc.target, nil)
			w := httptest.NewRecorder()
			s.router.ServeHTTP(w, r)

			if want, got := tc.wantStatus, w.Code; got != want {
				t.Errorf("want status %d, gt %d", want, got)
			}

			res := w.Result()
			defer res.Body.Close()

			data, err := ioutil.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			var resp Response
			if err := json.Unmarshal(data, &resp); err != nil {
				t.Fatal(err)
			}

			if got, want := resp.Meta.TotalPosts, tc.wantBody.(Response).Meta.TotalPosts; got != want {
				t.Errorf("want %d, got %d", want, got)
			}

			if got, want := resp.Meta.PageSize, tc.wantBody.(Response).Meta.PageSize; got != want {
				t.Errorf("want %d, got %d", want, got)
			}

			if tc.name != "random" {
				if got, want := resp.Meta.PageID, tc.wantBody.(Response).Meta.PageID; got != want {
					t.Errorf("want %d, got %d", want, got)
				}

				if got, want := resp.Meta.PageURL, tc.wantBody.(Response).Meta.PageURL; got != want {
					t.Errorf("want %s, got %s", want, got)
				}
			}
		})
	}
}

func TestFindPost(t *testing.T) {
	t.Run("Existing Post", func(t *testing.T) {
		s, db := mustOpen(t)
		defer mustClose(t, s, db)

		r := httptest.NewRequest(http.MethodGet, "/post/2066", nil)
		w := httptest.NewRecorder()

		// chi URL params need to be added
		// https://github.com/go-chi/chi/issues/76#issuecomment-370145140
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "2066")
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		s.router.ServeHTTP(w, r)

		if want, got := http.StatusOK, w.Code; got != want {
			t.Errorf("want status %d, got %d", want, got)
		}

		res := w.Result()
		defer res.Body.Close()

		data, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}

		var post analogdb.Post
		if err := json.Unmarshal(data, &post); err != nil {
			t.Fatal(err)
		}

		if got, want := post.Id, 2066; got != want {
			t.Errorf("want %d, got %d", want, got)
		}
	})
	t.Run("Nonexisting Post", func(t *testing.T) {
		s, db := mustOpen(t)
		defer mustClose(t, s, db)

		r := httptest.NewRequest(http.MethodGet, "/post/69", nil)
		w := httptest.NewRecorder()

		// chi URL params need to be added
		// https://github.com/go-chi/chi/issues/76#issuecomment-370145140
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "69")
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		s.router.ServeHTTP(w, r)

		if want, got := http.StatusNotFound, w.Code; got != want {
			t.Errorf("want status %d, got %d", want, got)
		}
	})
}
