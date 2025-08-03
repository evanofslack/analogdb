package analogdb

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

const defaultMinColorPercent = 0.0

// seeds for random post order
var primes = []int{11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67, 71, 73, 79, 83, 89, 97, 101, 107, 113, 131, 137, 149, 167, 173, 179, 191, 197, 227, 233, 239, 251, 257, 263}

// Image represents the source info for an image
type Image struct {
	Label  string `json:"resolution" example:"high"`
	Url    string `json:"url" example:"https://cloudfront.net/1.jpeg"`
	Width  int    `json:"width" example:"4800"`
	Height int    `json:"height" example:"7600"`
}

// Color represents a single color of an image
type Color struct {
	Hex     string  `json:"hex" example:"#837d5c"`
	Css     string  `json:"css" example:"dimgray"`
	Html    string  `json:"html" example:"gray"`
	Percent float64 `json:"percent" example:"0.55"`
}

// CreatePost is the model for creating a post.
// This includes info from the original reddit post
// as well as attributes about the image.
type CreatePost struct {
	Title       string    `json:"title" example:"A day at the fields [Nikon FM2 | Portra 400]"`
	Author      string    `json:"author" example:"thecameraman"`
	Permalink   string    `json:"permalink" example:"https://www.reddit.com/r/analog/comments/1/post"`
	Description *string   `json:"description,omitempty" example:"My favorite camera and film combo on 35mm at f/2.0"`
	Score       int       `json:"score" example:"1000"`
	Nsfw        bool      `json:"nsfw" example:"false"`
	Grayscale   bool      `json:"grayscale" example:"false"`
	Time        int       `json:"timestamp" example:"1752354541"`
	Sprocket    bool      `json:"sprocket" example:"false"`
	CameraMake  *string   `json:"camera_make,omitempty" example:"nikon"`
	CameraModel *string   `json:"camera_model,omitempty" example:"fm2"`
	FilmMake    *string   `json:"film_make,omitempty" example:"kodak"`
	FilmType    *string   `json:"film_type,omitempty" example:"color"`
	FilmSpeed   *int64    `json:"film_speed,omitempty" example:"400"`
	FocalLength *int64    `json:"focal_length,omitempty" example:"35"`
	Aperture    *string   `json:"aperture,omitempty" example:"f/2.0"`
	Images      []Image   `json:"images"`
	Colors      []Color   `json:"colors"`
	Keywords    []Keyword `json:"keywords"`
}

// DisplayPost is the model for displaying a post.
// Renames some of the json keys.
type DisplayPost struct {
	Title       string    `json:"title" example:"A day at the fields [Nikon FM2 | Portra 400]"`
	Author      string    `json:"author" example:"thecameraman"`
	Permalink   string    `json:"permalink" example:"https://www.reddit.com/r/analog/comments/1/post"`
	Description *string   `json:"description,omitempty" example:"My favorite camera and film combo on 35mm at f/2.0"`
	Score       int       `json:"score" example:"1000"`
	Nsfw        bool      `json:"nsfw" example:"false"`
	Grayscale   bool      `json:"grayscale" example:"false"`
	Time        int       `json:"timestamp" example:"1752354541"`
	Sprocket    bool      `json:"sprocket" example:"false"`
	CameraMake  *string   `json:"camera_make,omitempty" example:"nikon"`
	CameraModel *string   `json:"camera_model,omitempty" example:"fm2"`
	FilmMake    *string   `json:"film_make,omitempty" example:"kodak"`
	FilmType    *string   `json:"film_type,omitempty" example:"color"`
	FilmSpeed   *int64    `json:"film_speed,omitempty" example:"400"`
	FocalLength *int64    `json:"focal_length,omitempty" example:"35"`
	Aperture    *string   `json:"aperture,omitempty" example:"f/2.0"`
	Images      []Image   `json:"images"`
	Colors      []Color   `json:"colors"`
	Keywords    []Keyword `json:"keywords"`
}

