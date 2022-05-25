package analogdb

import "context"

// Image represents the source info for an image.
type Image struct {
	Label  string `json:"resolution"`
	Url    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Post is the attributes associated with an image.
// This includes info from the original reddit post
// as well as attributes about the image.
type Post struct {
	Id        int     `json:"id"`
	Images    []Image `json:"images"`
	Title     string  `json:"title"`
	Author    string  `json:"author"`
	Permalink string  `json:"permalink"`
	Score     int     `json:"upvotes"`
	Nsfw      bool    `json:"nsfw"`
	Grayscale bool    `json:"grayscale"`
	Time      int     `json:"unix_time"`
	Sprocket  bool    `json:"sprocket"`
}

// PostFilter are options used for querying posts
type PostFilter struct {
	Limit     *int
	Time      *int
	Score     *int
	Nsfw      *bool
	Grayscale *bool
	Sprocket  *bool
	Seed      *int
	ID        *int
	Title     *string
	Author    *string
}

// Meta includes details about the response.
type Meta struct {
	TotalPosts int    `json:"total_posts"`
	PageSize   int    `json:"page_size"`
	PageID     string `json:"next_page_id"`
	PageURL    string `json:"next_page_url"`
	Seed       int    `json:"seed,omitempty"`
}

// HTTP response
type Response struct {
	Meta  Meta   `json:"meta"`
	Posts []Post `json:"posts"`
}

type PostService interface {
	LatestPosts(ctx context.Context, filter *PostFilter) ([]*Post, int, error)
	TopPosts(ctx context.Context, filter *PostFilter) ([]*Post, int, error)
	RandomPosts(ctx context.Context, filter *PostFilter) ([]*Post, int, error)
	FindPostByID(ctx context.Context, id int) (*Post, error)
	DeletePost(ctx context.Context, id int) (*Post, error)
}
