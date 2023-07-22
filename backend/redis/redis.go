package redis

import (
	"context"
	"time"

	"github.com/evanofslack/analogdb/logger"
	"github.com/evanofslack/analogdb/metrics"
	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"
)

type RDB struct {
	db      *redis.Client
	ctx     context.Context
	cancel  func()
	logger  *logger.Logger
	metrics *metrics.Metrics
}

// create a new redis database
func NewRDB(url string, logger *logger.Logger, metrics *metrics.Metrics) (*RDB, error) {

	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	db := redis.NewClient(opt)

	ctx, cancel := context.WithCancel(context.Background())

	rdb := &RDB{
		db:      db,
		ctx:     ctx,
		cancel:  cancel,
		logger:  logger,
		metrics: metrics,
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

type Cache struct {
	cache  *cache.Cache
	errors uint64
}

// create a new cache backed by redis
func (rdb *RDB) NewCache(instance string, size int, ttl time.Duration) *Cache {

	inner := cache.New(&cache.Options{
		Redis:      rdb.db,
		LocalCache: cache.NewTinyLFU(size, ttl),
	})

	cache := &Cache{
		cache:  inner,
		errors: 0,
	}

	rdb.logger.Info().Str("instance", instance).Msg("Created new cache")

	// start collecting metrics for this instance
	collector := newCacheCollector(cache, instance)
	rdb.metrics.Registry.MustRegister(collector)

	rdb.logger.Info().Str("instance", instance).Msg("Registered cache collector with prometheus")

	return cache
}

func (cache *Cache) stats() cacheStats {
	innerStats := cache.cache.Stats()
	stats := cacheStats{
		hits:   innerStats.Hits,
		misses: innerStats.Misses,
		errors: cache.errors,
	}
	return stats
}
