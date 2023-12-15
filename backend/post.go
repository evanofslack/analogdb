package analogdb

import (
	"context"
	"math/rand"
	"strings"
)

const defaultMinColorPercent = 0.0

// seeds for random post order
var primes = []int{11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67, 71, 73, 79, 83, 89, 97, 101, 107, 113, 131, 137, 149, 167, 173, 179, 191, 197, 227, 233, 239, 251, 257, 263}

// Image represents the source info for an image.
type Image struct {
	Label  string `json:"resolution"`
	Url    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Color represents a single color of an image
type Color struct {
	Hex     string  `json:"hex"`
	Css     string  `json:"css"`
	Html    string  `json:"html"`
	Percent float64 `json:"percent"`
}

// CreatePost is the model for creating a post.
// This includes info from the original reddit post
// as well as attributes about the image
type CreatePost struct {
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	Permalink string    `json:"permalink"`
	Score     int       `json:"upvotes"`
	Nsfw      bool      `json:"nsfw"`
	Grayscale bool      `json:"grayscale"`
	Time      int       `json:"unix_time"`
	Sprocket  bool      `json:"sprocket"`
	Images    []Image   `json:"images"`
	Colors    []Color   `json:"colors"`
	Keywords  []Keyword `json:"keywords"`
}

// DisplayPost is the model for displaying a post.
// Renames some of the json keys.
type DisplayPost struct {
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	Permalink string    `json:"permalink"`
	Score     int       `json:"score"`
	Nsfw      bool      `json:"nsfw"`
	Grayscale bool      `json:"grayscale"`
	Time      int       `json:"timestamp"`
	Sprocket  bool      `json:"sprocket"`
	Images    []Image   `json:"images"`
	Colors    []Color   `json:"colors"`
	Keywords  []Keyword `json:"keywords,omitempty"`
}

// PatchPost is the model for patching a post.
// Intentionally only allow certain fields to be updated.
// Uses pointers and omit empty to allow partial unmarshalling
type PatchPost struct {
	Score     *int       `json:"upvotes,omitempty"`
	Nsfw      *bool      `json:"nsfw,omitempty"`
	Grayscale *bool      `json:"grayscale,omitempty"`
	Sprocket  *bool      `json:"sprocket,omitempty"`
	Colors    *[]Color   `json:"colors,omitempty"`
	Keywords  *[]Keyword `json:"keywords,omitempty"`
}

// Post is the model of a returned post
// including the auto-incremented ID from the DB
type Post struct {
	Id int `json:"id"`
	DisplayPost
}

type PostSort int

const (
	SortUnknown PostSort = iota
	SortTime
	SortScore
	SortRandom
)

func (s PostSort) String() string {
	switch s {
	case SortTime:
		return "time"
	case SortScore:
		return "score"
	case SortRandom:
		return "random"
	default:
		return "unknown"
	}
}

func PostSortFromString(s string) PostSort {
	switch strings.ToLower(s) {
	case "time":
		return SortTime
	case "score":
		return SortScore
	case "random":
		return SortRandom
	default:
		return SortUnknown
	}
}

// Dimension represents a dimension with
// optional minimum and maximum sizes.
type Dimension struct {
	Min *float64
	Max *float64
}

// PostFilter are options used for querying posts
type PostFilter struct {
	Limit         *int
	Sort          *PostSort
	Keyset        *int
	Nsfw          *bool
	Grayscale     *bool
	Sprocket      *bool
	Seed          *int
	IDs           *[]int
	Title         *string
	Author        *string
	Colors        *[]string
	ColorPercents *[]float64
	Keywords      *[]string
	Width         *Dimension
	Height        *Dimension
	AspectRatio   *Dimension
}

func (filter *PostFilter) SetSeed() {
	if filter.Seed == nil {
		randomIndex := rand.Intn(len(primes))
		seed := primes[randomIndex]
		filter.Seed = &seed
	}
}

func (filter *PostFilter) SetMinColorPercent() {

	// If we have no colors, should have no percent
	if filter.Colors == nil {
		filter.ColorPercents = nil
		return
	}

	// don't have a valid pointer, create one
	if filter.ColorPercents == nil {
		percents := []float64{}
		filter.ColorPercents = &percents
	}

	colors, percents := *filter.Colors, *filter.ColorPercents

	// ensure at least as long as colors
	for len(colors) > len(percents) {
		percents = append(percents, defaultMinColorPercent)
	}

	// ensure at no longer than colors
	for len(percents) > len(colors) {
		if count := len(percents); count > 0 {
			percents = (percents)[:count-1]
		}
	}

	// finally, set modified back as pointer
	filter.ColorPercents = &percents
}

func NewPostFilter(limit *int, sort *PostSort, keyset *int, nsfw, grayscale, sprocket *bool, seed *int, ids *[]int, title, author *string, colors *[]string, colorPercents *[]float64, keywords *[]string) *PostFilter {

	filter := &PostFilter{
		Limit:         limit,
		Sort:          sort,
		Keyset:        keyset,
		Nsfw:          nsfw,
		Grayscale:     grayscale,
		Sprocket:      sprocket,
		Seed:          seed,
		IDs:           ids,
		Title:         title,
		Author:        author,
		Colors:        colors,
		ColorPercents: colorPercents,
		Keywords:      keywords,
		Width:         &Dimension{},
		Height:        &Dimension{},
		AspectRatio:   &Dimension{},
	}

	filter.SetMinColorPercent()

	return filter
}

// NewPostFilterWithIDs is a convenience function
// to create a post filter with only IDs set.
func NewPostFilterWithIDs(ids []int) *PostFilter {
	return NewPostFilter(nil, nil, nil, nil, nil, nil, nil, &ids, nil, nil, nil, nil, nil)
}

// PostSimilarityFilter are options used for querying similar posts
type PostSimilarityFilter struct {
	Limit      *int
	Nsfw       *bool
	Grayscale  *bool
	Sprocket   *bool
	ID         *int
	ExcludeIDs *[]int
}

func NewPostSimilarityFilter(limit *int, nsfw, grayscale, sprocket *bool, id *int, excludedIDs []int) PostSimilarityFilter {
	filter := PostSimilarityFilter{
		Limit:      limit,
		Nsfw:       nsfw,
		Grayscale:  grayscale,
		Sprocket:   sprocket,
		ID:         id,
		ExcludeIDs: &excludedIDs,
	}
	return filter
}

// Meta includes details about the response.
type Meta struct {
	TotalPosts int    `json:"total_posts"`
	PageSize   int    `json:"page_size"`
	NextPageID string `json:"next_page_id"`
	PageURL    string `json:"next_page_url"`
	Seed       int    `json:"seed,omitempty"`
}

// HTTP response
type Response struct {
	Meta  Meta   `json:"meta"`
	Posts []Post `json:"posts"`
}

type PostService interface {
	FindPosts(ctx context.Context, filter *PostFilter) ([]*Post, int, error)
	FindPostByID(ctx context.Context, id int) (*Post, error)
	CreatePost(ctx context.Context, post *CreatePost) (*Post, error)
	PatchPost(ctx context.Context, post *PatchPost, id int) error
	DeletePost(ctx context.Context, id int) error
	AllPostIDs(ctx context.Context) ([]int, error)
}
