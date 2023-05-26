package postgres

import (
	"context"

	"github.com/evanofslack/analogdb"
)

var _ analogdb.ReadyService = (*ReadyService)(nil)

type ReadyService struct {
	db *DB
}

func NewReadyService(db *DB) *ReadyService {
	return &ReadyService{db: db}
}

func (s *ReadyService) Readyz(ctx context.Context) error {
	return s.db.db.Ping()
}
