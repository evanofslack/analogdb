package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/evanofslack/analogdb"
)

type Meta struct {
	TotalPosts int    `json:"total_posts"`
	PageSize   int    `json:"page_size"`
	PageID     string `json:"next_page_id"`
	PageURL    string `json:"next_page_url"`
	Seed       int    `json:"seed,omitempty"`
}

type Response struct {
	Meta  Meta            `json:"meta"`
	Posts []analogdb.Post `json:"posts"`
}

var defaultLimit = 20

func (s *Server) latestPosts(w http.ResponseWriter, r *http.Request) {
	filter, err := parseToFilter(r)
	if err != nil {
		writeError(w, r, err)
	}

	sort := "time"
	filter.Sort = &sort

	_, _, err = s.PostService.FindPosts(r.Context(), filter)
	if err != nil {
		writeError(w, r, err)
	}
	resp := &Response{}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(resp); err != nil {
		writeError(w, r, err)
	}

}

func (s *Server) createResponse(r *http.Request, filter *analogdb.PostFilter) (Response, error) {
	posts, _, err := s.PostService.FindPosts(r.Context(), filter)
	if err != nil {
		return Response{}, err
	}

	resp := Response{}
	for _, p := range posts {
		resp.Posts = append(resp.Posts, *p)
	}

	return resp, nil

}

func setMeta(filter *analogdb.PostFilter, posts []*analogdb.Post, count int) (Meta, error) {
	meta := Meta{TotalPosts: count}
	if limit := filter.Limit; limit != nil {
		meta.PageSize = *limit
	}
	//pageid
	if sort := filter.Sort; sort != nil {
		if *sort == "time" || *sort == "random" {
		}
	}
	//pageurl
	if seed := filter.Seed; seed != nil {
		meta.Seed = *seed
	}
	return meta, nil
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