// PatchPost is the model for patching a post.
// Intentionally only allow certain fields to be updated.
// Uses pointers and omit empty to allow partial unmarshalling
type PatchPost struct {
	Score       *int       `json:"score,omitempty" example:"1010"`
	Description *string    `json:"description,omitempty" example:"New description"`
	Nsfw        *bool      `json:"nsfw,omitempty" example:"true"`
	Grayscale   *bool      `json:"grayscale,omitempty" example:"false"`
	Sprocket    *bool      `json:"sprocket,omitempty" example:"true"`
	CameraMake  *string    `json:"camera_make,omitempty" example:"canon"`
	CameraModel *string    `json:"camera_model,omitempty" example:"ae-1"`
	FilmMake    *string    `json:"film_make,omitempty" example:"kodak"`
	FilmType    *string    `json:"film_type,omitempty" example:"gold 200"`
	FilmSpeed   *int       `json:"film_speed,omitempty" example:"200"`
	FocalLength *int       `json:"focal_length,omitempty" example:"50"`
	Aperture    *string    `json:"aperture,omitempty" example:"f/2.4"`
	Colors      *[]Color   `json:"colors,omitempty"`
	Keywords    *[]Keyword `json:"keywords,omitempty"`
}

// Post is the model of a returned post
// including the auto-incremented ID from the DB
type Post struct {
	Id int `json:"id" example:"1"`
	DisplayPost
}

type PostSort int

const (
	PostSortUnknown PostSort = iota
	PostSortTime
	PostSortScore
	PostSortRandom
)

func (s PostSort) String() string {
	switch s {
	case PostSortTime:
		return "time"
	case PostSortScore:
		return "score"
	case PostSortRandom:
		return "random"
	default:
		return "unknown"
	}
}

func PostSortFromString(s string) PostSort {
	switch strings.ToLower(s) {
	case "time":
		return PostSortTime
	case "score":
		return PostSortScore
	case "random":
		return PostSortRandom
	default:
		return PostSortUnknown
	}
}

// Dimension represents a dimension with
// optional minimum and maximum sizes.
type Dimension struct {
	Min *float64
	Max *float64
}

func (dim *Dimension) String() string {
	min, max := "min=nil", "max=nil"
	if dim.Min != nil {
		min = fmt.Sprintf("min=%.2f", *dim.Min)
	}
	if dim.Max != nil {
		max = fmt.Sprintf("max=%.2f", *dim.Max)
	}
	return fmt.Sprintf("%s, %s", min, max)
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
	TimeStart     *time.Time
	TimeEnd       *time.Time
	CameraMake    *string
	CameraModel   *string
	FilmMake      *string
	FilmType      *string
	FilmSpeed     *int
	FocalLength   *int
	Aperture      *string
	Colors        *[]string
	ColorPercents *[]float64
	Keywords      *[]string
	Width         *Dimension
	Height        *Dimension
	AspectRatio   *Dimension
}

