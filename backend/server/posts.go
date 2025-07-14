package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/go-chi/chi/v5"
)

type Meta struct {
	TotalPosts int    `json:"total_posts" example:"200"`
	PageSize   int    `json:"page_size" example:"20"`
	PageID     int `json:"next_page_id" example:"1752244116"`
	PageURL    string `json:"next_page_url" example:"/posts?sort=latest&page_size=20&page_id=1752244116"`
	Seed       int    `json:"seed,omitempty" example:"37"`
}

type PostResponse struct {
	Meta  Meta            `json:"meta"`
	Posts []analogdb.Post `json:"posts"`
}

type SimilarPostsResponse struct {
	Posts []analogdb.Post `json:"posts"`
}

type DeleteResponse struct {
	Message string `json:"message" example:"Success, post deleted"`
}

type PatchResponse struct {
	Message string `json:"message" example:"Success, post patched"`
}

type CreatePostResponse struct {
	Message string        `json:"message" example:"Success, post created"`
	Post    analogdb.Post `json:"post"`
}

type IDsResponse struct {
	Ids []int `json:"ids" example:"1,2,3,4,5"`
}

// default limit on number of posts returned
var defaultPostsLimit = 20

// max limit of posts returned
var maxPostsLimit = 200

// default limit on number of similar posts returned
var defaultSimilarityLimit = 12

// max limit of similar posts returned
var maxSimilarityLimit = 50

// default to sorting by time descending (latest)
var defaultPostsSort = analogdb.PostSortTime

const (
	postsPath = "/posts"
	postPath  = "/post"
	idsPath   = "/ids"
)

func (s *Server) mountPostHandlers(r chi.Router) {
	r.Route(postsPath, func(r chi.Router) {
		r.Get("/", s.getPosts)
	})
	r.Route(postPath, func(r chi.Router) {
		r.Get("/{id}", s.findPost)
		r.Get("/{id}/similar", s.getSimilarPosts)
		r.With(s.auth).Delete("/{id}", s.deletePost)
		r.With(s.auth).Patch("/{id}", s.patchPost)
		r.With(s.auth).Put("/", s.createPost)
		r.With(s.auth).Post("/", s.createPost)
	})
	r.Route(idsPath, func(r chi.Router) {
		r.Get("/", s.allPostIDs)
	})
}

