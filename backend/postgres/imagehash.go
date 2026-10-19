package postgres

import (
	"context"
	"database/sql"
	"slices"

	"github.com/evanofslack/analogdb"
	"github.com/lib/pq"
)

// FindImageMatch returns the oldest post the new image duplicates. A hash
// with many near neighbours is a plain image, and matches nothing.
func (s *ScrapeService) FindImageMatch(ctx context.Context, q analogdb.ImageMatchQuery) (analogdb.ImageMatch, bool, error) {
	query := `
		SELECT id, author, width, height, bit_count((image_hash # $1)::bit(64))
		FROM pictures
		WHERE image_hash IS NOT NULL AND greyscale = $2
			AND bit_count((image_hash # $1)::bit(64)) <= $3
		ORDER BY time ASC, id ASC`
	rows, err := s.db.db.QueryContext(ctx, query, q.Hash, q.Grayscale, analogdb.DuplicateCrowdBits)
	if err != nil {
		return analogdb.ImageMatch{}, false, err
	}
	defer rows.Close()

	type candidate struct {
		facts    analogdb.ImageFacts
		distance int
	}
	candidates := make([]candidate, 0)
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.facts.ID, &c.facts.Author, &c.facts.Width, &c.facts.Height, &c.distance); err != nil {
			return analogdb.ImageMatch{}, false, err
		}
		candidates = append(candidates, c)
	}
	if err := rows.Err(); err != nil {
		return analogdb.ImageMatch{}, false, err
	}
	if len(candidates) >= analogdb.DuplicateCrowdSize {
		s.db.logger.DebugContext(ctx, "Skip image match on crowded hash", "neighbours", len(candidates))
		return analogdb.ImageMatch{}, false, nil
	}

	post := analogdb.ImageFacts{Author: q.Author, Width: q.Width, Height: q.Height}
	for _, c := range candidates {
		if analogdb.IsDuplicateImage(c.facts, post, c.distance) {
			return analogdb.ImageMatch{ID: c.facts.ID, Distance: c.distance, SameAuthor: c.facts.Author == post.Author}, true, nil
		}
	}
	return analogdb.ImageMatch{}, false, nil
}

type nearPair struct {
	keep, dup, distance int
}

// DuplicatePosts compares every pair of hashed posts, which takes seconds
// over the whole archive. Each post is kept against its oldest match under
// the duplicate rule, so a group of copies all point at the original.
func (s *ScrapeService) DuplicatePosts(ctx context.Context) ([]analogdb.DuplicatePost, error) {
	s.db.logger.DebugContext(ctx, "Start find duplicate posts")

	pairs, err := s.nearPairs(ctx)
	if err != nil {
		s.db.logger.ErrorContext(ctx, "Fail find duplicate posts", "error", err)
		return nil, err
	}
	neighbours := make(map[int]int)
	ids := make([]int64, 0, len(pairs)*2)
	for _, p := range pairs {
		neighbours[p.keep]++
		neighbours[p.dup]++
		ids = append(ids, int64(p.keep), int64(p.dup))
	}
	posts, err := s.postFacts(ctx, ids)
	if err != nil {
		s.db.logger.ErrorContext(ctx, "Fail get duplicate post facts", "error", err)
		return nil, err
	}

	// pairs come oldest keep first for each duplicate, so the first match wins
	dupes := make([]analogdb.DuplicatePost, 0)
	matched := make(map[int]bool)
	for _, p := range pairs {
		if matched[p.dup] || neighbours[p.keep] >= analogdb.DuplicateCrowdSize {
			continue
		}
		keep, dup := posts[p.keep], posts[p.dup]
		if !analogdb.IsDuplicateImage(keep.ImageFacts, dup.ImageFacts, p.distance) {
			continue
		}
		matched[p.dup] = true
		dupes = append(dupes, analogdb.DuplicatePost{
			KeepID:       p.keep,
			DuplicateID:  p.dup,
			Distance:     p.distance,
			SameAuthor:   keep.Author == dup.Author,
			KeepURL:      keep.url,
			DuplicateURL: dup.url,
		})
	}
	slices.SortFunc(dupes, func(a, b analogdb.DuplicatePost) int { return a.DuplicateID - b.DuplicateID })
	s.db.logger.InfoContext(ctx, "Finish find duplicate posts", "count", len(dupes), "near_pairs", len(pairs))
	return dupes, nil
}