func NewPostFilter(limit *int, sort *PostSort, keyset *int, nsfw, grayscale, sprocket *bool, seed *int, ids *[]int, title, author *string, timeStart, timeEnd *time.Time, cameraMake, cameraModel, filmMake, filmType *string, filmSpeed, focalLength *int, aperture *string, colors *[]string, colorPercents *[]float64, keywords *[]string) *PostFilter {
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
		TimeStart:     timeStart,
		TimeEnd:       timeEnd,
		CameraMake:    cameraMake,
		CameraModel:   cameraModel,
		FilmMake:      filmMake,
		FilmType:      filmType,
		FilmSpeed:     filmSpeed,
		FocalLength:   focalLength,
		Aperture:      aperture,
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
	return NewPostFilter(nil, nil, nil, nil, nil, nil, nil, &ids, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
}

func (filter *PostFilter) String() string {
	out := []string{}
	if filter.Limit != nil {
		out = append(out, fmt.Sprintf("limit: %d", *filter.Limit))
	}
	if filter.Sort != nil {
		out = append(out, fmt.Sprintf("sort: %s", *filter.Sort))
	}
	if filter.Keyset != nil {
		out = append(out, fmt.Sprintf("keyset: %d", *filter.Keyset))
	}
	if filter.Nsfw != nil {
		out = append(out, fmt.Sprintf("keyset: %t", *filter.Nsfw))
	}
	if filter.Grayscale != nil {
		out = append(out, fmt.Sprintf("grayscale: %t", *filter.Grayscale))
	}
	if filter.Sprocket != nil {
		out = append(out, fmt.Sprintf("sprocket: %t", *filter.Sprocket))
	}
	if filter.Seed != nil {
		out = append(out, fmt.Sprintf("seed: %d", *filter.Seed))
	}
	if filter.IDs != nil {
		out = append(out, fmt.Sprintf("ids: %v", *filter.IDs))
	}
	if filter.Title != nil {
		out = append(out, fmt.Sprintf("title: %s", *filter.Title))
	}
	if filter.Author != nil {
		out = append(out, fmt.Sprintf("author: %s", *filter.Author))
	}
	if filter.TimeStart != nil {
		out = append(out, fmt.Sprintf("time_start: %s", *filter.TimeStart))
	}
	if filter.TimeEnd != nil {
		out = append(out, fmt.Sprintf("time_end: %s", *filter.TimeEnd))
	}
	if filter.CameraMake != nil {
		out = append(out, fmt.Sprintf("camera_make: %s", *filter.CameraMake))
	}
	if filter.CameraModel != nil {
		out = append(out, fmt.Sprintf("camera_model: %s", *filter.CameraModel))
	}
	if filter.FilmMake != nil {
		out = append(out, fmt.Sprintf("film_make: %s", *filter.FilmMake))
	}
	if filter.FilmType != nil {
		out = append(out, fmt.Sprintf("film_type: %s", *filter.FilmType))
	}
	if filter.FilmSpeed != nil {
		out = append(out, fmt.Sprintf("film_speed: %d", *filter.FilmSpeed))
	}
	if filter.FocalLength != nil {
		out = append(out, fmt.Sprintf("focal_length: %d", *filter.FocalLength))
	}
	if filter.Aperture != nil {
		out = append(out, fmt.Sprintf("aperture: %s", *filter.Aperture))
	}
	if filter.Colors != nil {
		out = append(out, fmt.Sprintf("colors: %v", *filter.Colors))
	}
	if filter.ColorPercents != nil {
		out = append(out, fmt.Sprintf("color_percents: %v", *filter.ColorPercents))
	}
	if filter.Keywords != nil {
		out = append(out, fmt.Sprintf("keywords: %v", *filter.Keywords))
	}
	if filter.Width != nil {
		out = append(out, fmt.Sprintf("width: %s", filter.Width))
	}
	if filter.Height != nil {
		out = append(out, fmt.Sprintf("height: %s", filter.Height.String()))
	}
	if filter.AspectRatio != nil {
		out = append(out, fmt.Sprintf("aspect_ratio: %s", filter.AspectRatio))
	}
	return strings.Join(out, ", ")
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
    TotalPosts int    `json:"total_posts" example:"200"`
    PageSize   int    `json:"page_size" example:"20"`
    NextPageID string `json:"next_page_id" example:"1752244116"`
    PageURL    string `json:"next_page_url" example:"/posts?sort=time&page_size=20&page_id=1752244116"`
    Seed       int    `json:"seed,omitempty" example:"37"`
}

type PostService interface {
	FindPosts(ctx context.Context, filter *PostFilter) ([]*Post, int, error)
	FindPostByID(ctx context.Context, id int) (*Post, error)
	CreatePost(ctx context.Context, post *CreatePost) (*Post, error)
	PatchPost(ctx context.Context, post *PatchPost, id int) error
	DeletePost(ctx context.Context, id int) error
	AllPostIDs(ctx context.Context) ([]int, error)
}
