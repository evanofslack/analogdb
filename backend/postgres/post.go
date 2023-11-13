package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	goTime "time"

	"github.com/evanofslack/analogdb"
)

// ensure interface is implemented
var _ analogdb.PostService = (*PostService)(nil)

// rawPostCreate corresponds to the columns as a post is inserted in DB
type rawCreatePost struct {
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
	hexes      NullString
	csses      NullString
	htmls      NullString
	percents   NullString
	words      NullString
	weights    NullString
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

func (s *PostService) CreatePost(ctx context.Context, post *analogdb.CreatePost) (*analogdb.Post, error) {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	createdPost, err := s.db.createPost(ctx, tx, post)
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
	return s.db.findPosts(ctx, tx, filter)
}

func (s *PostService) FindPostByID(ctx context.Context, id int) (*analogdb.Post, error) {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	ids := []int{id}
	filter := analogdb.NewPostFilterWithIDs(ids)
	posts, _, err := s.db.findPosts(ctx, tx, filter)
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
	err = s.db.patchPost(ctx, tx, patch, id)
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
	err = s.db.deletePost(ctx, tx, id)
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
	ids, err := s.db.allPostIDs(ctx, tx)

	if err != nil {
		return nil, err
	}

	return ids, nil
}

// insertPost inserts a post into the DB and returns the post's ID
func (db *DB) insertPost(ctx context.Context, tx *sql.Tx, post *analogdb.CreatePost) (*int64, error) {

	db.logger.Debug().Ctx(ctx).Msg("Starting insert post")

	create, err := createPostToRawPostCreate(post)
	if err != nil {
		db.logger.Error().Ctx(ctx).Err(err).Msg("Failed to insert post")
		return nil, err
	}

	var id int64

	query :=
		`
	INSERT INTO pictures
	(url, title, author, permalink, score, nsfw, greyscale, time, width, height, sprocket, lowUrl, lowWidth, lowHeight, medUrl, medWidth, medHeight, highUrl, highWidth, highHeight)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	ON CONFLICT (permalink) DO NOTHING
	RETURNING id
	`

	stmt, err := tx.PrepareContext(ctx, query)

	if err != nil {
		db.logger.Error().Ctx(ctx).Err(err).Int64("postID", id).Msg("Failed to insert post")
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
		create.highHeight).Scan(&id)

	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int64("postID", id).Msg("Failed to insert post")
		return nil, err
	}

	db.logger.Info().Ctx(ctx).Int64("postID", id).Msg("Finished inserting post")

	return &id, nil
}

// insertKeywords inserts a post's keywords into the DB
func (db *DB) insertKeywords(ctx context.Context, tx *sql.Tx, keywords []analogdb.Keyword, postID int64) error {

	db.logger.Debug().Ctx(ctx).Int64("postID", postID).Msg("Starting insert keywords")

	first := 1
	second := 2
	third := 3

	vals := []any{}
	inserts := []string{}

	query :=
		`
	INSERT INTO keywords
	(word, weight, post_id)
	VALUES `

	for _, kw := range keywords {
		inserts = append(inserts, fmt.Sprintf("($%d, $%d, $%d)", first, second, third))
		vals = append(vals, kw.Word, kw.Weight, postID)
		first += 3
		second += 3
		third += 3
	}

	query += strings.Join(inserts, ",")
	stmt, err := tx.PrepareContext(ctx, query)

	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int64("postID", postID).Msg("Failed to insert keywords")
		return err
	}

	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, vals...)

	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int64("postID", postID).Msg("Failed to insert keywords")
		return err
	}

	db.logger.Info().Ctx(ctx).Int64("postID", postID).Msg("Finished inserting keywords")

	return nil
}

// deleteKeywords deletes all keywords for a given post
func (db *DB) deleteKeywords(ctx context.Context, tx *sql.Tx, postID int64) error {

	db.logger.Debug().Ctx(ctx).Int64("postID", postID).Msg("Starting delete keywords")

	query :=
		"DELETE FROM keywords WHERE post_id = $1"

	rows, err := tx.QueryContext(ctx, query, postID)
	defer rows.Close()
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int64("postID", postID).Msg("Failed to delete keywords")
		return err
	}

	db.logger.Info().Ctx(ctx).Int64("postID", postID).Msg("Finished deleting keywords")
	return nil
}

