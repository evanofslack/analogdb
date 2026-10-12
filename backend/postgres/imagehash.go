package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/evanofslack/analogdb"
	"github.com/lib/pq"
)

// imageMatchMaxBits is how many of the 64 hash bits two images may differ by
// and still count as the same photo. Reposts measure 0, different photos 19+.
const imageMatchMaxBits = 4

func (s *ScrapeService) ImageHashMissingPosts(ctx context.Context) ([]analogdb.ImageHashMissing, error) {
	s.db.logger.DebugContext(ctx, "Start get image hash missing posts")
	defer s.db.logger.DebugContext(ctx, "Finish get image hash missing posts")

	rows, err := s.db.db.QueryContext(ctx, `SELECT id, lowurl FROM pictures WHERE image_hash IS NULL ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := make([]analogdb.ImageHashMissing, 0)
	for rows.Next() {
		var p analogdb.ImageHashMissing
		if err := rows.Scan(&p.ID, &p.URL); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

// SetImageHashes skips touchPost and post_updates, since a backfill over the
// archive is not a change to the posts
func (s *ScrapeService) SetImageHashes(ctx context.Context, hashes []analogdb.ImageHash) (int, error) {
	s.db.logger.DebugContext(ctx, "Start set image hashes", "count", len(hashes))

	ids := make([]int64, 0, len(hashes))
	values := make([]int64, 0, len(hashes))
	for _, h := range hashes {
		value, err := analogdb.ParseImageHash(h.Hash)
		if err != nil {
			return 0, err
		}
		ids = append(ids, int64(h.ID))
		values = append(values, value)
	}

	query := `
		UPDATE pictures p SET image_hash = v.hash
		FROM unnest($1::int[], $2::bigint[]) AS v(id, hash)
		WHERE p.id = v.id`
	result, err := s.db.db.ExecContext(ctx, query, pq.Array(ids), pq.Array(values))
	if err != nil {
		s.db.logger.ErrorContext(ctx, "Fail set image hashes", "error", err)
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	s.db.logger.InfoContext(ctx, "Finish set image hashes", "count", affected)
	return int(affected), nil
}

func (s *ScrapeService) FindImageMatch(ctx context.Context, hash int64, grayscale bool) (int, bool, error) {
	query := `
		SELECT id FROM pictures
		WHERE image_hash IS NOT NULL AND greyscale = $2
			AND bit_count((image_hash # $1)::bit(64)) <= $3
		ORDER BY time ASC, id ASC
		LIMIT 1`
	var id int
	err := s.db.db.QueryRowContext(ctx, query, hash, grayscale, imageMatchMaxBits).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

// DuplicatePosts compares every pair of hashed posts, which takes seconds
// over the whole archive. Each post with an older match is a duplicate, kept
// against its oldest match, so a group of copies all point at the original.
func (s *ScrapeService) DuplicatePosts(ctx context.Context) ([]analogdb.DuplicatePost, error) {
	s.db.logger.DebugContext(ctx, "Start find duplicate posts")

	query := `
		SELECT DISTINCT ON (b.id)
			a.id, b.id, bit_count((a.image_hash # b.image_hash)::bit(64)), a.lowurl, b.lowurl
		FROM pictures a
		JOIN pictures b ON (a.time, a.id) < (b.time, b.id) AND a.greyscale = b.greyscale
		WHERE a.image_hash IS NOT NULL AND b.image_hash IS NOT NULL
			AND bit_count((a.image_hash # b.image_hash)::bit(64)) <= $1
		ORDER BY b.id, a.time, a.id`
	rows, err := s.db.db.QueryContext(ctx, query, imageMatchMaxBits)
	if err != nil {
		s.db.logger.ErrorContext(ctx, "Fail find duplicate posts", "error", err)
		return nil, err
	}
	defer rows.Close()

	dupes := make([]analogdb.DuplicatePost, 0)
	for rows.Next() {
		var d analogdb.DuplicatePost
		if err := rows.Scan(&d.KeepID, &d.DuplicateID, &d.Distance, &d.KeepURL, &d.DuplicateURL); err != nil {
			return nil, err
		}
		dupes = append(dupes, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	s.db.logger.InfoContext(ctx, "Finish find duplicate posts", "count", len(dupes))
	return dupes, nil
}
