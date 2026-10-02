package server

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/go-chi/chi/v5"
)

type Meta struct {
	TotalPosts int `json:"total_posts" example:"200"`
	PageSize   int `json:"page_size" example:"20"`
	// Opaque cursor for the next page, empty at the end
	NextCursor string `json:"next_cursor" example:"eyJzIjoidGltZSIsInYiOjE3NTIyNDQxMTYsImlkIjo0MDIxMX0"`
	// Deprecated: use next_cursor
	PageID  int    `json:"next_page_id" example:"1752244116"`
	PageURL string `json:"next_page_url" example:"/posts?cursor=eyJzIjoidGltZSIsInYiOjE3NTIyNDQxMTYsImlkIjo0MDIxMX0&page_size=20&sort=time"`
	Seed    int    `json:"seed,omitempty" example:"37"`
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

// random sort picks a seed from 1..randomSeedPool when none is given
const randomSeedPool = 500

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
// @Param cursor query string false "Opaque cursor from next_cursor for the next page"
// @Param page_id query int false "Deprecated: use cursor. Keyset from next_page_id, not supported with sort=random"
// @Param sort query string false "Sort order" Enums(time,score,random) default(time)
// @Param seed query int false "Random seed for consistent random sorting, picked from 1..500 when missing"
// @Param id query int false "Filter by post ID"
// @Param title query string false "Filter by post title"
// @Param author query string false "Filter by author"
// @Param time_start query int false "Filter by start time (unix timestamp)"
// @Param time_end query int false "Filter by end time (unix timestamp)"
// @Param camera_make query string false "Filter by camera make"
// @Param camera_model query string false "Filter by camera model"
// @Param film_make query string false "Filter by film make"
// @Param film_type query string false "Filter by film type"
// @Param film_speed query int false "Filter by film speed"
// @Param focal_length query int false "Filter by focal length"
// @Param aperture query string false "Filter by aperture"
// @Param nsfw query bool false "Filter by NSFW (true=only, false=exclude)"
// @Param grayscale query bool false "Filter by black and white (true=only, false=exclude)"
// @Param sprocket query bool false "Filter by sprocketshots (true=only, false=exclude)"
// @Param keyword query []string false "Filter by keywords (multiple allowed)" collectionFormat(multi)
// @Param color query []string false "Color filter (multiple allowed)" collectionFormat(multi)
// @Param min_color query []number false "Minimum color percentage (multiple allowed)" collectionFormat(multi)
// @Param width_min query number false "Minimum picture width"
// @Param width_max query number false "Maximum picture width"
// @Param height_min query number false "Minimum picture height"
// @Param height_max query number false "Maximum picture height"
// @Param ratio_min query number false "Minimum picture aspect ratio"
// @Param ratio_max query number false "Maximum picture aspect ratio"
// @Success 200 {object} PostResponse
// @Failure 400 {object} analogdb.Error "Invalid request body"
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Router /posts [get]
func (s *Server) getPosts(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("page_id") != "" {
		w.Header().Set("Deprecation", "true")
		w.Header().Set("Sunset", sunsetDate)
		s.stats.deprecatedParams.WithLabelValues("page_id", legacyClient(r.UserAgent())).Inc()
	}
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
// @Tags post
// @Accept json
// @Produce json
// @Param id path int true "Post ID to find similar posts for"
// @Param page_size query int false "Maximum number of similar posts to return" default(12)
// @Param nsfw query bool false "Include nsfw posts in query"
// @Param grayscale query bool false "Include b&w posts in query"
// @Param sprocket query bool false "Include sprocketshot posts in query"
// @Success 200 {object} SimilarPostsResponse
// @Failure 400 {object} analogdb.Error "Invalid request body"
// @Failure 404 {object} analogdb.Error "Not found"
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Router /post/{id}/similar [get]
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
		if identify, err := stringToInt(id); err == nil {
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
	if identify, err = stringToInt(id); err != nil {
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
// @Failure 409 {object} analogdb.Error "Post with permalink already exists"
// @Failure 422 {object} analogdb.Error "Unprocessable entity"
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Security BasicAuth
// @Router /post [post]
func (s *Server) createPost(w http.ResponseWriter, r *http.Request) {
	var createPost analogdb.CreatePost
	if err := s.decodeBody(w, r, &createPost, "parse post from request body"); err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := validateCaption(createPost.Caption); err != nil {
		s.writeError(w, r, err)
		return
	}

	// create the post in db
	created, err := s.PostService.CreatePost(r.Context(), &createPost)
	if err != nil || created == nil {
		s.writeError(w, r, err)
		return
	}

	s.encodePost(r, created.Id, "Fail encode created post")

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
	if err := s.decodeBody(w, r, &patchPost, "parse patch from request body"); err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := patchPost.ValidateClear(); err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := validateCaption(patchPost.Caption); err != nil {
		s.writeError(w, r, err)
		return
	}

	if id := chi.URLParam(r, "id"); id != "" {
		if identify, err := stringToInt(id); err == nil {
			if err := s.PostService.PatchPost(r.Context(), &patchPost, identify); err == nil {
				if patchChangesVector(&patchPost) {
					s.encodePost(r, identify, "Fail encode patched post")
				}
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

// patchChangesVector reports whether a patch touches fields stored in the vector DB
func patchChangesVector(patch *analogdb.PatchPost) bool {
	return patch.Nsfw != nil || patch.Grayscale != nil || patch.Sprocket != nil ||
		patch.Keywords != nil || patch.Caption != nil
}

// encodePost writes the post to the vector DB, logging failures without failing the request
func (s *Server) encodePost(r *http.Request, id int, failMsg string) {
	// skip when the context disables encoding
	encode := r.Context().Value(analogdb.EncodeContextKey)
	if doEncode, _ := encode.(bool); encode != nil && !doEncode {
		return
	}
	failedIDs, err := s.SimilarityService.BatchEncodePosts(r.Context(), []int{id}, 1)
	if err == nil && len(failedIDs) != 0 {
		err = fmt.Errorf("failed to encode post ids %v", failedIDs)
	}
	if err != nil {
		s.logger.ErrorContext(r.Context(), failMsg, "error", err, "post_id", id)
		s.stats.postEncodeFailures.Inc()
	}
}

func validateCaption(caption *analogdb.PostCaption) error {
	if caption == nil {
		return nil
	}
	if caption.Model == "" || caption.Version == "" {
		return badRequest("caption: model and version are required")
	}
	if !jsonStartsWith(caption.Raw, '{') {
		return badRequest("caption: raw must be a JSON object")
	}
	return nil
}

// @Summary Get all post IDs
// @Description Retrieve a list of all post IDs in the system
// @Tags posts
// @Accept json
// @Produce json
// @Success 200 {object} IDsResponse
// @Failure 500 {object} analogdb.Error "Internal server error"
// @Router /ids [get]
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
	// fetch one extra post to learn if there is a next page
	query := *filter
	pageSize := 0
	if filter.Limit != nil {
		pageSize = *filter.Limit
		limit := pageSize + 1
		query.Limit = &limit
	}
	posts, count, err := s.PostService.FindPosts(r.Context(), &query)
	resp := PostResponse{}
	if err != nil {
		return resp, err
	}
	hasMore := pageSize > 0 && len(posts) > pageSize
	if hasMore {
		posts = posts[:pageSize]
	}
	for _, p := range posts {
		resp.Posts = append(resp.Posts, *p)
	}
	resp.Meta, err = setMeta(filter, posts, count, hasMore)
	if err != nil {
		return PostResponse{}, err
	}
	return resp, nil
}

// setMeta computes the metadata from a query
func setMeta(filter *analogdb.PostFilter, posts []*analogdb.Post, count int, hasMore bool) (Meta, error) {
	meta := Meta{}

	// totalPosts
	meta.TotalPosts = count

	sort := defaultPostsSort
	if filter.Sort != nil {
		sort = *filter.Sort
	}

	// add seed if sort order is random
	seed := 0
	if sort == analogdb.PostSortRandom && filter.Seed != nil {
		seed = *filter.Seed
		meta.Seed = seed
	}

	// pageSize
	if limit := filter.Limit; limit != nil {
		meta.PageSize = *limit
	}

	if !hasMore || len(posts) == 0 {
		// reached the end of pagination
		return meta, nil
	}
	last := posts[len(posts)-1]

	// pageID, kept non-zero while there are more pages
	switch sort {
	case analogdb.PostSortTime, analogdb.PostSortRandom:
		meta.PageID = last.Time
	case analogdb.PostSortScore:
		meta.PageID = last.Score
	default:
		return Meta{}, fmt.Errorf("invalid sort parameter: %s", sort.String())
	}
	if meta.PageID == 0 {
		meta.PageID = 1
	}

	cursor, err := encodeCursor(sort, last, seed)
	if err != nil {
		return Meta{}, err
	}
	meta.NextCursor = cursor

	values := filterToValues(filter, sort)
	values.Set("cursor", cursor)
	meta.PageURL = postsPath + "?" + values.Encode()
	return meta, nil
}

// filterToValues turns a filter back into query parameters
func filterToValues(filter *analogdb.PostFilter, sort analogdb.PostSort) url.Values {
	values := url.Values{}
	values.Set("sort", sort.String())
	if limit := filter.Limit; limit != nil {
		values.Set("page_size", strconv.Itoa(*limit))
	}
	if seed := filter.Seed; seed != nil && sort == analogdb.PostSortRandom {
		values.Set("seed", strconv.Itoa(*seed))
	}
	if nsfw := filter.Nsfw; nsfw != nil {
		values.Set("nsfw", strconv.FormatBool(*nsfw))
	}
	if grayscale := filter.Grayscale; grayscale != nil {
		values.Set("grayscale", strconv.FormatBool(*grayscale))
	}
	if sprock := filter.Sprocket; sprock != nil {
		values.Set("sprocket", strconv.FormatBool(*sprock))
	}
	if ids := filter.IDs; ids != nil {
		for _, id := range *ids {
			values.Add("id", strconv.Itoa(id))
		}
	}
	if title := filter.Title; title != nil {
		values.Set("title", *title)
	}
	if author := filter.Author; author != nil {
		values.Set("author", *author)
	}
	if start := filter.TimeStart; start != nil {
		values.Set("time_start", strconv.FormatInt(start.Unix(), 10))
	}
	if end := filter.TimeEnd; end != nil {
		values.Set("time_end", strconv.FormatInt(end.Unix(), 10))
	}
	if cm := filter.CameraMake; cm != nil {
		values.Set("camera_make", *cm)
	}
	if cm := filter.CameraModel; cm != nil {
		values.Set("camera_model", *cm)
	}
	if fm := filter.FilmMake; fm != nil {
		values.Set("film_make", *fm)
	}
	if ft := filter.FilmType; ft != nil {
		values.Set("film_type", *ft)
	}
	if fs := filter.FilmSpeed; fs != nil {
		values.Set("film_speed", strconv.Itoa(*fs))
	}
	if fl := filter.FocalLength; fl != nil {
		values.Set("focal_length", strconv.Itoa(*fl))
	}
	if a := filter.Aperture; a != nil {
		values.Set("aperture", *a)
	}
	setDimension := func(dim *analogdb.Dimension, minKey, maxKey string) {
		if dim == nil {
			return
		}
		if min := dim.Min; min != nil {
			values.Set(minKey, formatFloat(*min))
		}
		if max := dim.Max; max != nil {
			values.Set(maxKey, formatFloat(*max))
		}
	}
	setDimension(filter.Width, "width_min", "width_max")
	setDimension(filter.Height, "height_min", "height_max")
	setDimension(filter.AspectRatio, "ratio_min", "ratio_max")
	if colors := filter.Colors; colors != nil {
		for _, color := range *colors {
			values.Add("color", color)
		}
	}
	if colorPercents := filter.ColorPercents; colorPercents != nil {
		for _, percent := range *colorPercents {
			values.Add("min_color", formatFloat(percent))
		}
	}
	if keywords := filter.Keywords; keywords != nil {
		for _, keyword := range *keywords {
			values.Add("keyword", keyword)
		}
	}
	return values
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func stringToBool(query string) (bool, error) {
	val, err := strconv.ParseBool(query)
	if err != nil {
		return false, badRequest("failed to parse %s to bool", query)
	}
	return val, nil
}

func stringToInt(query string) (int, error) {
	val, err := strconv.Atoi(query)
	if err != nil {
		return 0, badRequest("failed to parse %s to integer", query)
	}
	return val, nil
}

func stringToInt64(query string) (int64, error) {
	val, err := strconv.ParseInt(query, 10, 64)
	if err != nil {
		return 0, badRequest("failed to parse %s to integer", query)
	}
	return val, nil
}

func stringToFloat(query string) (float64, error) {
	val, err := strconv.ParseFloat(query, 64)
	if err != nil {
		return 0, badRequest("failed to parse %s to float", query)
	}
	return val, nil
}

// clampLimit bounds a limit to max, using fallback when below min
func clampLimit(limit, fallback, min, max int) int {
	if limit < min {
		return fallback
	}
	if limit > max {
		return max
	}
	return limit
}

// parse URL for query parameters and convert to PostFilter needed to query db
func parseToPostFilter(r *http.Request) (*analogdb.PostFilter, error) {
	filter := analogdb.NewPostFilter(&defaultPostsLimit, &defaultPostsSort, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	sortTime := analogdb.PostSortTime
	sortScore := analogdb.PostSortScore
	sortRandom := analogdb.PostSortRandom
	values := r.URL.Query()

	if sort := values.Get("sort"); sort != "" {
		switch sort {
		case sortTime.String():
			filter.Sort = &sortTime
		case sortScore.String():
			filter.Sort = &sortScore
		case sortRandom.String():
			filter.Sort = &sortRandom
		default:
			return nil, badRequest("invalid sort parameter %s, valid options are '%s', '%s', '%s'", sort, sortTime, sortScore, sortRandom)
		}
	}

	if limit := values.Get("page_size"); limit != "" {
		if intLimit, err := stringToInt(limit); err != nil {
			return nil, err
		} else {
			intLimit = clampLimit(intLimit, defaultPostsLimit, 1, maxPostsLimit)
			filter.Limit = &intLimit
		}
	}

	if key := values.Get("page_id"); key != "" {
		if keyset, err := stringToInt(key); err != nil {
			return nil, err
		} else {
			filter.Keyset = &keyset
		}
	}

	if seed := values.Get("seed"); seed != "" {
		if seed, err := strconv.Atoi(seed); err == nil && validSeed(seed) {
			filter.Seed = &seed
		}
	}

	if c := values.Get("cursor"); c != "" {
		cursor, seed, err := decodeCursor(c, *filter.Sort)
		if err != nil {
			return nil, err
		}
		filter.Cursor = cursor
		filter.Keyset = nil
		if seed != 0 {
			filter.Seed = &seed
		}
	}

	if *filter.Sort == analogdb.PostSortRandom {
		if filter.Keyset != nil {
			return nil, badRequest("page_id is not supported with sort=random; use cursor")
		}
		if filter.Seed == nil {
			seed := rand.IntN(randomSeedPool) + 1
			filter.Seed = &seed
		}
	} else {
		filter.Seed = nil
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

	if id := values.Get("id"); id != "" {
		if identify, err := stringToInt(id); err != nil {
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
		startInt, err := stringToInt64(start)
		if err != nil {
			return nil, err
		}
		startTime := time.Unix(startInt, 0)
		filter.TimeStart = &startTime
	}

	if end := values.Get("time_end"); end != "" {
		endInt, err := stringToInt64(end)
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
		if fsi, err := stringToInt(fs); err != nil {
			return nil, err
		} else {
			filter.FilmSpeed = &fsi
		}
	}

	if fl := values.Get("focal_length"); fl != "" {
		if fli, err := stringToInt(fl); err != nil {
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
			if percent, err := stringToFloat(p); err != nil {
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
		if width, err := stringToFloat(minWidth); err != nil {
			return nil, err
		} else {
			filter.Width.Min = &width
		}
	}

	if maxWidth := values.Get("width_max"); maxWidth != "" {
		if width, err := stringToFloat(maxWidth); err != nil {
			return nil, err
		} else {
			filter.Width.Max = &width
		}
	}

	if minHeight := values.Get("height_min"); minHeight != "" {
		if height, err := stringToFloat(minHeight); err != nil {
			return nil, err
		} else {
			filter.Height.Min = &height
		}
	}

	if maxHeight := values.Get("height_max"); maxHeight != "" {
		if height, err := stringToFloat(maxHeight); err != nil {
			return nil, err
		} else {
			filter.Height.Max = &height
		}
	}

	if minRatio := values.Get("ratio_min"); minRatio != "" {
		if ratio, err := stringToFloat(minRatio); err != nil {
			return nil, err
		} else {
			filter.AspectRatio.Min = &ratio
		}
	}

	if maxRatio := values.Get("ratio_max"); maxRatio != "" {
		if ratio, err := stringToFloat(maxRatio); err != nil {
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
		return nil, badRequest("must include post id to query similar from")
	}
	postID, err := stringToInt(id)
	if err != nil {
		return nil, err
	}
	filter.ID = &postID

	// if we are getting similar to that post, we don't want to match the same post
	excluded := []int{postID}
	filter.ExcludeIDs = &excluded

	if limit := r.URL.Query().Get("page_size"); limit != "" {
		if intLimit, err := stringToInt(limit); err != nil {
			return nil, err
		} else {
			intLimit = clampLimit(intLimit, defaultSimilarityLimit, 1, maxSimilarityLimit)
			filter.Limit = &intLimit
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
