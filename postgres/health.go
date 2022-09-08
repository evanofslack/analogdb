package postgres

import (
	"context"

	"github.com/evanofslack/analogdb"
)

var _ analogdb.HealthService = (*HealthService)(nil)

type HealthService struct {
	db *DB
}

func NewHealthService(db *DB) *HealthService {
	return &HealthService{db: db}
}

func (s *HealthService) Readyz(ctx context.Context) error {
	return s.db.db.Ping()
}
