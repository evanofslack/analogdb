package analogdb

import "context"

type ReadyService interface {
	Readyz(ctx context.Context) error
}
