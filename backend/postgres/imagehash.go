package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/evanofslack/analogdb"
	"github.com/lib/pq"
)

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
		p.URL = imageURL(s.db.imageBase, p.URL)
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

func (s *ScrapeService) FindImageMatch(ctx context.Context, hash int64, grayscale bool, maxBits int) (int, bool, error) {
	query := `
		SELECT id FROM pictures
		WHERE image_hash IS NOT NULL AND greyscale = $2
			AND bit_count((image_hash # $1)::bit(64)) <= $3
		ORDER BY time ASC, id ASC
		LIMIT 1`
	var id int
	err := s.db.db.QueryRowContext(ctx, query, hash, grayscale, maxBits).Scan(&id)
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
// The urls come from a second query, since selecting them in the pair query
// makes Postgres pick a plan more than ten times slower.
func (s *ScrapeService) DuplicatePosts(ctx context.Context, maxBits int) ([]analogdb.DuplicatePost, error) {
	s.db.logger.DebugContext(ctx, "Start find duplicate posts", "max_bits", maxBits)

	query := `
		SELECT DISTINCT ON (b.id)
			a.id, b.id, bit_count((a.image_hash # b.image_hash)::bit(64))
		FROM pictures a
		JOIN pictures b ON (a.time, a.id) < (b.time, b.id) AND a.greyscale = b.greyscale
		WHERE a.image_hash IS NOT NULL AND b.image_hash IS NOT NULL
			AND bit_count((a.image_hash # b.image_hash)::bit(64)) <= $1
		ORDER BY b.id, a.time, a.id`
	rows, err := s.db.db.QueryContext(ctx, query, maxBits)
	if err != nil {
		s.db.logger.ErrorContext(ctx, "Fail find duplicate posts", "error", err)
		return nil, err
	}
	defer rows.Close()

	dupes := make([]analogdb.DuplicatePost, 0)
	ids := make([]int64, 0)
	for rows.Next() {
		var d analogdb.DuplicatePost
		if err := rows.Scan(&d.KeepID, &d.DuplicateID, &d.Distance); err != nil {
			return nil, err
		}
		dupes = append(dupes, d)
		ids = append(ids, int64(d.KeepID), int64(d.DuplicateID))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(dupes) > 0 {
		urls, err := s.lowURLs(ctx, ids)
		if err != nil {
			s.db.logger.ErrorContext(ctx, "Fail get duplicate post urls", "error", err)
			return nil, err
		}
		for i := range dupes {
			dupes[i].KeepURL = urls[dupes[i].KeepID]
			dupes[i].DuplicateURL = urls[dupes[i].DuplicateID]
		}
	}
	s.db.logger.InfoContext(ctx, "Finish find duplicate posts", "count", len(dupes), "max_bits", maxBits)
	return dupes, nil
}

func (s *ScrapeService) lowURLs(ctx context.Context, ids []int64) (map[int]string, error) {
	rows, err := s.db.db.QueryContext(ctx, `SELECT id, lowurl FROM pictures WHERE id = ANY($1)`, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	urls := make(map[int]string, len(ids))
	for rows.Next() {
		var id int
		var url string
		if err := rows.Scan(&id, &url); err != nil {
			return nil, err
		}
		urls[id] = imageURL(s.db.imageBase, url)
	}
	return urls, rows.Err()
}
