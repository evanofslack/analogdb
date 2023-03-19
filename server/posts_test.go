package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/evanofslack/analogdb"
	"github.com/go-chi/chi/v5"
)

const (
	// number of posts matching each query from test DB
	totalPosts  = 4864
	totalNsfw   = 276
	totalPortra = 1463
)

type testInfo struct {
	name       string
	method     string
	target     string
	wantBody   any
	wantStatus int
}

func TestGetPosts(t *testing.T) {
	t1 := testInfo{
		name:   "latest",
		method: http.MethodGet,
		target: "/posts?page_size=20",
		wantBody: PostResponse{
			Meta: Meta{
				TotalPosts: totalPosts,
				PageSize:   20,
				PageID:     1679063590,
				PageURL:    "/posts?sort=latest&page_size=20&page_id=1679063590",
				Seed:       0,
			},
			Posts: []analogdb.Post{},
		},
		wantStatus: http.StatusOK,
	}
	t2 := testInfo{
		name:   "top",
		method: http.MethodGet,
		target: "/posts?sort=top&page_size=10",
		wantBody: PostResponse{
			Meta: Meta{
				TotalPosts: totalPosts,
				PageSize:   10,
				PageID:     5493,
				PageURL:    "/posts?sort=top&page_size=10&page_id=5493",
				Seed:       0,
			},
			Posts: []analogdb.Post{},
		},
		wantStatus: http.StatusOK,
	}
	t3 := testInfo{
		name:   "random",
		method: http.MethodGet,
		target: "/posts?sort=random&page_size=10",
		wantBody: PostResponse{
			Meta: Meta{
				TotalPosts: totalPosts,
				PageSize:   10,
				PageID:     1675964298,
				PageURL:    "/posts?sort=random&page_size=10&page_id=1675964298",
				Seed:       0,
			},
			Posts: []analogdb.Post{},
		},
		wantStatus: http.StatusOK,
	}
	t4 := testInfo{
		name:   "nsfw",
		method: http.MethodGet,
		target: "/posts?nsfw=true",
		wantBody: PostResponse{
			Meta: Meta{
				TotalPosts: totalNsfw,
				PageSize:   20,
				PageID:     1676314409,
				PageURL:    "/posts?sort=latest&page_size=20&page_id=1676314409&nsfw=true",
				Seed:       0,
			},
			Posts: []analogdb.Post{},
		},
		wantStatus: http.StatusOK,
	}

	t5 := testInfo{
		name:   "inverse nsfw",
		method: http.MethodGet,
		target: "/posts?nsfw=false",
		wantBody: PostResponse{
			Meta: Meta{
				TotalPosts: totalPosts - totalNsfw,
				PageSize:   20,
				PageID:     1679063590,
				PageURL:    "/posts?sort=latest&page_size=20&page_id=1679063590&nsfw=false",
				Seed:       0,
			},
			Posts: []analogdb.Post{},
		},
		wantStatus: http.StatusOK,
	}

	t6 := testInfo{
		name:   "title",
		method: http.MethodGet,
		target: "/posts?title=portra&page_size=10",
		wantBody: PostResponse{
			Meta: Meta{
				TotalPosts: totalPortra,
				PageSize:   10,
				PageID:     1679038927,
				PageURL:    "/posts?sort=latest&page_size=10&page_id=1679038927&title=portra",
				Seed:       0,
			},
			Posts: []analogdb.Post{},
		},
		wantStatus: http.StatusOK,
	}

	t7 := testInfo{
		name:   "title next page",
		method: http.MethodGet,
		target: "/posts?sort=latest&page_size=10&page_id=1679038927&title=portra",

		wantBody: PostResponse{
			Meta: Meta{
				TotalPosts: totalPortra - 10,
				PageSize:   10,
				PageID:     1678898231,
				PageURL:    "/posts?sort=latest&page_size=10&page_id=1678898231&title=portra",
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
			var resp PostResponse
			if err := json.Unmarshal(data, &resp); err != nil {
				t.Fatal(err)
			}

			if got, want := resp.Meta.TotalPosts, tc.wantBody.(PostResponse).Meta.TotalPosts; got != want {
				t.Errorf("want %d, got %d", want, got)
			}

			if got, want := resp.Meta.PageSize, tc.wantBody.(PostResponse).Meta.PageSize; got != want {
				t.Errorf("want %d, got %d", want, got)
			}

			if tc.name != "random" {
				if got, want := resp.Meta.PageID, tc.wantBody.(PostResponse).Meta.PageID; got != want {
					t.Errorf("want %d, got %d", want, got)
				}

				if got, want := resp.Meta.PageURL, tc.wantBody.(PostResponse).Meta.PageURL; got != want {
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

func TestCreateAndDeletePost(t *testing.T) {
	t.Run("Valid put request", func(t *testing.T) {
		s, db := mustOpen(t)
		defer mustClose(t, s, db)

		createPost := makeTestCreatePost(true)
		jsonCreatePost, _ := json.Marshal(createPost)

		r := httptest.NewRequest(http.MethodPut, "/post", bytes.NewBuffer(jsonCreatePost))

		r.Header.Set("Authorization", makeAuthHeader())

		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, r)

		if want, got := http.StatusCreated, w.Code; got != want {
			t.Errorf("want status %d, got %d", want, got)
		}

		res := w.Result()
		defer res.Body.Close()

		data, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}

		var createResponse CreateResponse
		if err := json.Unmarshal(data, &createResponse); err != nil {
			t.Fatal(err)
		}

		id := createResponse.Post.Id

		// delete the created post
		r = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/post/%d", createResponse.Post.Id), nil)
		w = httptest.NewRecorder()

		// chi URL params need to be added
		// https://github.com/go-chi/chi/issues/76#issuecomment-370145140
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", fmt.Sprintf("%d", id))
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		r.Header.Set("Authorization", makeAuthHeader())
		s.router.ServeHTTP(w, r)

		if want, got := http.StatusOK, w.Code; got != want {
			t.Errorf("want status %d, got %d", want, got)
		}
	})
	t.Run("Valid post request", func(t *testing.T) {
		s, db := mustOpen(t)
		defer mustClose(t, s, db)

		createPost := makeTestCreatePost(true)
		jsonCreatePost, _ := json.Marshal(createPost)

		r := httptest.NewRequest(http.MethodPost, "/post", bytes.NewBuffer(jsonCreatePost))

		r.Header.Set("Authorization", makeAuthHeader())

		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, r)

		if want, got := http.StatusCreated, w.Code; got != want {
			t.Errorf("want status %d, got %d", want, got)
		}

		res := w.Result()
		defer res.Body.Close()

		data, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}

		var createResponse CreateResponse
		if err := json.Unmarshal(data, &createResponse); err != nil {
			t.Fatal(err)
		}

		id := createResponse.Post.Id

		// delete the created post
		r = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/post/%d", createResponse.Post.Id), nil)
		w = httptest.NewRecorder()

		// chi URL params need to be added
		// https://github.com/go-chi/chi/issues/76#issuecomment-370145140
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", fmt.Sprintf("%d", id))
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		r.Header.Set("Authorization", makeAuthHeader())
		s.router.ServeHTTP(w, r)

		if want, got := http.StatusOK, w.Code; got != want {
			t.Errorf("want status %d, got %d", want, got)
		}
	})
	t.Run("Invalid put request", func(t *testing.T) {
		s, db := mustOpen(t)
		defer mustClose(t, s, db)

		createPost := makeTestCreatePost(false)
		jsonCreatePost, _ := json.Marshal(createPost)

		r := httptest.NewRequest(http.MethodPut, "/post", bytes.NewBuffer(jsonCreatePost))

		r.Header.Set("Authorization", makeAuthHeader())

		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, r)

		if want, got := http.StatusUnprocessableEntity, w.Code; got != want {
			t.Errorf("want status %d, got %d", want, got)
		}
	})
	t.Run("Invalid post request", func(t *testing.T) {
		s, db := mustOpen(t)
		defer mustClose(t, s, db)

		createPost := makeTestCreatePost(false)
		jsonCreatePost, _ := json.Marshal(createPost)

		r := httptest.NewRequest(http.MethodPost, "/post", bytes.NewBuffer(jsonCreatePost))

		r.Header.Set("Authorization", makeAuthHeader())

		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, r)

		if want, got := http.StatusUnprocessableEntity, w.Code; got != want {
			t.Errorf("want status %d, got %d", want, got)
		}
	})
}

