package http

import (
	"fmt"
	"net/http"

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

func (s *Server) latestPosts(w http.ResponseWriter, r *http.Request) {
	sort := "time"
	filter := &analogdb.PostFilter{Sort: &sort}
	posts, count, err := s.PostService.FindPosts(r.Context(), filter)
	if err != nil {
		writeError(w, r, err)
	}
	fmt.Println(posts, count)

}

func parseToFilter(r *http.Request) *analogdb.PostFilter {
	// limit := r.URL.Query().Get("page_size")
	// pageID := r.URL.Query().Get("page_id")
	// nsfw := r.URL.Query().Get("nsfw")
	// grayscale := r.URL.Query().Get("bw")
	// sprocket := r.URL.Query().Get("sprocket")
	// seed := r.URL.Query().Get("seed")
	// id := r.URL.Query().Get("id")
	// title := r.URL.Query().Get("title")
	// author := r.URL.Query().Get("author")

	return nil
}

func queryParam(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}
