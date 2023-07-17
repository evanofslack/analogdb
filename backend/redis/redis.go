package redis

import (
	"context"

	"github.com/evanofslack/analogdb/logger"
	"github.com/go-redis/redis/v8"
)

type RDB struct {
	db     *redis.Client
	ctx    context.Context
	cancel func()
	logger *logger.Logger
}

func NewRDB(url string, logger *logger.Logger) (*RDB, error) {

	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	db := redis.NewClient(opt)

	ctx, cancel := context.WithCancel(context.Background())

	rdb := &RDB{
		db:     db,
		ctx:    ctx,
		cancel: cancel,
		logger: logger,
	}

	rdb.logger.Info().Msg("Initialized cache instance")

	return rdb, nil
}

func (rdb *RDB) Open() error {
	if err := rdb.db.Ping(rdb.ctx).Err(); err != nil {
		return err
	}
	return nil
}

func (rdb *RDB) Close() error {

	rdb.cancel()
	if rdb.db != nil {
		if err := rdb.db.Close(); err != nil {
			return err
		}
	}
	return nil
}
