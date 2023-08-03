package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
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

type SimilarPostsResponse struct {
	Posts []analogdb.Post `json:"posts"`
}

type DeleteResponse struct {
	Message string `json:"message"`
}

type CreateResponse struct {
	Message string        `json:"message"`
	Post    analogdb.Post `json:"post"`
}

type IDsResponse struct {
	Ids []int `json:"ids"`
}

// default limit on number of posts returned
var defaultLimit = 20

// max limit of posts returned
var maxLimit = 200

// default limit on number of similar posts returned
var defaultSimilarityLimit = 12

// max limit of similar posts returned
var maxSimilarityLimit = 50

// default to sorting by time descending (latest)
var defaultSort = "time"

// seeds for random post order
var primes = []int{11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67, 71, 73, 79, 83, 89, 97, 101, 107, 113, 131, 137, 149, 167, 173, 179, 191, 197, 227, 233, 239, 251, 257, 263}

const (
	postsPath = "/posts"
	postPath  = "/post"
	idsPath   = "/ids"
)

func (s *Server) mountPostHandlers() {
	s.router.Route(postsPath, func(r chi.Router) {
		r.Get("/", s.getPosts)
	})
	s.router.Route(postPath, func(r chi.Router) {
		r.Get("/{id}", s.findPost)
		r.Get("/{id}/similar", s.getSimilarPosts)
		r.With(s.auth).Delete("/{id}", s.deletePost)
		r.With(s.auth).Patch("/{id}", s.patchPost)
		r.With(s.auth).Put("/", s.createPost)
		r.With(s.auth).Post("/", s.createPost)
	})
	s.router.Route(idsPath, func(r chi.Router) {
		r.Get("/", s.allPostIDs)
	})
}

func (s *Server) getPosts(w http.ResponseWriter, r *http.Request) {
	filter, err := parseToFilter(r)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	resp, err := s.makePostResponse(r, filter)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	err = encodeResponse(w, r, http.StatusOK, resp)
	if err != nil {
		s.writeError(w, r, err)
	}
}

func (s *Server) getSimilarPosts(w http.ResponseWriter, r *http.Request) {

	resp := SimilarPostsResponse{}

	similarityFilter, err := parseToSimilarityFilter(r)
	if err != nil {
		s.writeError(w, r, err)
		return
	}

	if posts, err := s.SimilarityService.FindSimilarPosts(r.Context(), similarityFilter); err == nil {
		for _, p := range posts {
			resp.Posts = append(resp.Posts, *p)
		}
		if err := encodeResponse(w, r, http.StatusOK, resp); err != nil {
			s.writeError(w, r, err)
		}
	} else {
		s.writeError(w, r, err)
	}
}

func (s *Server) findPost(w http.ResponseWriter, r *http.Request) {
	if id := chi.URLParam(r, "id"); id != "" {
		if identify, err := strconv.Atoi(id); err == nil {
			if post, err := s.PostService.FindPostByID(r.Context(), identify); err == nil {
				if err := encodeResponse(w, r, http.StatusOK, post); err != nil {
					s.writeError(w, r, err)
				}
			} else {
				s.writeError(w, r, err)
			}
		} else {
			s.writeError(w, r, err)
		}
	}
}

func (s *Server) deletePost(w http.ResponseWriter, r *http.Request) {

	var err error

	var id string
	if id = chi.URLParam(r, "id"); id == "" {
		err = &analogdb.Error{Code: analogdb.ERRUNPROCESSABLE, Message: "Must provide id as parameter"}
		s.writeError(w, r, err)
		return
	}

	var identify int
	if identify, err = strconv.Atoi(id); err != nil {
		s.writeError(w, r, err)
		return
	}

	if err := s.PostService.DeletePost(r.Context(), identify); err != nil {
		s.writeError(w, r, err)
		return
	}

	if err := s.SimilarityService.DeletePost(r.Context(), identify); err != nil {
		s.writeError(w, r, err)
		return
	}

	success := DeleteResponse{Message: "success, post deleted"}

	if err := encodeResponse(w, r, http.StatusOK, success); err != nil {
		s.writeError(w, r, err)
		return
	}
}

func (s *Server) createPost(w http.ResponseWriter, r *http.Request) {
	var createPost analogdb.CreatePost
	if err := json.NewDecoder(r.Body).Decode(&createPost); err != nil {
		err = &analogdb.Error{Code: analogdb.ERRUNPROCESSABLE, Message: "error parsing post from request body"}
		s.writeError(w, r, err)
		return
	}

	// create the post in db
	created, err := s.PostService.CreatePost(r.Context(), &createPost)
	if err != nil || created == nil {
		s.writeError(w, r, err)
		return
	}

	// check if encoding is disabled
	encode := r.Context().Value(analogdb.EncodeContextKey)
	doEncode, _ := encode.(bool)

	// if there is no context value or context value is true, do encode
	if encode == nil || doEncode {
		toEncode := []int{created.Id}
		err = s.SimilarityService.BatchEncodePosts(r.Context(), toEncode, 1)
		if err != nil {
			s.writeError(w, r, err)
			return
		}
	}

	createdResponse := CreateResponse{
		Message: "Success, post created",
		Post:    *created,
	}
	if err := encodeResponse(w, r, http.StatusCreated, createdResponse); err != nil {
		s.writeError(w, r, err)
	}
}

