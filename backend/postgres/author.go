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

	s.db.logger.Debug().Ctx(ctx).Msg("Starting find authors")
	defer s.db.logger.Debug().Ctx(ctx).Msg("Finished find authors")

	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	authors, err := findAuthors(ctx, tx)

	if err != nil {
		return nil, err
	}

	return authors, nil
}

func findAuthors(ctx context.Context, tx *sql.Tx) ([]string, error) {
	query := `
			SELECT id, author FROM pictures ORDER BY id ASC`
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	authors := make([]string, 0)
	var id int
	var author string
	for rows.Next() {
		if err := rows.Scan(&id, &author); err != nil {
			return nil, err
		}
		authors = append(authors, author)
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return authors, nil
}
