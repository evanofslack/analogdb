package analogdb

import "context"

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
	Percent float64 `json:"percent"`
}

// Keyword represents a single word/tag for a post
type Keyword struct {
	Word   string  `json:"word"`
	Weight float64 `json:"weight"`
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

// PostFilter are options used for querying posts
type PostFilter struct {
	Limit     *int
	Sort      *string
	Keyset    *int
	Nsfw      *bool
	Grayscale *bool
	Sprocket  *bool
	Seed      *int
	IDs       *[]int
	Title     *string
	Author    *string
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