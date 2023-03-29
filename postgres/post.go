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

// rawPostCreate corresponds to the columns as a post is inserted in DB
type rawCreatePost struct {
	url              string
	title            string
	author           string
	permalink        string
	score            int
	nsfw             bool
	grayscale        bool
	time             int
	width            int
	height           int
	sprocket         bool
	lowUrl           string
	lowWidth         int
	lowHeight        int
	medUrl           string
	medWidth         int
	medHeight        int
	highUrl          string
	highWidth        int
	highHeight       int
	c1_hex           string
	c1_css           string
	c1_percent       float64
	c2_hex           string
	c2_css           string
	c2_percent       float64
	c3_hex           string
	c3_css           string
	c3_percent       float64
	c4_hex           string
	c4_css           string
	c4_percent       float64
	c5_hex           string
	c5_css           string
	c5_percent       float64
	keywords         string
	keywords_percent string
}

// rawPost corresponds to the columns as a post is selected from the DB
type rawPost struct {
	id int
	rawCreatePost
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

func (s *PostService) CreatePost(ctx context.Context, post *analogdb.CreatePost) (*analogdb.Post, error) {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	createdPost, err := createPost(ctx, tx, post)
	if err != nil {
		return nil, err
	}
	return createdPost, nil
}

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
		return nil, &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "Post not found"}
	}
	return posts[0], nil
}

func (s *PostService) PatchPost(ctx context.Context, patch *analogdb.PatchPost, id int) error {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = patchPost(ctx, tx, patch, id)
	if err != nil {
		return &analogdb.Error{Code: analogdb.ERRINTERNAL, Message: err.Error()}
	}
	return nil
}

func (s *PostService) DeletePost(ctx context.Context, id int) error {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = deletePost(ctx, tx, id)
	if err != nil {
		return &analogdb.Error{Code: analogdb.ERRINTERNAL, Message: err.Error()}
	}
	return nil
}

func (s *PostService) AllPostIDs(ctx context.Context) ([]int, error) {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	ids, err := allPostIDs(ctx, tx)

	if err != nil {
		return nil, err
	}

	return ids, nil
}

func createPost(ctx context.Context, tx *sql.Tx, post *analogdb.CreatePost) (*analogdb.Post, error) {

	create, err := createPostToRawPostCreate(post)
	if err != nil {
		return nil, err
	}

	var id int64

	query :=
		`
	INSERT INTO pictures
	(url, title, author, permalink, score, nsfw, greyscale, time, width, height, sprocket, lowUrl, lowWidth, lowHeight, medUrl, medWidth, medHeight, highUrl, highWidth, highHeight, c1_hex, c1_css, c1_percent, c2_hex, c2_css, c2_percent, c3_hex, c3_css, c3_percent, c4_hex, c4_css, c4_percent, c5_hex, c5_css, c5_percent)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35)
	ON CONFLICT (permalink) DO NOTHING
	RETURNING id
	`

	stmt, err := tx.PrepareContext(ctx, query)

	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	err = stmt.QueryRowContext(
		ctx,
		create.url,
		create.title,
		create.author,
		create.permalink,
		create.score,
		create.nsfw,
		create.grayscale,
		create.time,
		create.width,
		create.height,
		create.sprocket,
		create.lowUrl,
		create.lowWidth,
		create.lowHeight,
		create.medUrl,
		create.medWidth,
		create.medHeight,
		create.highUrl,
		create.highWidth,
		create.highHeight,
		create.c1_hex,
		create.c1_css,
		create.c1_percent,
		create.c2_hex,
		create.c2_css,
		create.c2_percent,
		create.c3_hex,
		create.c3_css,
		create.c3_percent,
		create.c4_hex,
		create.c4_css,
		create.c4_percent,
		create.c5_hex,
		create.c5_css,
		create.c5_percent).Scan(&id)

	if err != nil {
		return nil, err
	}

	err = tx.Commit()

	if err != nil {
		return nil, err
	}

	// convert the CreatePost to a DisplayPost for return.
	displayPost := analogdb.DisplayPost{
		Title:     post.Title,
		Author:    post.Author,
		Permalink: post.Permalink,
		Score:     post.Score,
		Nsfw:      post.Nsfw,
		Grayscale: post.Grayscale,
		Time:      post.Time,
		Sprocket:  post.Sprocket,
		Images:    post.Images,
		Colors:    post.Colors,
	}

	createdPost := &analogdb.Post{
		Id:          int(id),
		DisplayPost: displayPost,
	}
	return createdPost, nil
}

