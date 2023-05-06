package analogdb

import "context"

type AuthorService interface {
	FindAuthors(ctx context.Context) ([]string, error)
}
