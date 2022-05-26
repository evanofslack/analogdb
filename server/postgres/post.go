package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"strings"

	"github.com/evanofslack/analogdb"
)

// ensure interface is implemented
var _ analogdb.PostService = (*PostService)(nil)

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

// Sorting constants
const (
	time   = "time"
	score  = "score"
	random = "random"
)

func (s *PostService) FindPosts(ctx context.Context, filter *analogdb.PostFilter) ([]*analogdb.Post, int, error) {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback()
	return findPosts(ctx, tx, filter)
}

func (s *PostService) FindPostByID(ctx context.Context, id int) (*analogdb.Post, error) {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	posts, _, err := findPosts(ctx, tx, &analogdb.PostFilter{ID: &id})
	if err != nil {
		return nil, err
	} else if len(posts) == 0 {
		return nil, errors.New("post not found")
	}
	return posts[0], nil
}

func (s *PostService) DeletePost(ctx context.Context, id int) (*analogdb.Post, error) {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	posts, err := deletePost(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if len(posts) == 0 {
		return nil, errors.New("post not found")
	}
	return posts[0], nil
}

// findPosts is the general function responsible for handling all queries
func findPosts(ctx context.Context, tx *sql.Tx, filter *analogdb.PostFilter) ([]*analogdb.Post, int, error) {

	if err := validateFilter(filter); err != nil {
		return nil, 0, err
	}

	where, args := filterToWhere(filter)
	order := filterToOrder(filter)
	limit := formatLimit(filter)
	query := `
			SELECT
				id,
				url,
				title,
				author,
				permalink,
				score,
				nsfw,
				greyscale,
				time,
				width,
				height,
				sprocket,
				lowUrl,
				lowWidth,
				lowHeight,
				medUrl,
				medWidth,
				medHeight,
				highUrl,
				highWidth,
				highHeight,
				COUNT(*) OVER()
			FROM pictures ` + where + order + limit
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

func deletePost(ctx context.Context, tx *sql.Tx, id int) ([]*analogdb.Post, error) {
	query := "DELETE FROM pictures WHERE id = $1"

	rows, err := tx.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := make([]*analogdb.Post, 0)
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
		); err != nil {
			return nil, err
		}
		lowImage := analogdb.Image{Label: "low", Url: p.lowUrl, Width: p.lowWidth, Height: p.lowHeight}
		medImage := analogdb.Image{Label: "medium", Url: p.medUrl, Width: p.medWidth, Height: p.medHeight}
		highImage := analogdb.Image{Label: "high", Url: p.highUrl, Width: p.highWidth, Height: p.highHeight}
		rawImage := analogdb.Image{Label: "raw", Url: p.url, Width: p.width, Height: p.height}
		images := []analogdb.Image{lowImage, medImage, highImage, rawImage}

		post := &analogdb.Post{Id: p.id, Images: images, Title: p.title, Author: p.author, Permalink: p.permalink, Score: p.score, Nsfw: p.nsfw, Grayscale: p.grayscale, Time: p.time, Sprocket: p.sprocket}
		posts = append(posts, post)
	}
	return posts, nil
}

// filterToOrder converts filter into an SQL "ORDER BY" statement
func filterToOrder(filter *analogdb.PostFilter) string {
	if sort := filter.Sort; sort != nil {
		switch *sort {
		case time:
			return " ORDER BY time DESC"
		case score:
			return "ORDER BY score DESC"
		case random:
			if seed := filter.Seed; seed != nil {
				return fmt.Sprintf(" ORDER BY MOD(time, %d), time DESC", *seed)
			} else {
				newSeed := seedGenerator()
				filter.Seed = &newSeed
				return fmt.Sprintf(" ORDER BY MOD(time, %d), time DESC", newSeed)
			}
		}
	}
	return ""
}

// formatLimit turns the limit into an SQL limit statement
func formatLimit(filter *analogdb.PostFilter) string {
	if limit := filter.Limit; limit != nil {
		if *limit > 0 {
			return fmt.Sprintf(` LIMIT %d`, *limit)
		}
	}
	return ""
}

// seedGenerator generates a random prime number
func seedGenerator() int {
	prime_seeds := []int{11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67, 71, 73, 79, 83, 89, 97}
	randomIndex := rand.Intn(len(prime_seeds))
	return prime_seeds[randomIndex]
}

// filterToWhere converts a PostFilter to an SQL WHERE statement
func filterToWhere(filter *analogdb.PostFilter) (string, []any) {
	index := 1
	where, args := []string{"1=1"}, []any{}

	if sort, keyset := filter.Sort, filter.Keyset; sort != nil && keyset != nil {
		switch *sort {
		case time:
			where = append(where, fmt.Sprintf("time < $%d", index))
			args = append(args, *keyset)
			index += 1
		case score:
			where = append(where, fmt.Sprintf("score < $%d", index))
			args = append(args, *keyset)
			index += 1
		case random:
			if seed := filter.Seed; seed != nil {
				where = append(where, fmt.Sprintf("MOD(time, $%d) > $%d", index, index+1))
				args = append(args, *seed, *keyset%*seed)
				index += 2
			}
		}
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
	if id := filter.ID; id != nil {
		where = append(where, fmt.Sprintf("id = $%d", index))
		args = append(args, *id)
		index += 1
	}
	// match partial text in post title with ILIKE
	if title := filter.Title; title != nil {
		where = append(where, fmt.Sprintf("title ILIKE $%d", index))
		args = append(args, "%"+*title+"%")
		index += 1
	}
	// if query does not prefix author with 'u/' we need to add it
	if author := filter.Author; author != nil {
		var matchAuthor string
		if pre := (*author)[0:2]; pre != "u/" {
			matchAuthor = "u/" + *author
		} else {
			matchAuthor = *author
		}
		where = append(where, fmt.Sprintf("author = $%d", index))
		args = append(args, matchAuthor)
		index += 1
	}
	return `WHERE ` + strings.Join(where, " AND "), args

}

// validateSort ensures that provided sort method is defined
func validateFilter(filter *analogdb.PostFilter) error {
	validSort := make(map[string]bool)
	validSort[time] = true
	validSort[score] = true
	validSort[random] = true

	if sort := filter.Sort; sort != nil {
		if !validSort[*sort] {
			return errors.New("invalid sort parameter, valid options are 'time', 'score' or 'random'")
		}
		return nil
	}
	return nil
}
