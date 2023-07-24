package redis

import (
	"context"
	"strings"
	"time"

	"github.com/evanofslack/analogdb/logger"
	"github.com/evanofslack/analogdb/metrics"
	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"
)

const (
	cacheMissErr = "cache: key is missing"
)

type RDB struct {
	db        *redis.Client
	ctx       context.Context
	cancel    func()
	logger    *logger.Logger
	metrics   *metrics.Metrics
	collector *cacheCollector
}

// create a new redis database
func NewRDB(url string, logger *logger.Logger, metrics *metrics.Metrics) (*RDB, error) {

	logger.Debug().Msg("Initializing cache instance")

	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	db := redis.NewClient(opt)
	logger.Debug().Msg("Created new redis client")

	ctx, cancel := context.WithCancel(context.Background())

	collector := newCacheCollector()

	rdb := &RDB{
		db:        db,
		ctx:       ctx,
		cancel:    cancel,
		logger:    logger,
		metrics:   metrics,
		collector: collector,
	}

	rdb.metrics.Registry.MustRegister(rdb.collector)
	rdb.logger.Info().Msg("Registered cache collector with prometheus")

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

	rdb.logger.Debug().Msg("Starting redis server close")
	defer rdb.logger.Info().Msg("Closed redis server")

	rdb.cancel()
	if rdb.db != nil {
		if err := rdb.db.Close(); err != nil {
			return err
		}
	}
	return nil
}

type Cache struct {
	cache    *cache.Cache
	instance string
	stats    *cacheStats
	logger   *logger.Logger
}

// create a new cache backed by redis
func (rdb *RDB) NewCache(instance string, size int, ttl time.Duration) *Cache {

	rdb.logger.Debug().Str("instance", instance).Msg("Initializing new cache")

	inner := cache.New(&cache.Options{
		Redis:        rdb.db,
		LocalCache:   cache.NewTinyLFU(size, ttl),
		StatsEnabled: true,
	})

	stats := newCacheStats()

	cache := &Cache{
		cache:    inner,
		instance: instance,
		stats:    stats,
		logger:   rdb.logger,
	}

	// register this cache instance with the collector
	rdb.collector.registerCache(cache)
	rdb.logger.Info().Str("instance", instance).Msg("Registered cache instance with prometheus")

	rdb.logger.Info().Str("instance", instance).Msg("Initialized new cache")

	return cache
}

func (cache *Cache) get(ctx context.Context, key string, item interface{}) error {

	cache.logger.Debug().Str("instance", cache.instance).Msg("Getting item from cache")

	// do the lookup on the inner cache
	err := cache.cache.Get(ctx, key, item)

	// we got an error
	if err != nil {

		// was it a cache miss?
		if strings.Contains(err.Error(), cacheMissErr) {
			cache.logger.Debug().Str("instance", cache.instance).Msg("Cache miss")
			cache.stats.incMisses()
			// or an actual error
		} else {
			cache.logger.Err(err).Str("instance", cache.instance).Msg("Error getting item from cache")
			cache.stats.incErrors()
		}
		return err
	}

	// no error means cache hit
	cache.logger.Debug().Str("instance", cache.instance).Msg("Cache hit")
	cache.stats.incHits()
	return nil
}

func (cache *Cache) set(item *cache.Item) error {

	cache.logger.Debug().Str("instance", cache.instance).Msg("Setting item in cache")

	err := cache.cache.Set(item)
	if err != nil {
		cache.logger.Err(err).Str("instance", cache.instance).Msg("Failed to set item")
	}

	cache.logger.Debug().Str("instance", cache.instance).Msg("Added item cache")
	return err
}

func (cache *Cache) delete(ctx context.Context, key string) error {

	cache.logger.Debug().Str("instance", cache.instance).Msg("Deleting item from cache")

	err := cache.cache.Delete(ctx, key)
	if err != nil {
		cache.logger.Err(err).Str("instance", cache.instance).Msg("Failed to delete item")
	}

	cache.logger.Debug().Str("instance", cache.instance).Msg("Deleted item from cache")
	return err
}