// findPosts is the general function responsible for handling all queries
func findPosts(ctx context.Context, tx *sql.Tx, filter *analogdb.PostFilter) ([]*analogdb.Post, int, error) {

	if err := validateFilter(filter); err != nil {
		return nil, 0, err
	}

	where, args := filterToWhere(filter)
	groupby := ` GROUP BY p.id`
	order := filterToOrder(filter)
	limit := formatLimit(filter)
	query := `
			SELECT
				p.id,
				p.url,
				p.title,
				p.author,
				p.permalink,
				p.score,
				p.nsfw,
				p.greyscale,
				p.time,
				p.width,
				p.height,
				p.sprocket,
				p.lowUrl,
				p.lowWidth,
				p.lowHeight,
				p.medUrl,
				p.medWidth,
				p.medHeight,
				p.highUrl,
				p.highWidth,
				p.highHeight,
				p.c1_hex,
				p.c1_css,
				p.c1_percent,
				p.c2_hex,
				p.c2_css,
				p.c2_percent,
				p.c3_hex,
				p.c3_css,
				p.c3_percent,
				p.c4_hex,
				p.c4_css,
				p.c4_percent,
				p.c5_hex,
				p.c5_css,
				p.c5_percent,
				STRING_AGG(k.word, ',' ORDER BY k.percent DESC) as keywords,
				ARRAY_AGG(k.percent, ORDER BY k.percent DESC) as keywords,
				COUNT(*) OVER()
			FROM pictures p
			LEFT OUTER JOIN keywords k ON (k.post_id = p.id)` + where + groupby + order + limit

	rows, err := tx.QueryContext(ctx, query, args...)

	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	posts := make([]*analogdb.Post, 0)
	var count int
	var p *rawPost
	for rows.Next() {
		p, count, err = scanRowToRawPostCount(rows)
		if err != nil {
			return nil, 0, err
		}
		post, err := rawPostToPost(*p)

		// strip `u/` prefix from author (modifies in place)
		stripAuthorPrefix(post)

		if err != nil {
			return nil, 0, err
		}
		posts = append(posts, post)

	}

	err = tx.Commit()
	if err != nil {
		return nil, 0, err
	}

	return posts, count, nil
}

