package postgres

import (
	"context"
	"database/sql"

	"github.com/evanofslack/analogdb"
)

// ensure interface is implemented
var _ analogdb.AuthorService = (*AuthorService)(nil)

type AuthorService struct {
	db *DB
}

func NewAuthorService(db *DB) *AuthorService {
	return &AuthorService{db: db}
}

func (s *AuthorService) FindAuthors(ctx context.Context) ([]string, error) {
	s.db.logger.DebugContext(ctx, "Starting find authors")
	defer s.db.logger.DebugContext(ctx, "Finished find authors")

	authors, err := findAuthors(ctx, s.db.db)
	if err != nil {
		return nil, err
	}

	return authors, nil
}

func findAuthors(ctx context.Context, db *sql.DB) ([]string, error) {
	query := `
			SELECT DISTINCT author FROM pictures ORDER BY author`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	authors := make([]string, 0)
	var author string
	for rows.Next() {
		if err := rows.Scan(&author); err != nil {
			return nil, err
		}
		authors = append(authors, author)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return authors, nil
}