// insertKeywords inserts a post's keywords into the DB
func (db *DB) insertColors(ctx context.Context, tx *sql.Tx, colors []analogdb.Color, postID int64) error {

	db.logger.Debug().Ctx(ctx).Int64("postID", postID).Msg("Starting insert colors")

	first := 1
	second := 2
	third := 3
	fourth := 4
	fifth := 5

	vals := []any{}
	inserts := []string{}

	query :=
		`
	INSERT INTO colors
	(hex, css, html, percent, post_id)
	VALUES `

	for _, c := range colors {
		inserts = append(inserts, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d)", first, second, third, fourth, fifth))
		vals = append(vals, c.Hex, c.Css, c.Html, c.Percent, postID)
		first += 5
		second += 5
		third += 5
		fourth += 5
		fifth += 5
	}

	query += strings.Join(inserts, ",")
	stmt, err := tx.PrepareContext(ctx, query)

	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int64("postID", postID).Msg("Failed to insert colors")
		return err
	}

	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, vals...)

	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int64("postID", postID).Msg("Failed to insert colors")
		return err
	}

	db.logger.Info().Ctx(ctx).Int64("postID", postID).Msg("Finished inserting colors")

	return nil
}

// deleteKeywords deletes all keywords for a given post
func (db *DB) deleteColors(ctx context.Context, tx *sql.Tx, postID int64) error {

	db.logger.Debug().Ctx(ctx).Int64("postID", postID).Msg("Starting delete colors")

	query :=
		"DELETE FROM colors WHERE post_id = $1"

	rows, err := tx.QueryContext(ctx, query, postID)
	defer rows.Close()
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int64("postID", postID).Msg("Failed to delete colors")
		return err
	}

	db.logger.Info().Ctx(ctx).Int64("postID", postID).Msg("Finished deleting colors")
	return nil
}

func (db *DB) createPost(ctx context.Context, tx *sql.Tx, post *analogdb.CreatePost) (*analogdb.Post, error) {

	db.logger.Debug().Ctx(ctx).Msg("Starting create post")

	id, err := db.insertPost(ctx, tx, post)
	if err != nil {
		return nil, err
	}

	// insert keywords if they are provided
	if len(post.Keywords) != 0 {
		err = db.insertKeywords(ctx, tx, post.Keywords, *id)
		if err != nil {
			return nil, err
		}
	}

	// insert colors if they are provided
	if len(post.Colors) != 0 {
		err = db.insertColors(ctx, tx, post.Colors, *id)
		if err != nil {
			return nil, err
		}
	}

	// commit transaction if all inserts are ok
	err = tx.Commit()
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int64("postID", *id).Msg("Failed to create post")
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
		Keywords:  post.Keywords,
	}

	createdPost := &analogdb.Post{
		Id:          int(*id),
		DisplayPost: displayPost,
	}

	db.logger.Info().Ctx(ctx).Int64("postID", *id).Msg("Finished creating post")

	return createdPost, nil
}