// nearPairs lists every pair of posts within the crowd distance, older post
// first. Postgres picks an index nested loop for this join that runs over
// ten times slower than a parallel hash join, so nested loops are turned off.
func (s *ScrapeService) nearPairs(ctx context.Context) ([]nearPair, error) {
	tx, err := s.db.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SET LOCAL enable_nestloop = off`); err != nil {
		return nil, err
	}

	query := `
		SELECT a.id, b.id, bit_count((a.image_hash # b.image_hash)::bit(64))
		FROM pictures a
		JOIN pictures b ON (a.time, a.id) < (b.time, b.id) AND a.greyscale = b.greyscale
		WHERE a.image_hash IS NOT NULL AND b.image_hash IS NOT NULL
			AND bit_count((a.image_hash # b.image_hash)::bit(64)) <= $1
		ORDER BY b.id, a.time, a.id`
	rows, err := tx.QueryContext(ctx, query, analogdb.DuplicateCrowdBits)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pairs := make([]nearPair, 0)
	for rows.Next() {
		var p nearPair
		if err := rows.Scan(&p.keep, &p.dup, &p.distance); err != nil {
			return nil, err
		}
		pairs = append(pairs, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return pairs, tx.Commit()
}

type postFact struct {
	analogdb.ImageFacts
	url string
}

func (s *ScrapeService) postFacts(ctx context.Context, ids []int64) (map[int]postFact, error) {
	rows, err := s.db.db.QueryContext(ctx, `SELECT id, author, width, height, lowurl FROM pictures WHERE id = ANY($1)`, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := make(map[int]postFact, len(ids))
	for rows.Next() {
		var p postFact
		if err := rows.Scan(&p.ID, &p.Author, &p.Width, &p.Height, &p.url); err != nil {
			return nil, err
		}
		p.url = imageURL(s.db.imageBase, p.url)
		posts[p.ID] = p
	}
	return posts, rows.Err()
}

// ReportDuplicates files a duplicate report on each duplicate post, skipping
// posts that already have an open one, and returns how many were filed
func (s *ScrapeService) ReportDuplicates(ctx context.Context, dupes []analogdb.DuplicatePost) (int, error) {
	if len(dupes) == 0 {
		return 0, nil
	}
	ids := make([]int64, 0, len(dupes))
	keeps := make([]int64, 0, len(dupes))
	messages := make([]string, 0, len(dupes))
	for _, d := range dupes {
		ids = append(ids, int64(d.DuplicateID))
		keeps = append(keeps, int64(d.KeepID))
		messages = append(messages, analogdb.DuplicateReportMessage(d))
	}

	query := `
		INSERT INTO post_reports (post_id, reason, message, duplicate_of)
		SELECT v.id, $3, v.message, v.keep
		FROM unnest($1::int[], $2::text[], $4::int[]) AS v(id, message, keep)
		WHERE EXISTS (SELECT 1 FROM pictures p WHERE p.id = v.id)
			AND NOT EXISTS (
				SELECT 1 FROM post_reports r
				WHERE r.post_id = v.id AND r.reason = $3 AND r.resolved_at IS NULL
			)`
	result, err := s.db.db.ExecContext(ctx, query, pq.Array(ids), pq.Array(messages), string(analogdb.ReportDuplicate), pq.Array(keeps))
	if err != nil {
		s.db.logger.ErrorContext(ctx, "Fail report duplicate posts", "error", err)
		return 0, err
	}
	reported, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	s.db.logger.InfoContext(ctx, "Reported duplicate posts", "count", reported, "duplicates", len(dupes))
	return int(reported), nil
}
