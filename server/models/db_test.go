package models

import (
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
		t.Errorf("failed to get latest post")
	}

	if len(resp.Posts) != 10 {
		t.Errorf("response should contain 10 posts")
	}

	latestTime := resp.Posts[0].Time
	for _, p := range resp.Posts {
		if p.Time > latestTime {
			t.Errorf("posts not sorted newest to oldest")
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
		t.Errorf("failed to get top post")
	}

	if len(resp.Posts) != 10 {
		t.Errorf("response should contain 10 posts")
	}

	topScore := resp.Posts[0].Score
	for _, p := range resp.Posts {
		if p.Score > topScore {
			t.Errorf("posts not sorted most votes to least votes")
		}
	}
}
