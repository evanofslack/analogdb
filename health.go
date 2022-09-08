package analogdb

import "context"

type HealthService interface {
	Readyz(ctx context.Context) error
}