// findPosts is the general function responsible for handling all queries
func (db *DB) findPosts(ctx context.Context, tx *sql.Tx, filter *analogdb.PostFilter) ([]*analogdb.Post, int, error) {

	db.logger.Debug().Ctx(ctx).Msg("Starting find posts")

	var colorArgs, keywordArgs, postArgs []any
	index := 1
	var colorWhere, keywordWhere, postWhere string

	colorWhere, colorArgs, index = filterToWhereColor(filter, index)
	keywordWhere, keywordArgs, index = filterToWhereKeyword(filter, index)
	postWhere, postArgs, index = filterToWherePost(filter, index)

	keywordJoin := "LEFT OUTER"
	if filter.Keywords != nil {
		keywordJoin = "INNER"
	}

	colorJoin := "LEFT OUTER"
	if filter.Colors != nil {
		colorJoin = "INNER"
	}

	args := append(colorArgs, keywordArgs...)
	args = append(args, postArgs...)

	order := filterToOrder(filter)
	limit := formatLimit(filter)
	query := fmt.Sprintf(`
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
				c.hexes,
				c.csses,
				c.htmls,
				c.percents,
				k.words,
				k.weights,
				COUNT(*) OVER()
			FROM
				pictures p
				%s JOIN (
					SELECT
						post_id,
						STRING_AGG(colors.hex, ',' ORDER BY colors.percent DESC) as hexes,
						STRING_AGG(colors.css, ',' ORDER BY colors.percent DESC) as csses,
						STRING_AGG(colors.html, ',' ORDER BY colors.percent DESC) as htmls,
						ARRAY_AGG(colors.percent ORDER BY colors.percent DESC) as percents
					FROM colors
					WHERE %s
					GROUP BY post_id
				) c on c.post_id = p.id
				%s JOIN (
					SELECT
						post_id,
						STRING_AGG(keywords.word, ',' ORDER BY keywords.weight DESC) as words,
						ARRAY_AGG(keywords.weight ORDER BY keywords.weight DESC) as weights
					FROM keywords
					WHERE %s
					GROUP BY post_id
				) k on k.post_id = p.id
			WHERE %s
	`, colorJoin, colorWhere, keywordJoin, keywordWhere, postWhere) + order + limit

	rows, err := tx.QueryContext(ctx, query, args...)

	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Msg("Failed to find posts")
		return nil, 0, err
	}
	defer rows.Close()

	posts := make([]*analogdb.Post, 0)
	var count int
	var p *rawPost
	for rows.Next() {
		p, count, err = scanRowToRawPostCount(rows)
		if err != nil {
			db.logger.Error().Err(err).Ctx(ctx).Msg("Failed to find posts")
			return nil, 0, err
		}
		post, err := rawPostToPost(*p)

		// strip `u/` prefix from author (modifies in place)
		stripAuthorPrefix(post)

		if err != nil {
			db.logger.Error().Err(err).Ctx(ctx).Msg("Failed to find posts")
			return nil, 0, err
		}
		posts = append(posts, post)

	}

	err = tx.Commit()
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Msg("Failed to find posts")
		return nil, 0, err
	}

	db.logger.Info().Ctx(ctx).Msg("Finished finding posts")

	return posts, count, nil
}

func (db *DB) patchPost(ctx context.Context, tx *sql.Tx, patch *analogdb.PatchPost, id int) error {

	db.logger.Debug().Ctx(ctx).Int("postID", id).Msg("Starting patch post")

	hasPatchFields := false

	// if the patch includes general updates for the post
	if patch.Nsfw != nil || patch.Sprocket != nil || patch.Grayscale != nil || patch.Score != nil {
		hasPatchFields = true
		if err := db.updatePostGeneral(ctx, tx, patch, id); err != nil {
			db.logger.Error().Err(err).Ctx(ctx).Int("postID", id).Msg("Failed to patch post")
			return err
		}
	}

	// if the patch includes updates for keywords
	if patch.Keywords != nil {
		hasPatchFields = true
		if err := db.updateKeywords(ctx, tx, *patch.Keywords, id); err != nil {
			db.logger.Error().Err(err).Ctx(ctx).Int("postID", id).Msg("Failed to patch post")
			return err
		}
	}

	// if the patch includes updates for colors
	if patch.Colors != nil {
		hasPatchFields = true
		if err := db.updateColors(ctx, tx, *patch.Colors, id); err != nil {
			db.logger.Error().Err(err).Ctx(ctx).Int("postID", id).Msg("Failed to patch post")
			return err
		}
	}

	if !hasPatchFields {
		err := errors.New("must include patch parameters")
		db.logger.Error().Err(err).Ctx(ctx).Int("postID", id).Msg("Failed to patch post")
		return err
	}

	// always insert the updated timestamp
	if err := db.insertPostUpdateTimes(ctx, tx, patch, id); err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int("postID", id).Msg("Failed to patch post")
		return err
	}

	err := tx.Commit()
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int("postID", id).Msg("Failed to patch post")
		return err
	}

	db.logger.Info().Ctx(ctx).Int("postID", id).Msg("Finished patching post")

	return nil
}