// @Summary Get posts with optional filtering and pagination
// @Description Retrieve posts with optional query parameters for filtering by camera, film, keywords, etc. Supports pagination
// @Tags posts
// @Accept json
// @Produce json
// @Param page_size query int false "Number of posts per page" default(20)
// @Param page_id query int false "Page offset for pagination"
// @Param sort query string false "Sort order" Enums(time,score,random) default(time)
// @Param camera_make query string false "Filter by camera make"
// @Param camera_model query string false "Filter by camera model"
// @Param film_make query string false "Filter by film make"
// @Param film_type query string false "Filter by film type"
// @Param film_speed query int false "Filter by film speed"
// @Param focal_length query int false "Filter by focal length"
// @Param aperture query string false "Filter by aperture"
// @Param author query string false "Filter by author"
// @Param nsfw query bool false "Include NSFW posts"
// @Param grayscale query bool false "Filter by grayscale posts"
// @Param sprocket query bool false "Filter by sprocket posts"
// @Param keywords query string false "Filter by keywords (comma-separated)"
// @Param seed query int false "Random seed for consistent random sorting"
// @Success 200 {object} PostResponse
// @Failure 400 {object} analogdb.Error "Invalid request body"
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Router /posts [get]
func (s *Server) getPosts(w http.ResponseWriter, r *http.Request) {
	filter, err := parseToPostFilter(r)
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

// @Summary Find similar posts
// @Description Find posts similar to a given post using similarity matching
// @Tags posts
// @Accept json
// @Produce json
// @Param id query int true "Post ID to find similar posts for"
// @Param page_size query int false "Maximum number of similar posts to return" default(12)
// @Param nsfw query bool false "Include nsfw posts in query"
// @Param grayscale query bool false "Include b&w posts in query"
// @Param sprocket query bool false "Include sprocketshot posts in query"
// @Success 200 {object} SimilarPostsResponse
// @Failure 400 {object} analogdb.Error "Invalid request body"
// @Failure 404 {object} analogdb.Error "Not found"
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Router /posts/similar [get]
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

// @Summary Get a post
// @Description Get a post by ID from database
// @Tags post
// @Accept json
// @Produce json
// @Param id path int true "Post ID to get"
// @Success 200 {object} analogdb.Post
// @Failure 400 {object} analogdb.Error "Invalid request body"
// @Failure 404 {object} analogdb.Error "Not found"
// @Failure 401 {object} analogdb.Error "Unauthorized"
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Security BasicAuth
// @Router /post/{id} [get]
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

// @Summary Delete a post
// @Description Delete a post by ID from database (requires authentication)
// @Tags post
// @Accept json
// @Produce json
// @Param id path int true "Post ID to delete"
// @Success 200 {object} DeleteResponse
// @Failure 400 {object} analogdb.Error "Invalid request body"
// @Failure 404 {object} analogdb.Error "Not found"
// @Failure 401 {object} analogdb.Error "Unauthorized"
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Security BasicAuth
// @Router /post/{id} [delete]
func (s *Server) deletePost(w http.ResponseWriter, r *http.Request) {
	var err error

	var id string
	if id = chi.URLParam(r, "id"); id == "" {
		err = &analogdb.Error{Code: analogdb.ERRUNPROCESSABLE, Message: "must provide id as parameter"}
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

	success := DeleteResponse{Message: "Success, post deleted"}

	if err := encodeResponse(w, r, http.StatusOK, success); err != nil {
		s.writeError(w, r, err)
		return
	}
}

// @Summary Create a new post
// @Description Create a new post with image analysis and similarity encoding (requires authentication)
// @Tags post
// @Accept json
// @Produce json
// @Param post body analogdb.CreatePost true "Post data to create"
// @Success 201 {object} CreatePostResponse
// @Failure 400 {object} analogdb.Error "Invalid request body"
// @Failure 401 {object} analogdb.Error "Unauthorized"
// @Failure 422 {object} analogdb.Error "Unprocessable entity"
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Security BasicAuth
// @Router /post [post]
func (s *Server) createPost(w http.ResponseWriter, r *http.Request) {
	var createPost analogdb.CreatePost
	if err := json.NewDecoder(r.Body).Decode(&createPost); err != nil {
		err = &analogdb.Error{Code: analogdb.ERRUNPROCESSABLE, Message: "parse post from request body"}
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

	createdResponse := CreatePostResponse{
		Message: "Success, post created",
		Post:    *created,
	}
	if err := encodeResponse(w, r, http.StatusCreated, createdResponse); err != nil {
		s.writeError(w, r, err)
	}
}

// @Summary Update a post
// @Description Partially update a post's properties by ID (requires authentication)
// @Tags post
// @Accept json
// @Produce json
// @Param id path int true "Post ID to update"
// @Param post body analogdb.PatchPost true "Post fields to update"
// @Success 200 {object} PatchResponse
// @Failure 400 {object} analogdb.Error "Invalid request body"
// @Failure 401 {object} analogdb.Error "Unauthorized"
// @Failure 404 {object} analogdb.Error "Not found"
// @Failure 422 {object} analogdb.Error "Unprocessable entity"
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Security BasicAuth
// @Router /post/{id} [patch]
func (s *Server) patchPost(w http.ResponseWriter, r *http.Request) {
	var patchPost analogdb.PatchPost
	if err := json.NewDecoder(r.Body).Decode(&patchPost); err != nil {
		err = &analogdb.Error{Code: analogdb.ERRUNPROCESSABLE, Message: "parse patch from request body"}
		s.writeError(w, r, err)
		return
	}

	if id := chi.URLParam(r, "id"); id != "" {
		if identify, err := strconv.Atoi(id); err == nil {
			if err := s.PostService.PatchPost(r.Context(), &patchPost, identify); err == nil {
				success := PatchResponse{Message: "Success, post patched"}
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

// @Summary Get all post IDs
// @Description Retrieve a list of all post IDs in the system
// @Tags posts
// @Accept json
// @Produce json
// @Success 200 {object} IDsResponse
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Router /posts/ids [get]
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

	// add seed if sort order is random
	if sort := filter.Sort; *sort == analogdb.PostSortRandom {
		if seed := filter.Seed; seed != nil {
			meta.Seed = *seed
		}
	}

	// pageSize
	if limit := filter.Limit; limit != nil {
		meta.PageSize = *limit
		if len(posts) != *limit {
			// reached the end of pagination
			return meta, nil
		}
	}

	// pageID
	if sort := filter.Sort; sort != nil {
		sortVal := *sort
		if sortVal == analogdb.PostSortTime || sortVal == analogdb.PostSortRandom {
			meta.PageID = posts[len(posts)-1].Time
		} else if sortVal == analogdb.PostSortScore {
			meta.PageID = posts[len(posts)-1].Score
		} else {
			return Meta{}, fmt.Errorf("invalid sort parameter: %s", sortVal.String())
		}
	}

	// pageUrl
	if sort := filter.Sort; sort != nil {
		path := postsPath
		numParams := 0
		switch *sort {
		case analogdb.PostSortTime:
			path += fmt.Sprintf("%ssort=latest", paramJoiner(&numParams))
		case analogdb.PostSortScore:
			path += fmt.Sprintf("%ssort=top", paramJoiner(&numParams))
		case analogdb.PostSortRandom:
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
		if cm := filter.CameraMake; cm != nil {
			path += fmt.Sprintf("%scamera_make=%s", paramJoiner(&numParams), *cm)
		}
		if cm := filter.CameraModel; cm != nil {
			path += fmt.Sprintf("%scamera_model=%s", paramJoiner(&numParams), *cm)
		}
		if fm := filter.FilmMake; fm != nil {
			path += fmt.Sprintf("%sfilm_make=%s", paramJoiner(&numParams), *fm)
		}
		if ft := filter.FilmType; ft != nil {
			path += fmt.Sprintf("%sfilm_type=%s", paramJoiner(&numParams), *ft)
		}
		if fs := filter.FilmSpeed; fs != nil {
			path += fmt.Sprintf("%sfilm_speed=%d", paramJoiner(&numParams), *fs)
		}
		if fl := filter.FocalLength; fl != nil {
			path += fmt.Sprintf("%sfocal_length=%d", paramJoiner(&numParams), *fl)
		}
		if a := filter.Aperture; a != nil {
			path += fmt.Sprintf("%saperture=%s", paramJoiner(&numParams), *a)
		}
		if w := filter.Width; w != nil {
			if min := w.Min; min != nil {
				path += fmt.Sprintf("%swidth_min=%.2f", paramJoiner(&numParams), *min)
			}
			if max := w.Max; max != nil {
				path += fmt.Sprintf("%swidth_max=%.2f", paramJoiner(&numParams), *max)
			}
		}
		if h := filter.Height; h != nil {
			if min := h.Min; min != nil {
				path += fmt.Sprintf("%sheight_min=%.2f", paramJoiner(&numParams), *min)
			}
			if max := h.Max; max != nil {
				path += fmt.Sprintf("%sheight_max=%.2f", paramJoiner(&numParams), *max)
			}
		}
		if r := filter.AspectRatio; r != nil {
			if min := r.Min; min != nil {
				path += fmt.Sprintf("%sratio_min=%.2f", paramJoiner(&numParams), *min)
			}
			if max := r.Max; max != nil {
				path += fmt.Sprintf("%sratio_max=%.2f", paramJoiner(&numParams), *max)
			}
		}
		if colors := filter.Colors; colors != nil {
			for _, color := range *colors {
				path += fmt.Sprintf("%scolor=%s", paramJoiner(&numParams), color)
			}
		}
		if colorPercents := filter.ColorPercents; colorPercents != nil {
			for _, percent := range *colorPercents {
				path += fmt.Sprintf("%smin_color=%.2f", paramJoiner(&numParams), percent)
			}
		}
		if keywords := filter.Keywords; keywords != nil {
			for _, keyword := range *keywords {
				path += fmt.Sprintf("%skeyword=%s", paramJoiner(&numParams), keyword)
			}
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

func stringToBool(query string) (bool, error) {
	val, err := strconv.ParseBool(query)
	if err != nil {
		return false, fmt.Errorf("failed to parse %s to bool, err=%w", query, err)
	}
	return val, nil
}

func stringToInt(query string) (int, error) {
	val, err := strconv.Atoi(query)
	if err != nil {
		return 0, fmt.Errorf("failed to parse %s to integer, err=%w", query, err)
	}
	return val, nil
}

// parse URL for query parameters and convert to PostFilter needed to query db
func parseToPostFilter(r *http.Request) (*analogdb.PostFilter, error) {
	filter := analogdb.NewPostFilter(&defaultPostsLimit, &defaultPostsSort, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	values := r.URL.Query()

	if sort := values.Get("sort"); sort != "" {
		if sort == "latest" || sort == "top" || sort == "random" {
			switch sort {
			case "latest":
				time := analogdb.PostSortTime
				filter.Sort = &time
			case "top":
				top := analogdb.PostSortScore
				filter.Sort = &top
			case "random":
				random := analogdb.PostSortRandom
				filter.Sort = &random
			}
		} else {
			return nil, fmt.Errorf("invalid sort parameter %s, valid options are 'latest', 'top', 'random'", sort)
		}
	}

	if limit := values.Get("page_size"); limit != "" {
		if intLimit, err := stringToInt(limit); err != nil {
			return nil, err
		} else {
			// ensure limit is less than configured max
			if intLimit <= maxPostsLimit {
				filter.Limit = &intLimit
			} else {
				filter.Limit = &maxPostsLimit
			}
		}
	}

	if key := values.Get("page_id"); key != "" {
		if keyset, err := stringToInt(key); err != nil {
			err := fmt.Errorf("failed to parse %s to integer, err=%w", key, err)
			return nil, err
		} else {
			filter.Keyset = &keyset
		}
	}

	if nsfw := values.Get("nsfw"); nsfw != "" {
		if val, err := stringToBool(nsfw); err != nil {
			return nil, err
		} else {
			filter.Nsfw = &val
		}
	}

	if grayscale := values.Get("grayscale"); grayscale != "" {
		if val, err := stringToBool(grayscale); err != nil {
			return nil, err
		} else {
			filter.Grayscale = &val
		}
	}

	if sprock := values.Get("sprocket"); sprock != "" {
		if val, err := stringToBool(sprock); err != nil {
			return nil, err
		} else {
			filter.Sprocket = &val
		}
	}

	if seed := values.Get("seed"); seed != "" {
		if seed, err := stringToInt(seed); err != nil {
			return nil, err
		} else {
			filter.Seed = &seed
		}
	}

	if id := values.Get("id"); id != "" {
		if identify, err := strconv.Atoi(id); err != nil {
			return nil, err
		} else {
			filter.IDs = &[]int{identify}
		}
	}

	if title := values.Get("title"); title != "" {
		filter.Title = &title
	}

	if author := values.Get("author"); author != "" {
		filter.Author = &author
	}

	if start := values.Get("time_start"); start != "" {
		startInt, err := strconv.ParseInt(start, 10, 64)
		if err != nil {
			return nil, err
		}
		startTime := time.Unix(startInt, 0)
		filter.TimeStart = &startTime
	}

	if end := values.Get("time_end"); end != "" {
		endInt, err := strconv.ParseInt(end, 10, 64)
		if err != nil {
			return nil, err
		}
		endTime := time.Unix(endInt, 0)
		filter.TimeEnd = &endTime
	}

	if cm := values.Get("camera_make"); cm != "" {
		filter.CameraMake = &cm
	}

	if cm := values.Get("camera_model"); cm != "" {
		filter.CameraModel = &cm
	}

	if fm := values.Get("film_make"); fm != "" {
		filter.FilmMake = &fm
	}

	if ft := values.Get("film_type"); ft != "" {
		filter.FilmType = &ft
	}

	if fs := values.Get("film_speed"); fs != "" {
		if fsi, err := strconv.Atoi(fs); err != nil {
			return nil, err
		} else {
			filter.FilmSpeed = &fsi
		}
	}

	if fl := values.Get("focal_length"); fl != "" {
		if fli, err := strconv.Atoi(fl); err != nil {
			return nil, err
		} else {
			filter.FocalLength = &fli
		}
	}

	if a := values.Get("aperture"); a != "" {
		filter.Aperture = &a
	}

	if colorPercent, ok := values["min_color"]; ok {
		percents := []float64{}
		for _, p := range colorPercent {
			if percent, err := strconv.ParseFloat(p, 64); err != nil {
				err := fmt.Errorf("failed to parse %s to float, err=%w", colorPercent, err)
				return nil, err
			} else {
				percents = append(percents, percent)
			}
		}
		filter.ColorPercents = &percents
	}

	if colors, ok := values["color"]; ok {
		filter.Colors = &colors
		filter.SetMinColorPercent()
	}

	if keywords, ok := values["keyword"]; ok {
		filter.Keywords = &keywords
	}

	if minWidth := values.Get("width_min"); minWidth != "" {
		if width, err := strconv.ParseFloat(minWidth, 64); err != nil {
			err := fmt.Errorf("failed to parse %s to float, err=%w", minWidth, err)
			return nil, err
		} else {
			filter.Width.Min = &width
		}
	}

	if maxWidth := values.Get("width_max"); maxWidth != "" {
		if width, err := strconv.ParseFloat(maxWidth, 64); err != nil {
			err := fmt.Errorf("failed to parse %s to float, err=%w", maxWidth, err)
			return nil, err
		} else {
			filter.Width.Max = &width
		}
	}

	if minHeight := values.Get("height_min"); minHeight != "" {
		if height, err := strconv.ParseFloat(minHeight, 64); err != nil {
			err := fmt.Errorf("failed to parse %s to float, err=%w", minHeight, err)
			return nil, err
		} else {
			filter.Height.Min = &height
		}
	}

	if maxHeight := values.Get("height_max"); maxHeight != "" {
		if height, err := strconv.ParseFloat(maxHeight, 64); err != nil {
			err := fmt.Errorf("failed to parse %s to float, err=%w", maxHeight, err)
			return nil, err
		} else {
			filter.Height.Max = &height
		}
	}

	if minRatio := values.Get("ratio_min"); minRatio != "" {
		if ratio, err := strconv.ParseFloat(minRatio, 64); err != nil {
			err := fmt.Errorf("failed to parse %s to float, err=%w", minRatio, err)
			return nil, err
		} else {
			filter.AspectRatio.Min = &ratio
		}
	}

	if maxRatio := values.Get("ratio_max"); maxRatio != "" {
		if ratio, err := strconv.ParseFloat(maxRatio, 64); err != nil {
			err := fmt.Errorf("failed to parse %s to float, err=%w", maxRatio, err)
			return nil, err
		} else {
			filter.AspectRatio.Max = &ratio
		}
	}
	return filter, nil
}

// parse URL for query parameters and
// convert to PostSimilarityFilter (query vector db)
func parseToSimilarityFilter(r *http.Request) (*analogdb.PostSimilarityFilter, error) {
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
		if val, err := stringToBool(nsfw); err != nil {
			return nil, err
		} else {
			filter.Nsfw = &val
		}
	}

	if grayscale := r.URL.Query().Get("grayscale"); grayscale != "" {
		if val, err := stringToBool(grayscale); err != nil {
			return nil, err
		} else {
			filter.Grayscale = &val
		}
	}

	if sprock := r.URL.Query().Get("sprocket"); sprock != "" {
		if val, err := stringToBool(sprock); err != nil {
			return nil, err
		} else {
			filter.Sprocket = &val
		}
	}

	return filter, nil
}
