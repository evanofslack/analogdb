package http

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

type Response struct {
	Meta  Meta            `json:"meta"`
	Posts []analogdb.Post `json:"posts"`
}

var defaultLimit = 20

const (
	latestPath = "/latest"
	topPath    = "/top"
	randomPath = "/random"
	findPath   = "/posts/{id}"
)

func (s *Server) MountPostHandlers() {
	s.router.Group(func(r chi.Router) {
		r.Get(latestPath, s.latestPosts)
		r.Get(topPath, s.topPosts)
		r.Get(randomPath, s.randomPosts)
	})
	s.router.Route(findPath, func(r chi.Router) {
		r.Get("/", s.findPost)
		r.With(auth).Delete("/", s.deletePost)
	})
}

func (s *Server) latestPosts(w http.ResponseWriter, r *http.Request) {
	filter, err := parseToFilter(r)
	if err != nil {
		writeError(w, r, err)
	}
	sort := "time"
	filter.Sort = &sort

	resp, err := s.createResponse(r, filter)
	if err != nil {
		writeError(w, r, err)
	}
	err = encodeResponse(w, r, resp)
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
	resp, err := s.createResponse(r, filter)
	if err != nil {
		writeError(w, r, err)
	}
	err = encodeResponse(w, r, resp)
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
	resp, err := s.createResponse(r, filter)
	if err != nil {
		writeError(w, r, err)
	}
	err = encodeResponse(w, r, resp)
	if err != nil {
		writeError(w, r, err)
	}
}

func (s *Server) findPost(w http.ResponseWriter, r *http.Request) {
	if id := r.URL.Query().Get("id"); id != "" {
		if identify, err := strconv.Atoi(id); err != nil {
			if post, err := s.PostService.FindPostByID(r.Context(), identify); err != nil {
				if err := encodeResponse(w, r, post); err != nil {
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
	if id := r.URL.Query().Get("id"); id != "" {
		if identify, err := strconv.Atoi(id); err != nil {
			if post, err := s.PostService.DeletePost(r.Context(), identify); err != nil {
				if err := encodeResponse(w, r, post); err != nil {
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

func encodeResponse(w http.ResponseWriter, r *http.Request, v any) error {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return err
	}
	return nil
}

func (s *Server) createResponse(r *http.Request, filter *analogdb.PostFilter) (Response, error) {
	posts, count, err := s.PostService.FindPosts(r.Context(), filter)
	if err != nil {
		return Response{}, err
	}
	resp := Response{}
	for _, p := range posts {
		resp.Posts = append(resp.Posts, *p)
	}
	resp.Meta, err = setMeta(filter, posts, count)
	if err != nil {
		return Response{}, err
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
			path += latestPath
		case "score":
			path += topPath
		case "random":
			path += randomPath
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
	falsey["false"] = true
	falsey["f"] = true
	falsey["no"] = true
	falsey["n"] = true
	falsey["0"] = true

	filter := &analogdb.PostFilter{Limit: &defaultLimit}

	if li := r.URL.Query().Get("page_size"); li != "" {
		if limit, err := strconv.Atoi(li); err != nil {
			filter.Limit = &limit
		} else {
			return nil, err
		}
	}
	if ke := r.URL.Query().Get("page_id"); ke != "" {
		if keyset, err := strconv.Atoi(ke); err != nil {
			filter.Keyset = &keyset
		} else {
			return nil, err
		}
	}
	if ns := r.URL.Query().Get("nsfw"); ns != "" {
		if yes := truthy[ns]; yes {
			filter.Nsfw = &yes
		} else if no := falsey[strings.ToLower(ns)]; no {
			filter.Nsfw = &no
		} else {
			return nil, errors.New("invalid string to boolean conversion")
		}
	}
	if gr := r.URL.Query().Get("bw"); gr != "" {
		if yes := truthy[gr]; yes {
			filter.Grayscale = &yes
		} else if no := falsey[strings.ToLower(gr)]; no {
			filter.Grayscale = &no
		} else {
			return nil, errors.New("invalid string to boolean conversion")
		}
	}
	if sp := r.URL.Query().Get("sprocket"); sp != "" {
		if yes := truthy[sp]; yes {
			filter.Sprocket = &yes
		} else if no := falsey[strings.ToLower(sp)]; no {
			filter.Sprocket = &no
		} else {
			return nil, errors.New("invalid string to boolean conversion")
		}
	}
	if se := r.URL.Query().Get("seed"); se != "" {
		if seed, err := strconv.Atoi(se); err != nil {
			filter.Seed = &seed
		} else {
			return nil, err
		}
	}
	if id := r.URL.Query().Get("id"); id != "" {
		if identify, err := strconv.Atoi(id); err != nil {
			filter.ID = &identify
		} else {
			return nil, err
		}
	}
	if ti := r.URL.Query().Get("title"); ti != "" {
		filter.Title = &ti
	}
	if au := r.URL.Query().Get("author"); au != "" {
		filter.Author = &au
	}

	return filter, nil
}