func (db *DB) updateKeywords(ctx context.Context, tx *sql.Tx, keywords []analogdb.Keyword, id int) error {

	db.logger.Debug().Ctx(ctx).Int("postID", id).Msg("Starting update keywords")

	// first delete all keywords associated with post
	if err := db.deleteKeywords(ctx, tx, int64(id)); err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int("postID", id).Msg("Failed to update keywords")
		return err
	}

	// if we have no keywords to insert, just return
	if len(keywords) == 0 {
		db.logger.Info().Ctx(ctx).Int("postID", id).Msg("Finished updating keywords (dropped all keywords)")
		return nil
	}

	// then insert all new keywords
	if err := db.insertKeywords(ctx, tx, keywords, int64(id)); err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int("postID", id).Msg("Failed to update keywords")
		return err
	}

	db.logger.Info().Ctx(ctx).Int("postID", id).Msg("Finished updating keywords")
	return nil
}

func (db *DB) updateColors(ctx context.Context, tx *sql.Tx, colors []analogdb.Color, id int) error {

	db.logger.Debug().Ctx(ctx).Int("postID", id).Msg("Starting update colors")

	// first delete all colors associated with post
	if err := db.deleteColors(ctx, tx, int64(id)); err != nil {
		return err
	}

	// if we have no colors to insert, just return
	if len(colors) == 0 {
		db.logger.Info().Ctx(ctx).Int("postID", id).Msg("Finished updating colors (dropped all colors)")
		return nil
	}

	// then insert all new colors
	if err := db.insertColors(ctx, tx, colors, int64(id)); err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int("postID", id).Msg("Failed to update colors")
		return err
	}

	db.logger.Info().Ctx(ctx).Int("postID", id).Msg("Finished updating colors")
	return nil
}

func (db *DB) updatePostGeneral(ctx context.Context, tx *sql.Tx, patch *analogdb.PatchPost, id int) error {

	db.logger.Debug().Ctx(ctx).Int("postID", id).Msg("Starting update post")

	set, args, err := patchToSet(patch)
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int("postID", id).Msg("Failed to update post")
		return err
	}

	args = append(args, id)
	idPos := len(args)

	query :=
		"UPDATE pictures " + set + fmt.Sprintf(" WHERE id =  $%d", idPos)

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int("postID", id).Msg("Failed to update post")
		return err
	}
	defer rows.Close()

	db.logger.Info().Ctx(ctx).Int("postID", id).Msg("Finished updating post")
	return nil
}

