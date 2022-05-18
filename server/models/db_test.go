package models

import (
	"strconv"
	"testing"

	"github.com/joho/godotenv"
)

func TestDB(t *testing.T) {
	if err := godotenv.Load("../.env"); err != nil {
		t.Error("Error loading .env file")
	}

	err := InitDB(false)
	if err != nil {
		t.Errorf("failed to start DB")
	}
}

func TestLatestPost(t *testing.T) {
	const (
		limit     = 10
		time      = 0
		nsfw      = true
		grayscale = true
	)
	resp, err := LatestPost(limit, time, nsfw, grayscale)
	if err != nil {
		t.Errorf("failed to get latest posts")
	}

	if len(resp.Posts) != 10 {
		t.Errorf("response should contain 10 posts")
	}

	latestTime := resp.Posts[0].Time
	oldestTime := resp.Posts[9].Time
	for _, p := range resp.Posts {
		if p.Time > latestTime {
			t.Errorf("posts not sorted newest to oldest")
		}
	}

	time_next, err := strconv.Atoi(resp.Meta.PageID)
	if err != nil {
		t.Errorf("failed to parse next time")
	}

	resp, err = LatestPost(limit, time_next, nsfw, grayscale)
	if err != nil {
		t.Errorf("failed to get latest post")
	}

	for _, p := range resp.Posts {
		if p.Time > oldestTime {
			t.Errorf("time offset failed")
		}
	}
}

func TestTopPost(t *testing.T) {
	const (
		limit     = 10
		score     = 0
		nsfw      = true
		grayscale = true
	)
	resp, err := TopPost(limit, score, nsfw, grayscale)
	if err != nil {
		t.Errorf("failed to get top posts")
	}

	if len(resp.Posts) != 10 {
		t.Errorf("response should contain 10 posts")
	}

	topScore := resp.Posts[0].Score
	bottomScore := resp.Posts[9].Score
	for _, p := range resp.Posts {
		if p.Score > topScore {
			t.Errorf("posts not sorted most votes to least votes")
		}
	}

	score_next, err := strconv.Atoi(resp.Meta.PageID)
	if err != nil {
		t.Errorf("failed to parse next score")
	}

	resp, err = TopPost(limit, score_next, nsfw, grayscale)
	if err != nil {
		t.Errorf("failed to get top posts")
	}

	for _, p := range resp.Posts {
		if p.Score > bottomScore {
			t.Errorf("score offset failed")
		}
	}
}

func TestRandomPost(t *testing.T) {
	const (
		limit     = 20
		time      = 0
		nsfw      = true
		grayscale = true
		seed      = 0
	)
	resp, err := RandomPost(limit, time, nsfw, grayscale, seed)
	if err != nil {
		t.Errorf("failed to get random posts")
	}

	if len(resp.Posts) != 20 {
		t.Errorf("response should contain 20 posts")
	}

	seenPosts := make(map[string]bool)
	for _, p := range resp.Posts {
		seenPosts[p.Permalink] = true
	}

	timeNext, err := strconv.Atoi(resp.Meta.PageID)
	if err != nil {
		t.Errorf("failed to parse next time")
	}
	newSeed := resp.Meta.Seed

	resp, err = RandomPost(limit, timeNext, nsfw, grayscale, newSeed)

	if err != nil {
		t.Errorf("failed to get random posts")
	}

	for _, p := range resp.Posts {
		if seenPosts[p.Permalink] {
			t.Errorf("repeated random post")
		}
	}
}

func TestNsfwPost(t *testing.T) {
	const (
		limit = 4
		time  = 0
	)
	resp, err := NsfwPost(limit, time)
	if err != nil {
		t.Errorf("failed to get random posts")
	}

	if len(resp.Posts) != 4 {
		t.Errorf("response should contain 4 posts")
	}
	for _, p := range resp.Posts {
		if p.Nsfw == false {
			t.Errorf("Post should only be NSFW")
		}
	}
}

func TestBwPost(t *testing.T) {
	const (
		limit = 7
		time  = 0
	)
	resp, err := BwPost(limit, time)
	if err != nil {
		t.Errorf("failed to get b&w posts")
	}

	if len(resp.Posts) != 7 {
		t.Errorf("response should contain 7 posts")
	}
	for _, p := range resp.Posts {
		if p.Grayscale == false {
			t.Errorf("Post should only be B&W")
		}
	}
}

func TestSprocketPost(t *testing.T) {
	const (
		limit = 2
		time  = 0
	)
	resp, err := SprocketPost(limit, time)
	if err != nil {
		t.Errorf("failed to get sprocket posts")
	}

	if len(resp.Posts) != 2 {
		t.Errorf("response should contain 2 posts")
	}
	for _, p := range resp.Posts {
		if p.Sprocket == false {
			t.Errorf("Post should only be sprocket shots")
		}
	}
}

func TestFindPost(t *testing.T) {
	id := 2058
	author := "u/MichaWha"
	post, err := FindPost(id)
	if err != nil {
		t.Errorf("failed to find post")
	}
	if post.Author != author {
		t.Errorf("found post is incorrect")
	}
}

func TestFindNonExistPost(t *testing.T) {
	id := 5000
	post, err := FindPost(id)
	if err != nil {
		t.Errorf("post does not exist but should not return error")
	}
	if len(post.Images) != 0 {
		t.Errorf("should return zero value post")
	}

}