func TestPatchPost(t *testing.T) {
	t.Run("Valid Patch", func(t *testing.T) {
		s, db := mustOpen(t)
		defer mustClose(t, s, db)

		// first we get an existing post so that we can modify it
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

		var ogPost analogdb.Post
		if err := json.Unmarshal(data, &ogPost); err != nil {
			t.Fatal(err)
		}

		if got, want := ogPost.Id, 2066; got != want {
			t.Errorf("want %d, got %d", want, got)
		}

		// then we modify that post by patching it
		newScore := ogPost.Score + 1

		patchPost := analogdb.PatchPost{
			Score: &newScore,
		}

		jsonPatchPost, _ := json.Marshal(patchPost)

		r = httptest.NewRequest(http.MethodPatch, "/post/2066", bytes.NewBuffer(jsonPatchPost))
		w = httptest.NewRecorder()

		// chi URL params need to be added
		// https://github.com/go-chi/chi/issues/76#issuecomment-370145140
		rctx = chi.NewRouteContext()
		rctx.URLParams.Add("id", "2066")
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		r.Header.Set("Authorization", makeAuthHeader())
		s.router.ServeHTTP(w, r)

		// then we need to again get the post and confirm its been modified
		r = httptest.NewRequest(http.MethodGet, "/post/2066", nil)
		w = httptest.NewRecorder()

		// chi URL params need to be added
		// https://github.com/go-chi/chi/issues/76#issuecomment-370145140
		rctx = chi.NewRouteContext()
		rctx.URLParams.Add("id", "2066")
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		s.router.ServeHTTP(w, r)

		if want, got := http.StatusOK, w.Code; got != want {
			t.Errorf("want status %d, got %d", want, got)
		}

		res = w.Result()
		defer res.Body.Close()

		data, err = io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}

		var modifiedPost analogdb.Post
		if err := json.Unmarshal(data, &modifiedPost); err != nil {
			t.Fatal(err)
		}

		if og, mod := ogPost.Score, modifiedPost.Score; og == mod {
			t.Errorf("Updated post should have different score than original post, og %d, new %d", og, mod)
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

func TestAllPostIDs(t *testing.T) {
	t.Run("valid IDs", func(t *testing.T) {
		s, db := mustOpen(t)
		defer mustClose(t, s, db)

		r := httptest.NewRequest(http.MethodGet, "/ids", nil)
		w := httptest.NewRecorder()

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

		type IDsResponse struct {
			Ids []int `json:"ids"`
		}
		var ids IDsResponse

		if err := json.Unmarshal(data, &ids); err != nil {
			t.Fatal(err)
		}

		if got, want := len(ids.Ids), totalPosts; got != want {
			t.Errorf("invalid number of post IDs, want %d, got %d", want, got)
		}
	})
}

func makeTestCreatePost(valid bool) analogdb.CreatePost {
	testImage := analogdb.Image{
		Label:  "test",
		Url:    "test.com",
		Width:  0,
		Height: 0,
	}
	var testImages []analogdb.Image
	if valid {
		// valid post has 4 images
		testImages = append(testImages, testImage, testImage, testImage, testImage)
	} else {
		// invalid post has anything other than 4 images
		testImages = append(testImages, testImage, testImage)
	}

	testTitle := "test title"

	createPost := analogdb.CreatePost{
		Images:    testImages,
		Title:     testTitle,
		Author:    "test author",
		Permalink: "test.permalink.com",
		Score:     0,
		Nsfw:      false,
		Grayscale: false,
		Time:      0,
		Sprocket:  false,
	}

	return createPost
}

func makeAuthHeader() string {
	username := os.Getenv("AUTH_USERNAME")
	password := os.Getenv("AUTH_PASSWORD")
	auth := fmt.Sprintf("%s:%s", username, password)
	enc_auth := base64.StdEncoding.EncodeToString([]byte(auth))
	header := fmt.Sprintf("Basic %s", enc_auth)

	return header
}