func (s *Server) patchPost(w http.ResponseWriter, r *http.Request) {

	var patchPost analogdb.PatchPost
	if err := json.NewDecoder(r.Body).Decode(&patchPost); err != nil {
		err = &analogdb.Error{Code: analogdb.ERRUNPROCESSABLE, Message: "error parsing patch from request body"}
		s.writeError(w, r, err)
		return
	}

	if id := chi.URLParam(r, "id"); id != "" {
		if identify, err := strconv.Atoi(id); err == nil {
			if err := s.PostService.PatchPost(r.Context(), &patchPost, identify); err == nil {
				success := DeleteResponse{Message: "success, post patched"}
				if err := encodeResponse(w, r, http.StatusOK, success); err != nil {
					s.writeError(w, r, err)
				}
			} else {
				s.writeError(w, r, err)
			}
		} else {
			s.writeError(w, r, err)
		}
	}
}

func (s *Server) allPostIDs(w http.ResponseWriter, r *http.Request) {
	ids, err := s.PostService.AllPostIDs(r.Context())
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	idsResponse := IDsResponse{
		Ids: ids,
	}
	if err := encodeResponse(w, r, http.StatusOK, idsResponse); err != nil {
		s.writeError(w, r, err)
	}
}

func (s *Server) makePostResponse(r *http.Request, filter *analogdb.PostFilter) (PostResponse, error) {
	posts, count, err := s.PostService.FindPosts(r.Context(), filter)
	resp := PostResponse{}
	if err != nil {
		return resp, err
	}
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
	if sort := filter.Sort; sort != nil {
		path := postsPath
		numParams := 0
		switch *sort {
		case "time":
			path += fmt.Sprintf("%ssort=latest", paramJoiner(&numParams))
		case "score":
			path += fmt.Sprintf("%ssort=top", paramJoiner(&numParams))
		case "random":
			path += fmt.Sprintf("%ssort=random", paramJoiner(&numParams))
		}
		if limit := filter.Limit; limit != nil {
			path += fmt.Sprintf("%spage_size=%d", paramJoiner(&numParams), *limit)
		}
		path += fmt.Sprintf("%spage_id=%d", paramJoiner(&numParams), meta.PageID)
		if nsfw := filter.Nsfw; nsfw != nil {
			path += fmt.Sprintf("%snsfw=%t", paramJoiner(&numParams), *nsfw)
		}
		if grayscale := filter.Grayscale; grayscale != nil {
			path += fmt.Sprintf("%sgrayscale=%t", paramJoiner(&numParams), *grayscale)
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

func newSeed() int {
	randomIndex := rand.Intn(len(primes))
	return primes[randomIndex]
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

	filter := &analogdb.PostFilter{Limit: &defaultLimit, Sort: &defaultSort}

	if sort := r.URL.Query().Get("sort"); sort != "" {
		if sort == "latest" || sort == "top" || sort == "random" {
			switch sort {
			case "latest":
				time := "time"
				filter.Sort = &time
			case "top":
				score := "score"
				filter.Sort = &score
			case "random":
				random := "random"
				seed := newSeed()
				filter.Sort = &random
				filter.Seed = &seed
			}
		} else {
			return nil, errors.New("invalid sort parameter - valid options: latest, top, random")
		}
	}

	if limit := r.URL.Query().Get("page_size"); limit != "" {
		if intLimit, err := strconv.Atoi(limit); err != nil {
			return nil, err
		} else {
			// ensure limit is less than configured max
			if intLimit <= maxLimit {
				filter.Limit = &intLimit
			} else {
				filter.Limit = &maxLimit
			}
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
	if grayscale := r.URL.Query().Get("grayscale"); grayscale != "" {
		if yes := truthy[strings.ToLower(grayscale)]; yes {
			filter.Grayscale = &yes
		} else if no := falsey[strings.ToLower(grayscale)]; !no {
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
			filter.IDs = &[]int{identify}
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

// parse URL for query parameters and
// convert to PostSimilarityFilter (query vector db)
func parseToSimilarityFilter(r *http.Request) (*analogdb.PostSimilarityFilter, error) {

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

	filter := &analogdb.PostSimilarityFilter{Limit: &defaultSimilarityLimit}

	// there must be a post id
	id := chi.URLParam(r, "id")
	if id == "" {
		return nil, errors.New("must include post id to query similar from")
	}
	postID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("post id to query similar from must convert to int, error=%w", err)
	}
	filter.ID = &postID

	// if we are getting similar to that post, we don't want to match the same post
	excluded := []int{postID}
	filter.ExcludeIDs = &excluded

	if limit := r.URL.Query().Get("page_size"); limit != "" {
		if intLimit, err := strconv.Atoi(limit); err != nil {
			return nil, err
		} else {
			// ensure limit is less than configured max
			if intLimit <= maxSimilarityLimit {
				filter.Limit = &intLimit
			} else {
				filter.Limit = &maxSimilarityLimit
			}
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
	if grayscale := r.URL.Query().Get("grayscale"); grayscale != "" {
		if yes := truthy[strings.ToLower(grayscale)]; yes {
			filter.Grayscale = &yes
		} else if no := falsey[strings.ToLower(grayscale)]; !no {
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
	return filter, nil
}
