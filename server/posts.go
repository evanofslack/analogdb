package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/evanofslack/analogdb"
	"github.com/go-chi/chi/v5"
)

type Meta struct {
	TotalPosts int    `json:"total_posts"`
	PageSize   int    `json:"page_size"`
	PageID     int    `json:"next_page_id"`
	PageURL    string `json:"next_page_url"`
	Seed       int    `json:"seed,omitempty"`
}

type PostResponse struct {
	Meta  Meta            `json:"meta"`
	Posts []analogdb.Post `json:"posts"`
}

type DeleteResponse struct {
	Message string `json:"message"`
}

type CreateResponse struct {
	Message string        `json:"message"`
	Post    analogdb.Post `json:"post"`
}

var defaultLimit = 20

const (
	postsPath = "/posts"
	postPath  = "/post"
)

func (s *Server) mountPostHandlers() {
	s.router.Route(postsPath, func(r chi.Router) {
		r.Get("/latest", s.latestPosts)
		r.Get("/top", s.topPosts)
		r.Get("/random", s.randomPosts)
	})
	s.router.Route(postPath, func(r chi.Router) {
		r.Get("/{id}", s.findPost)
		r.With(auth).Delete("/{id}", s.deletePost)
		r.With(auth).Put("/", s.createPost)
		r.With(auth).Post("/", s.createPost)
	})
}

func (s *Server) latestPosts(w http.ResponseWriter, r *http.Request) {
	filter, err := parseToFilter(r)
	if err != nil {
		writeError(w, r, err)
	}
	sort := "time"
	filter.Sort = &sort

	resp, err := s.makePostResponse(r, filter)
	if err != nil {
		writeError(w, r, err)
	}
	err = encodeResponse(w, r, http.StatusOK, resp)
	if err != nil {
		writeError(w, r, err)
	}
}

func (s *Server) topPosts(w http.ResponseWriter, r *http.Request) {
	filter, err := parseToFilter(r)
	if err != nil {
		writeError(w, r, err)
	}
	sort := "score"
	filter.Sort = &sort
	resp, err := s.makePostResponse(r, filter)
	if err != nil {
		writeError(w, r, err)
	}
	err = encodeResponse(w, r, http.StatusOK, resp)
	if err != nil {
		writeError(w, r, err)
	}
}

func (s *Server) randomPosts(w http.ResponseWriter, r *http.Request) {
	filter, err := parseToFilter(r)
	if err != nil {
		writeError(w, r, err)
	}
	sort := "random"
	filter.Sort = &sort
	resp, err := s.makePostResponse(r, filter)
	if err != nil {
		writeError(w, r, err)
	}
	err = encodeResponse(w, r, http.StatusOK, resp)
	if err != nil {
		writeError(w, r, err)
	}
}

func (s *Server) findPost(w http.ResponseWriter, r *http.Request) {
	if id := chi.URLParam(r, "id"); id != "" {
		if identify, err := strconv.Atoi(id); err == nil {
			if post, err := s.PostService.FindPostByID(r.Context(), identify); err == nil {
				if err := encodeResponse(w, r, http.StatusOK, post); err != nil {
					writeError(w, r, err)
				}
			} else {
				writeError(w, r, err)
			}
		} else {
			writeError(w, r, err)
		}
	}
}

func (s *Server) deletePost(w http.ResponseWriter, r *http.Request) {
	if id := chi.URLParam(r, "id"); id != "" {
		if identify, err := strconv.Atoi(id); err == nil {
			if err := s.PostService.DeletePost(r.Context(), identify); err == nil {
				success := DeleteResponse{Message: "Success, post deleted"}
				if err := encodeResponse(w, r, http.StatusOK, success); err != nil {
					writeError(w, r, err)
				}
			} else {
				writeError(w, r, err)
			}
		} else {
			writeError(w, r, err)
		}
	}
}

func (s *Server) createPost(w http.ResponseWriter, r *http.Request) {
	var createPost analogdb.CreatePost
	if err := json.NewDecoder(r.Body).Decode(&createPost); err != nil {
		println("here: cant parse response body")
		err = &analogdb.Error{Code: analogdb.ERRUNPROCESSABLE, Message: "Error parsing post from request body"}
		writeError(w, r, err)
	}
	created, err := s.PostService.CreatePost(r.Context(), &createPost)
	if err != nil {
		println(err.Error())
		writeError(w, r, err)
	}
	createdResponse := CreateResponse{
		Message: "Success, post created",
		Post:    *created,
	}
	if err := encodeResponse(w, r, http.StatusCreated, createdResponse); err != nil {
		writeError(w, r, err)
	}
}

func encodeResponse(w http.ResponseWriter, r *http.Request, status int, v any) error {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return err
	}
	return nil
}

func (s *Server) makePostResponse(r *http.Request, filter *analogdb.PostFilter) (PostResponse, error) {
	posts, count, err := s.PostService.FindPosts(r.Context(), filter)
	if err != nil {
		return PostResponse{}, err
	}
	resp := PostResponse{}
	for _, p := range posts {
		resp.Posts = append(resp.Posts, *p)
	}
	resp.Meta, err = setMeta(filter, posts, count)
	if err != nil {
		return PostResponse{}, err
	}
	return resp, nil
}