func patchPost(ctx context.Context, tx *sql.Tx, patch *analogdb.PatchPost, id int) error {

	set, args, err := patchToSet(patch)
	if err != nil {
		return err
	}

	args = append(args, id)
	idPos := len(args)

	query :=
		"UPDATE pictures " + set + fmt.Sprintf(" WHERE id =  $%d", idPos)

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}

	defer rows.Close()

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func deletePost(ctx context.Context, tx *sql.Tx, id int) error {
	query := `
			DELETE FROM pictures
			WHERE id = $1
			RETURNING id`

	row := tx.QueryRowContext(ctx, query, id)

	var returnedID int
	err := row.Scan(&returnedID)
	if err != nil {
		return err
	}

	if id != returnedID {
		return fmt.Errorf("error deleting post with id %d", id)
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func allPostIDs(ctx context.Context, tx *sql.Tx) ([]int, error) {
	query := `
			SELECT id FROM pictures ORDER BY id ASC`
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int, 0)
	var id int
	for rows.Next() {
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// filterToOrder converts filter into an SQL "ORDER BY" statement
func filterToOrder(filter *analogdb.PostFilter) string {
	if sort := filter.Sort; sort != nil {
		switch *sort {
		case time:
			return " ORDER BY time DESC"
		case score:
			return " ORDER BY score DESC"
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

	// no where arguments provided
	// if index == 1 {
	// 	return ``, args
	// }

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

// Converts a patch to an SQL set statement
func patchToSet(patch *analogdb.PatchPost) (string, []any, error) {

	index := 1
	set, args := []string{}, []any{}

	if score := patch.Score; score != nil {
		set = append(set, fmt.Sprintf("score = $%d", index))
		args = append(args, *score)
		index += 1
	}

	if nsfw := patch.Nsfw; nsfw != nil {
		set = append(set, fmt.Sprintf("nsfw = $%d", index))
		args = append(args, *nsfw)
		index += 1
	}
	if grayscale := patch.Grayscale; grayscale != nil {
		set = append(set, fmt.Sprintf("greyscale = $%d", index))
		args = append(args, *grayscale)
		index += 1
	}
	if sprocket := patch.Sprocket; sprocket != nil {
		set = append(set, fmt.Sprintf("sprocket = $%d", index))
		args = append(args, *sprocket)
		index += 1
	}
	if colors := patch.Colors; colors != nil {
		if len(*colors) != 5 {
			return "", args, fmt.Errorf("Invalid color array provided, expected %d colors, got %d", 5, len(*colors))
		}

		// for each color in colors, we need to append hex, css and percent fields
		for i, color := range *colors {

			// add the hex
			set = append(set, fmt.Sprintf("c%d_hex = $%d", i+1, index))
			args = append(args, color.Hex)
			index += 1

			// add the css
			set = append(set, fmt.Sprintf("c%d_css = $%d", i+1, index))
			args = append(args, color.Css)
			index += 1

			// add the percent
			set = append(set, fmt.Sprintf("c%d_percent = $%d", i+1, index))
			args = append(args, color.Percent)
			index += 1
		}
	}

	// no update fields provided
	if len(set) == 0 {
		return "", args, fmt.Errorf("No updated fields were provided in patch")
	}

	return `SET ` + strings.Join(set, ", "), args, nil
}

func createPostToRawPostCreate(p *analogdb.CreatePost) (*rawCreatePost, error) {
	if len(p.Images) != 4 {
		return nil, &analogdb.Error{Code: analogdb.ERRUNPROCESSABLE, Message: "Unable to create post, expected 4 images (low, medium, high, raw)"}
	}
	low := p.Images[0]
	med := p.Images[1]
	high := p.Images[2]
	raw := p.Images[3]

	if len(p.Colors) != 5 {
		fmt.Println(p.Colors)
		fmt.Println(len(p.Colors))
		fmt.Println("HEREERERERE")
		return nil, &analogdb.Error{Code: analogdb.ERRUNPROCESSABLE, Message: "Unable to create post, expected 5 colors"}
	}
	c1 := p.Colors[0]
	c2 := p.Colors[1]
	c3 := p.Colors[2]
	c4 := p.Colors[3]
	c5 := p.Colors[4]

	post := &rawCreatePost{
		url:        raw.Url,
		title:      p.Title,
		author:     p.Author,
		permalink:  p.Permalink,
		score:      p.Score,
		nsfw:       p.Nsfw,
		grayscale:  p.Grayscale,
		time:       p.Time,
		width:      raw.Width,
		height:     raw.Height,
		sprocket:   p.Sprocket,
		lowUrl:     low.Url,
		lowWidth:   low.Width,
		lowHeight:  low.Height,
		medUrl:     med.Url,
		medWidth:   med.Width,
		medHeight:  med.Height,
		highUrl:    high.Url,
		highWidth:  high.Width,
		highHeight: high.Height,
		c1_hex:     c1.Hex,
		c1_css:     c1.Css,
		c1_percent: c1.Percent,
		c2_hex:     c2.Hex,
		c2_css:     c2.Css,
		c2_percent: c2.Percent,
		c3_hex:     c3.Hex,
		c3_css:     c3.Css,
		c3_percent: c3.Percent,
		c4_hex:     c4.Hex,
		c4_css:     c4.Css,
		c4_percent: c4.Percent,
		c5_hex:     c5.Hex,
		c5_css:     c5.Css,
		c5_percent: c5.Percent,
	}
	return post, nil

}

func rawPostToPost(p rawPost) (*analogdb.Post, error) {

	// grab the images from raw
	lowImage := analogdb.Image{Label: "low", Url: p.lowUrl, Width: p.lowWidth, Height: p.lowHeight}
	medImage := analogdb.Image{Label: "medium", Url: p.medUrl, Width: p.medWidth, Height: p.medHeight}
	highImage := analogdb.Image{Label: "high", Url: p.highUrl, Width: p.highWidth, Height: p.highHeight}
	rawImage := analogdb.Image{Label: "raw", Url: p.url, Width: p.width, Height: p.height}
	images := []analogdb.Image{lowImage, medImage, highImage, rawImage}

	// grab the colors from raw
	c1 := analogdb.Color{Hex: p.c1_hex, Css: p.c1_css, Percent: p.c1_percent}
	c2 := analogdb.Color{Hex: p.c2_hex, Css: p.c2_css, Percent: p.c2_percent}
	c3 := analogdb.Color{Hex: p.c3_hex, Css: p.c3_css, Percent: p.c3_percent}
	c4 := analogdb.Color{Hex: p.c4_hex, Css: p.c4_css, Percent: p.c4_percent}
	c5 := analogdb.Color{Hex: p.c5_hex, Css: p.c5_css, Percent: p.c5_percent}
	colors := []analogdb.Color{c1, c2, c3, c4, c5}

	post := &analogdb.Post{Id: p.id,
		DisplayPost: analogdb.DisplayPost{Title: p.title, Author: p.author, Permalink: p.permalink, Score: p.score, Nsfw: p.nsfw, Grayscale: p.grayscale, Time: p.time, Sprocket: p.sprocket, Images: images, Colors: colors}}
	return post, nil
}

func scanRowToRawPost(rows *sql.Rows) (*rawPost, error) {
	var p rawPost
	if err := rows.Scan(
		&p.id,
		&p.rawCreatePost.url,
		&p.rawCreatePost.title,
		&p.rawCreatePost.author,
		&p.rawCreatePost.permalink,
		&p.rawCreatePost.score,
		&p.rawCreatePost.nsfw,
		&p.rawCreatePost.grayscale,
		&p.rawCreatePost.time,
		&p.rawCreatePost.width,
		&p.rawCreatePost.height,
		&p.rawCreatePost.sprocket,
		&p.rawCreatePost.lowUrl,
		&p.rawCreatePost.lowWidth,
		&p.rawCreatePost.lowHeight,
		&p.rawCreatePost.medUrl,
		&p.rawCreatePost.medWidth,
		&p.rawCreatePost.medHeight,
		&p.rawCreatePost.highUrl,
		&p.rawCreatePost.highWidth,
		&p.rawCreatePost.highHeight); err != nil {
		return nil, err
	}
	return &p, nil
}

func scanRowToRawPostCount(rows *sql.Rows) (*rawPost, int, error) {
	var p rawPost
	var count int
	if err := rows.Scan(
		&p.id,
		&p.rawCreatePost.url,
		&p.rawCreatePost.title,
		&p.rawCreatePost.author,
		&p.rawCreatePost.permalink,
		&p.rawCreatePost.score,
		&p.rawCreatePost.nsfw,
		&p.rawCreatePost.grayscale,
		&p.rawCreatePost.time,
		&p.rawCreatePost.width,
		&p.rawCreatePost.height,
		&p.rawCreatePost.sprocket,
		&p.rawCreatePost.lowUrl,
		&p.rawCreatePost.lowWidth,
		&p.rawCreatePost.lowHeight,
		&p.rawCreatePost.medUrl,
		&p.rawCreatePost.medWidth,
		&p.rawCreatePost.medHeight,
		&p.rawCreatePost.highUrl,
		&p.rawCreatePost.highWidth,
		&p.rawCreatePost.highHeight,
		&p.rawCreatePost.c1_hex,
		&p.rawCreatePost.c1_css,
		&p.rawCreatePost.c1_percent,
		&p.rawCreatePost.c2_hex,
		&p.rawCreatePost.c2_css,
		&p.rawCreatePost.c2_percent,
		&p.rawCreatePost.c3_hex,
		&p.rawCreatePost.c3_css,
		&p.rawCreatePost.c3_percent,
		&p.rawCreatePost.c4_hex,
		&p.rawCreatePost.c4_css,
		&p.rawCreatePost.c4_percent,
		&p.rawCreatePost.c5_hex,
		&p.rawCreatePost.c5_css,
		&p.rawCreatePost.c5_percent,
		&p.rawCreatePost.keywords,
		&count); err != nil {
		return nil, 0, err
	}
	return &p, count, nil
}

// Strip the `u/` prefix from author
// Modifies the post in place
func stripAuthorPrefix(post *analogdb.Post) {
	post.Author = strings.TrimPrefix(post.Author, "u/")
}
