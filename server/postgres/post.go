package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/evanofslack/analogdb"
)

// rawPost corresponds to the columns as a post is stored in the DB
type rawPost struct {
	id         int
	url        string
	title      string
	author     string
	permalink  string
	score      int
	nsfw       bool
	grayscale  bool
	time       int
	width      int
	height     int
	sprocket   bool
	lowUrl     string
	lowWidth   int
	lowHeight  int
	medUrl     string
	medWidth   int
	medHeight  int
	highUrl    string
	highWidth  int
	highHeight int
}

type PostService struct {
	db *DB
}

func NewPostService(db *DB) *PostService {
	return &PostService{db: db}
}

func (s *PostService) LatestPosts(ctx context.Context, filter analogdb.PostFilter) ([]*analogdb.Post, int, error) {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback()
	return latestPost(ctx, tx, filter)
}

func latestPost(ctx context.Context, tx *sql.Tx, filter analogdb.PostFilter) ([]*analogdb.Post, int, error) {
	index := 1
	where, args := []string{"1=1"}, []any{}
	if time := filter.Time; time != nil {
		where = append(where, fmt.Sprintf("time < $%d", index))
		args = append(args, *time)
		index += 1
	}
	if nsfw := filter.Nsfw; nsfw != nil {
		where = append(where, fmt.Sprintf("nsfw = $%d", index))
		args = append(args, *nsfw)
		index += 1
	}
	if grayscale := filter.Grayscale; grayscale != nil {
		where = append(where, fmt.Sprintf("greyscale = $%d", index))
		args = append(args, *grayscale)
		index += 1
	}
	if sprocket := filter.Sprocket; sprocket != nil {
		where = append(where, fmt.Sprintf("sprocket = $%d", index))
		args = append(args, *sprocket)
		index += 1
	}

	query := `
			SELECT
				id
				url
				title
				author
				permalink
				score
				nsfw
				grayscale
				time
				width
				height
				sprocket
				lowUrl
				lowWidth
				lowHeight
				medUrl
				medWidth
				medHeight
				highUrl
				highWidth
				highHeight
				COUNT(*) OVER()
			FROM pictures 
			WHERE ` + strings.Join(where, " AND ") + `
			ORDER BY time DESC
			` + FormatLimit(*filter.Limit)

	rows, err := tx.QueryContext(ctx, query, args...)

	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	posts := make([]*analogdb.Post, 0)
	var count int
	for rows.Next() {
		var p rawPost
		if err := rows.Scan(
			&p.id,
			&p.url,
			&p.title,
			&p.author,
			&p.permalink,
			&p.score,
			&p.nsfw,
			&p.grayscale,
			&p.time,
			&p.width,
			&p.height,
			&p.sprocket,
			&p.lowUrl,
			&p.lowWidth,
			&p.lowHeight,
			&p.medUrl,
			&p.medWidth,
			&p.medHeight,
			&p.highUrl,
			&p.highWidth,
			&p.highHeight,
			&count,
		); err != nil {
			return nil, 0, err
		}
		lowImage := analogdb.Image{Label: "low", Url: p.lowUrl, Width: p.lowWidth, Height: p.lowHeight}
		medImage := analogdb.Image{Label: "medium", Url: p.medUrl, Width: p.medWidth, Height: p.medHeight}
		highImage := analogdb.Image{Label: "high", Url: p.highUrl, Width: p.highWidth, Height: p.highHeight}
		rawImage := analogdb.Image{Label: "raw", Url: p.url, Width: p.width, Height: p.height}
		images := []analogdb.Image{lowImage, medImage, highImage, rawImage}

		post := &analogdb.Post{Id: p.id, Images: images, Title: p.title, Author: p.author, Permalink: p.permalink, Score: p.score, Nsfw: p.nsfw, Grayscale: p.grayscale, Time: p.time, Sprocket: p.sprocket}
		posts = append(posts, post)
	}
	return posts, count, nil
}