// setMeta computes the metadata from a query
func setMeta(filter *analogdb.PostFilter, posts []*analogdb.Post, count int) (Meta, error) {
	meta := Meta{}
	// totalPosts
	meta.TotalPosts = count
	// seed
	if seed := filter.Seed; seed != nil {
		meta.Seed = *seed
	}
	// pageSize
	if limit := filter.Limit; limit != nil {
		meta.PageSize = *limit
		if len(posts) != *limit {
			// reached the end of pagination
			return meta, nil
		}
	}
	//pageID
	if sort := filter.Sort; sort != nil {
		if *sort == "time" || *sort == "random" {
			meta.PageID = posts[len(posts)-1].Time
		} else if *sort == "score" {
			meta.PageID = posts[len(posts)-1].Score
		} else {
			return Meta{}, errors.New("invalid sort parameter: " + *sort)
		}
	}
	//pageUrl
	if sort, path := filter.Sort, ""; sort != nil {
		switch *sort {
		case "time":
			path += postsPath + "/latest"
		case "score":
			path += postsPath + "/top"
		case "random":
			path += postsPath + "/random"
		}
		numParams := 0
		if limit := filter.Limit; limit != nil {
			path += fmt.Sprintf("%spage_size=%d", paramJoiner(&numParams), *limit)
		}
		path += fmt.Sprintf("%spage_id=%d", paramJoiner(&numParams), meta.PageID)
		if nsfw := filter.Nsfw; nsfw != nil {
			path += fmt.Sprintf("%snsfw=%t", paramJoiner(&numParams), *nsfw)
		}
		if bw := filter.Grayscale; bw != nil {
			path += fmt.Sprintf("%sbw=%t", paramJoiner(&numParams), *bw)
		}
		if sprock := filter.Sprocket; sprock != nil {
			path += fmt.Sprintf("%ssprocket=%t", paramJoiner(&numParams), *sprock)
		}
		if title := filter.Title; title != nil {
			path += fmt.Sprintf("%stitle=%s", paramJoiner(&numParams), *title)
		}
		if author := filter.Author; author != nil {
			path += fmt.Sprintf("%sauthor=%s", paramJoiner(&numParams), *author)
		}
		meta.PageURL = path
	}
	return meta, nil
}

func paramJoiner(numParams *int) string {
	if *numParams == 0 {
		*numParams += 1
		return "?"
	} else {
		*numParams += 1
		return "&"
	}
}

// parse URL for query parameters and convert to PostFilter needed to query db
func parseToFilter(r *http.Request) (*analogdb.PostFilter, error) {

	truthy := make(map[string]bool)
	truthy["true"] = true
	truthy["t"] = true
	truthy["yes"] = true
	truthy["y"] = true
	truthy["1"] = true

	falsey := make(map[string]bool)
	falsey["false"] = false
	falsey["f"] = false
	falsey["no"] = false
	falsey["n"] = false
	falsey["0"] = false

	filter := &analogdb.PostFilter{Limit: &defaultLimit}

	if limit := r.URL.Query().Get("page_size"); limit != "" {
		if intLimit, err := strconv.Atoi(limit); err != nil {
			return nil, err
		} else {
			filter.Limit = &intLimit
		}
	}
	if key := r.URL.Query().Get("page_id"); key != "" {
		if keyset, err := strconv.Atoi(key); err != nil {
			return nil, err
		} else {
			filter.Keyset = &keyset
		}
	}
	if nsfw := r.URL.Query().Get("nsfw"); nsfw != "" {
		if yes := truthy[strings.ToLower(nsfw)]; yes {
			filter.Nsfw = &yes
		} else if no := falsey[strings.ToLower(nsfw)]; !no {
			filter.Nsfw = &no
		} else {
			return nil, errors.New("invalid string to boolean conversion")
		}
	}
	if bw := r.URL.Query().Get("bw"); bw != "" {
		if yes := truthy[strings.ToLower(bw)]; yes {
			filter.Grayscale = &yes
		} else if no := falsey[strings.ToLower(bw)]; !no {
			filter.Grayscale = &no
		} else {
			return nil, errors.New("invalid string to boolean conversion")
		}
	}
	if sprock := r.URL.Query().Get("sprocket"); sprock != "" {
		if yes := truthy[strings.ToLower(sprock)]; yes {
			filter.Sprocket = &yes
		} else if no := falsey[strings.ToLower(sprock)]; !no {
			filter.Sprocket = &no
		} else {
			return nil, errors.New("invalid string to boolean conversion")
		}
	}
	if seed := r.URL.Query().Get("seed"); seed != "" {
		if seed, err := strconv.Atoi(seed); err != nil {
			return nil, err
		} else {
			filter.Seed = &seed
		}
	}
	if id := r.URL.Query().Get("id"); id != "" {
		if identify, err := strconv.Atoi(id); err != nil {
			return nil, err
		} else {
			filter.ID = &identify
		}
	}
	if title := r.URL.Query().Get("title"); title != "" {
		filter.Title = &title
	}
	if author := r.URL.Query().Get("author"); author != "" {
		filter.Author = &author
	}
	return filter, nil
}
