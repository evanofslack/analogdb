package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/colorspace"
	"github.com/lib/pq"
)

// rederiveBatch is how many post ids one rederive transaction covers
const rederiveBatch = 500

// ensure interface is implemented
var _ analogdb.ColorService = (*ColorService)(nil)

type ColorService struct {
	db *DB
}

func NewColorService(db *DB) *ColorService {
	return &ColorService{db: db}
}

// RederiveColors recomputes the derived columns of every color and post from
// the stored hex and percent values, one transaction per range of post ids
func (s *ColorService) RederiveColors(ctx context.Context) (*analogdb.ColorRederive, error) {
	start := time.Now()
	var maxID int
	if err := s.db.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(id), 0) FROM pictures").Scan(&maxID); err != nil {
		return nil, err
	}
	result := &analogdb.ColorRederive{}
	for from := 0; from <= maxID; from += rederiveBatch {
		if err := s.db.rederiveColors(ctx, from, from+rederiveBatch, result); err != nil {
			s.db.logger.ErrorContext(ctx, "Fail rederive colors", "from_id", from, "error", err)
			return nil, err
		}
	}
	result.DurationMs = time.Since(start).Milliseconds()
	s.db.logger.InfoContext(ctx, "Finish rederive colors", "posts", result.Posts, "posts_updated", result.PostsUpdated, "colors", result.Colors, "colors_updated", result.ColorsUpdated, "duration_ms", result.DurationMs)
	return result, nil
}

type rederivePost struct {
	grayscale bool
	derived   []colorspace.Derived
	percents  []float64
}

// rederiveColors rederives the posts with ids in [from, to)
func (db *DB) rederiveColors(ctx context.Context, from, to int, result *analogdb.ColorRederive) error {
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	posts, ids, err := rederivePosts(ctx, tx, from, to)
	if err != nil || len(ids) == 0 {
		return err
	}
	cols, err := rederiveClusters(ctx, tx, from, to, posts)
	if err != nil {
		return err
	}

	colorQuery := `
		UPDATE colors c SET html = v.html, css = v.css, oklab_l = v.l, oklab_a = v.a, oklab_b = v.b, chroma = v.chroma, hue = v.hue
		FROM unnest($1::int[], $2::text[], $3::text[], $4::real[], $5::real[], $6::real[], $7::real[], $8::real[])
			AS v(id, html, css, l, a, b, chroma, hue)
		WHERE c.id = v.id
			AND (c.html, c.css, c.oklab_l, c.oklab_a, c.oklab_b, c.chroma, c.hue)
				IS DISTINCT FROM (v.html, v.css, v.l, v.a, v.b, v.chroma, v.hue)`
	res, err := tx.ExecContext(ctx, colorQuery, pq.Array(cols.ids), pq.Array(cols.htmls), pq.Array(cols.csss),
		pq.Array(cols.ls), pq.Array(cols.as), pq.Array(cols.bs), pq.Array(cols.chromas), pq.Array(cols.hues))
	if err != nil {
		return err
	}
	colorsUpdated, err := res.RowsAffected()
	if err != nil {
		return err
	}

	hueCol, chromaCol, strengthCol, keyCol := make([]sql.NullFloat64, len(ids)), make([]sql.NullFloat64, len(ids)), make([]sql.NullFloat64, len(ids)), make([]sql.NullFloat64, len(ids))
	for i, id := range ids {
		p := posts[id]
		c := newColorColumns(id, p.derived, p.percents, p.grayscale)
		hueCol[i], chromaCol[i], strengthCol[i], keyCol[i] = c.hue, c.chroma, c.strength, c.key
	}
	postQuery := `
		UPDATE pictures p SET color_hue = v.hue, color_chroma = v.chroma, color_strength = v.strength, rainbow_key = v.key
		FROM unnest($1::int[], $2::real[], $3::real[], $4::real[], $5::float8[]) AS v(id, hue, chroma, strength, key)
		WHERE p.id = v.id
			AND (p.color_hue, p.color_chroma, p.color_strength, p.rainbow_key)
				IS DISTINCT FROM (v.hue, v.chroma, v.strength, v.key)`
	res, err = tx.ExecContext(ctx, postQuery, pq.Array(ids), pq.Array(hueCol), pq.Array(chromaCol), pq.Array(strengthCol), pq.Array(keyCol))
	if err != nil {
		return err
	}
	postsUpdated, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	result.Posts += len(ids)
	result.PostsUpdated += int(postsUpdated)
	result.Colors += len(cols.ids)
	result.ColorsUpdated += int(colorsUpdated)
	return nil
}

// rederivePosts locks the posts with ids in [from, to)
func rederivePosts(ctx context.Context, tx *sql.Tx, from, to int) (map[int]*rederivePost, []int, error) {
	rows, err := tx.QueryContext(ctx, "SELECT id, greyscale FROM pictures WHERE id >= $1 AND id < $2 ORDER BY id FOR UPDATE", from, to)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	posts := map[int]*rederivePost{}
	ids := []int{}
	for rows.Next() {
		var id int
		p := &rederivePost{}
		if err := rows.Scan(&id, &p.grayscale); err != nil {
			return nil, nil, err
		}
		posts[id] = p
		ids = append(ids, id)
	}
	return posts, ids, rows.Err()
}

// rederivedColors holds the derived columns of colors, one entry per color
type rederivedColors struct {
	ids                       []int
	htmls, csss               []string
	ls, as, bs, chromas, hues []float64
}

// rederiveClusters derives the colors of the posts and adds them to each post.
// Colors with an invalid hex are left alone.
func rederiveClusters(ctx context.Context, tx *sql.Tx, from, to int, posts map[int]*rederivePost) (*rederivedColors, error) {
	rows, err := tx.QueryContext(ctx, "SELECT id, post_id, hex, percent FROM colors WHERE post_id >= $1 AND post_id < $2 ORDER BY post_id, percent DESC, id", from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols := &rederivedColors{}
	for rows.Next() {
		var id, postID int
		var hex string
		var percent float64
		if err := rows.Scan(&id, &postID, &hex, &percent); err != nil {
			return nil, err
		}
		p, ok := posts[postID]
		if !ok {
			continue
		}
		d, err := colorspace.Derive(hex)
		if err != nil {
			continue
		}
		p.derived = append(p.derived, d)
		p.percents = append(p.percents, percent)
		cols.ids = append(cols.ids, id)
		cols.htmls, cols.csss = append(cols.htmls, d.Html), append(cols.csss, d.Css)
		cols.ls, cols.as, cols.bs = append(cols.ls, d.L), append(cols.as, d.A), append(cols.bs, d.B)
		cols.chromas, cols.hues = append(cols.chromas, d.Chroma), append(cols.hues, d.Hue)
	}
	return cols, rows.Err()
}