func (db *DB) insertPostUpdateTimes(ctx context.Context, tx *sql.Tx, patch *analogdb.PatchPost, id int) error {

	db.logger.Debug().Ctx(ctx).Int("postID", id).Msg("Starting post update times")

	query :=
		`
		INSERT INTO post_updates
	(post_id, score_update_time, nsfw_update_time, greyscale_update_time, sprocket_update_time, colors_update_time, keywords_update_time)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	stmt, err := tx.PrepareContext(ctx, query)

	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int("postID", id).Msg("Failed to post update times")
		return err
	}

	defer stmt.Close()

	// helper func to add update time if passed, else add null string
	addTimeOrNull := func(addTime bool, values *[]any) {
		now := goTime.Now().Unix()
		if addTime {
			*values = append(*values, now)
		} else {
			*values = append(*values, sql.NullInt64{})
		}

	}

	// we have to include post_id
	values := []any{id}

	score := patch.Score != nil
	addTimeOrNull(score, &values)
	nsfw := patch.Nsfw != nil
	addTimeOrNull(nsfw, &values)
	grayscale := patch.Grayscale != nil
	addTimeOrNull(grayscale, &values)
	sprocket := patch.Sprocket != nil
	addTimeOrNull(sprocket, &values)
	colors := patch.Colors != nil
	addTimeOrNull(colors, &values)
	keywords := patch.Keywords != nil
	addTimeOrNull(keywords, &values)

	rows, err := stmt.QueryContext(ctx, values...)
	defer rows.Close()
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int("postID", id).Msg("Failed to post update times")
		return err
	}

	db.logger.Info().Ctx(ctx).Int("postID", id).Msg("Finished post update times")

	return nil

}

func (db *DB) deletePost(ctx context.Context, tx *sql.Tx, id int) error {

	db.logger.Debug().Ctx(ctx).Int("postID", id).Msg("Starting delete post")

	query := `
			DELETE FROM pictures
			WHERE id = $1
			RETURNING id`

	row := tx.QueryRowContext(ctx, query, id)

	var returnedID int
	err := row.Scan(&returnedID)
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int("postID", id).Msg("Failed to delete post")
		return err
	}

	if id != returnedID {

		err := fmt.Errorf("error deleting post with id %d", id)
		db.logger.Error().Err(err).Ctx(ctx).Int("postID", id).Msg("Failed to delete post")
		return err
	}

	err = tx.Commit()
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Int("postID", id).Msg("Failed to delete post")
		return err
	}

	db.logger.Info().Ctx(ctx).Int("postID", id).Msg("Finished deleting post")
	return nil
}

func (db *DB) allPostIDs(ctx context.Context, tx *sql.Tx) ([]int, error) {

	db.logger.Debug().Ctx(ctx).Msg("Starting get all post IDs")

	query := `
			SELECT id FROM pictures ORDER BY id ASC`
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Msg("Failed to get all post IDs")
		return nil, err
	}
	defer rows.Close()

	ids := make([]int, 0)
	var id int
	for rows.Next() {
		if err := rows.Scan(&id); err != nil {
			db.logger.Error().Err(err).Ctx(ctx).Msg("Failed to get all post IDs")
			return nil, err
		}
		ids = append(ids, id)
	}
	err = tx.Commit()
	if err != nil {
		db.logger.Error().Err(err).Ctx(ctx).Msg("Failed to get all post IDs")
		return nil, err
	}

	db.logger.Info().Ctx(ctx).Msg("Finished getting all post IDs")
	return ids, nil
}

// filterToOrder converts filter into an SQL "ORDER BY" statement
func filterToOrder(filter *analogdb.PostFilter) string {
	if sort := filter.Sort; sort != nil {
		switch *sort {
		case analogdb.SortTime:
			return " ORDER BY p.time DESC"
		case analogdb.SortScore:
			return " ORDER BY p.score DESC"
		case analogdb.SortRandom:
			if filter.Seed == nil {
				filter.SetSeed()
			}
			return fmt.Sprintf(" ORDER BY MOD(p.time, %d), p.time DESC", *filter.Seed)
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

func filterToWhereColor(filter *analogdb.PostFilter, startIndex int) (string, []any, int) {

	index := startIndex
	base := "1=1"
	where, args := []string{base}, []any{}
	colorsP, colorPercentsP := filter.Colors, filter.ColorPercents

	if colorsP == nil || colorPercentsP == nil {
		return base, args, index
	}

	colors, colorPercents := *colorsP, *colorPercentsP

	// percents must not be shorter than colors
	for len(colors) > len(colorPercents) {
		colorPercents = append(colorPercents, 0.0)
	}

	// get all post ids matching colors.
	// group by html color and sum grouped percents.
	//
	// i.e.
	//
	// WHERE post_id IN (
	// 	SELECT post_id
	// 	FROM colors
	// 	WHERE html = 'red'
	// 	GROUP BY post_id, html
	// 	HAVING sum(percent) > 0.1
	// 	INTERSECT
	// 	SELECT post_id
	// 	FROM colors
	// 	WHERE html = 'black'
	// 	GROUP BY post_id, html
	// 	HAVING sum(percent) > 0.1
	// )

	inner := ""
	// must do one intersection for each color.
	for i := range colors {
		color, percent := colors[i], colorPercents[i]
		inner += fmt.Sprintf("SELECT post_id from colors WHERE html = $%d GROUP BY post_id, html HAVING sum(percent) > $%d INTERSECT ", index, index+1)
		index += 2
		args = append(args, color, percent)
	}

	// strip off the trailing intersect
	inner = strings.TrimSuffix(inner, " INTERSECT ")
	statement := fmt.Sprintf("post_id IN (%s)", inner)
	where = append(where, statement)

	whereQuery := strings.Join(where, " AND ")

	return whereQuery, args, index
}

func filterToWhereKeyword(filter *analogdb.PostFilter, startIndex int) (string, []any, int) {

	index := startIndex
	base := "1=1"
	where, args := []string{base}, []any{}

	if filter.Keywords == nil {
		return base, args, index
	}

	// get all post ids matching all keywords.
	//
	// i.e.
	//
	// WHERE post_id IN (
	// 	SELECT post_id
	// 	FROM keywords
	// 	WHERE word = '$word1'
	// 	INTERSECT
	// 	SELECT post_id
	// 	FROM keywords
	// 	WHERE word = '$word2'
	//  ...
	// )

	inner := ""
	// must do one intersection for each keyword.
	for _, keyword := range *filter.Keywords {
		inner += fmt.Sprintf("SELECT post_id from keywords WHERE word = $%d INTERSECT ", index)
		index += 1
		args = append(args, keyword)
	}

	// strip off the trailing intersect
	inner = strings.TrimSuffix(inner, " INTERSECT ")
	statement := fmt.Sprintf("post_id IN (%s)", inner)
	where = append(where, statement)

	whereQuery := strings.Join(where, " AND ")

	return whereQuery, args, index
}

// filterToWhere converts a PostFilter to an SQL WHERE statement
func filterToWherePost(filter *analogdb.PostFilter, startIndex int) (string, []any, int) {

	index := startIndex
	where, args := []string{"1=1"}, []any{}

	if sort, keyset := filter.Sort, filter.Keyset; sort != nil && keyset != nil {
		switch *sort {
		case analogdb.SortTime:
			where = append(where, fmt.Sprintf("p.time < $%d", index))
			args = append(args, *keyset)
			index += 1
		case analogdb.SortScore:
			where = append(where, fmt.Sprintf("p.score < $%d", index))
			args = append(args, *keyset)
			index += 1
		case analogdb.SortRandom:
			if seed := filter.Seed; seed != nil {
				where = append(where, fmt.Sprintf("MOD(p.time, $%d) > $%d", index, index+1))
				args = append(args, *seed, *keyset%*seed)
				index += 2
			}
		}
	}

	if nsfw := filter.Nsfw; nsfw != nil {
		where = append(where, fmt.Sprintf("p.nsfw = $%d", index))
		args = append(args, *nsfw)
		index += 1
	}

	if grayscale := filter.Grayscale; grayscale != nil {
		where = append(where, fmt.Sprintf("p.greyscale = $%d", index))
		args = append(args, *grayscale)
		index += 1
	}

	if sprocket := filter.Sprocket; sprocket != nil {
		where = append(where, fmt.Sprintf("p.sprocket = $%d", index))
		args = append(args, *sprocket)
		index += 1
	}

	if ids := filter.IDs; ids != nil {
		where = append(where, fmt.Sprintf("p.id = ANY($%d::int[])", index))
		// turn the slice of ids into a string i.e. "(1,2,3)"
		var idsFormat string
		if len(*ids) == 1 {
			// single id can't have a comma
			id := (*ids)[0]
			idsFormat = fmt.Sprintf("{%s}", strconv.Itoa(id))
		} else {
			idsString := []string{}
			for _, i := range *ids {
				idsString = append(idsString, strconv.Itoa(i))
			}
			idsFormat = "{" + strings.Join(idsString, ",") + "}"
		}
		args = append(args, idsFormat)
		index += 1
	}

	// match partial text in post title with ILIKE
	if title := filter.Title; title != nil {
		where = append(where, fmt.Sprintf("p.title ILIKE $%d", index))
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
		where = append(where, fmt.Sprintf("p.author = $%d", index))
		args = append(args, matchAuthor)
		index += 1
	}

	if minWidth := filter.Width.Min; minWidth != nil {
		where = append(where, fmt.Sprintf("p.width >= $%d", index))
		args = append(args, *minWidth)
		index += 1
	}

	if maxWidth := filter.Width.Max; maxWidth != nil {
		where = append(where, fmt.Sprintf("p.width <= $%d", index))
		args = append(args, *maxWidth)
		index += 1
	}

	if minHeight := filter.Height.Min; minHeight != nil {
		where = append(where, fmt.Sprintf("p.height >= $%d", index))
		args = append(args, *minHeight)
		index += 1
	}

	if maxHeight := filter.Height.Max; maxHeight != nil {
		where = append(where, fmt.Sprintf("p.height <= $%d", index))
		args = append(args, *maxHeight)
		index += 1
	}

	if minRatio := filter.AspectRatio.Min; minRatio != nil {
		where = append(where, fmt.Sprintf("p.width / p.height >= $%d", index))
		args = append(args, *minRatio)
		index += 1
	}

	if maxRatio := filter.AspectRatio.Max; maxRatio != nil {
		where = append(where, fmt.Sprintf("p.width / p.height <= $%d", index))
		args = append(args, *maxRatio)
		index += 1
	}

	whereQuery := strings.Join(where, " AND ")

	return whereQuery, args, index
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
		return nil, &analogdb.Error{Code: analogdb.ERRUNPROCESSABLE, Message: "Unable to create post, expected 5 colors"}
	}

	// we don't actually use these when creating the post here
	// keywords are handled with seperate function
	hexes := NullString{}
	csses := NullString{}
	htmls := NullString{}
	percents := NullString{}
	words := NullString{}
	weights := NullString{}

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
		hexes:      hexes,
		csses:      csses,
		htmls:      htmls,
		percents:   percents,
		words:      words,
		weights:    weights,
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

	// grab the colors
	var hexes, csses, htmls, percents []string
	var colors = []analogdb.Color{}

	// check for null
	if p.hexes.Valid {
		hexes = strings.Split(p.hexes.String, ",")
	}
	if p.csses.Valid {
		csses = strings.Split(p.csses.String, ",")
	}
	if p.htmls.Valid {
		htmls = strings.Split(p.htmls.String, ",")
	}
	if p.percents.Valid {
		// remove '{}' from postgres array then split on commas
		percents = strings.Split(strings.Trim(p.percents.String, "{}"), ",")

	}

	// iterate over shortest slice. should all be same length though
	var iter = hexes
	if len(csses) < len(iter) {
		iter = csses
	}
	if len(htmls) < len(iter) {
		iter = htmls
	}
	if len(percents) < len(iter) {
		iter = percents
	}

	for i := range iter {
		percent, err := strconv.ParseFloat(percents[i], 64)
		if err != nil {
			percent = 0.0
		}
		colors = append(colors, analogdb.Color{Hex: hexes[i], Css: csses[i], Html: htmls[i], Percent: percent})
	}

	// grab the keywords
	var words, weights []string
	var keywords = []analogdb.Keyword{}

	// check for null
	if p.words.Valid {
		words = strings.Split(p.words.String, ",")
	}
	if p.weights.Valid {
		// remove '{}' from postgres array then split on commas
		weights = strings.Split(strings.Trim(p.weights.String, "{}"), ",")

	}

	// iterate over keywords or percents, whichever is smaller
	// technically should both be the same size but we can't be sure
	iter = words
	if len(weights) < len(words) {
		iter = weights
	}

	for i := range iter {
		weight, err := strconv.ParseFloat(weights[i], 64)
		if err != nil {
			weight = 0.0
		}
		keywords = append(keywords, analogdb.Keyword{Word: words[i], Weight: weight})
	}

	post := &analogdb.Post{Id: p.id,
		DisplayPost: analogdb.DisplayPost{
			Title:     p.title,
			Author:    p.author,
			Permalink: p.permalink,
			Score:     p.score,
			Nsfw:      p.nsfw,
			Grayscale: p.grayscale,
			Time:      p.time,
			Sprocket:  p.sprocket,
			Images:    images,
			Colors:    colors,
			Keywords:  keywords}}
	return post, nil
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
		&p.rawCreatePost.hexes,
		&p.rawCreatePost.csses,
		&p.rawCreatePost.htmls,
		&p.rawCreatePost.percents,
		&p.rawCreatePost.words,
		&p.rawCreatePost.weights,
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
