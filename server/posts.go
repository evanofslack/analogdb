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
	if filter == nil {
		fmt.Println("nilllll")
	}
	if err != nil {
		fmt.Println(err)
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
		fmt.Print(*sort)
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
	falsey["false"] = true
	falsey["f"] = true
	falsey["no"] = true
	falsey["n"] = true
	falsey["0"] = true

	filter := &analogdb.PostFilter{Limit: &defaultLimit}

	if limit := r.URL.Query().Get("page_size"); limit != "" {
		fmt.Println(limit)
		if intLimit, err := strconv.Atoi(limit); err != nil {
			return nil, err
		} else {
			fmt.Println(intLimit)
			filter.Limit = &intLimit
			fmt.Println(*filter.Limit)
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
		if yes := truthy[nsfw]; yes {
			filter.Nsfw = &yes
		} else if no := falsey[strings.ToLower(nsfw)]; no {
			filter.Nsfw = &no
		} else {
			return nil, errors.New("invalid string to boolean conversion")
		}
	}
	if bw := r.URL.Query().Get("bw"); bw != "" {
		if yes := truthy[bw]; yes {
			filter.Grayscale = &yes
		} else if no := falsey[strings.ToLower(bw)]; no {
			filter.Grayscale = &no
		} else {
			return nil, errors.New("invalid string to boolean conversion")
		}
	}
	if sprock := r.URL.Query().Get("sprocket"); sprock != "" {
		if yes := truthy[sprock]; yes {
			filter.Sprocket = &yes
		} else if no := falsey[strings.ToLower(sprock)]; no {
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
