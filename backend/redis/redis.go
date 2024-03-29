package redis

import (
	"context"
	"strings"
	"time"

	"github.com/evanofslack/analogdb/logger"
	"github.com/evanofslack/analogdb/metrics"
	"github.com/go-redis/cache/v9"
	"github.com/redis/go-redis/v9"

	"github.com/redis/go-redis/extra/redisotel/v9"
)

const (
	cacheMissErr    = "cache: key is missing"
	decodeArrayErr1 = "msgpack: invalid code=8c decoding array length"
	decodeArrayErr2 = "msgpack: number of fields in array-encoded struct has changed"
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
func NewRDB(url string, logger *logger.Logger, metrics *metrics.Metrics, tracingEnabled bool) (*RDB, error) {

	logger.Debug().Msg("Initializing cache instance")

	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	db := redis.NewClient(opt)
	logger.Debug().Msg("Created new redis client")

	// prometheus metrics for redis
	redisCollector := newRedisCollector(db)
	metrics.Registry.MustRegister(redisCollector)
	logger.Info().Msg("Registered redis collector with prometheus")

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

	// prometheus metrics for redis based caches
	rdb.metrics.Registry.MustRegister(rdb.collector)
	rdb.logger.Info().Msg("Registered cache collector with prometheus")

	// otel instrumentation of redis
	if tracingEnabled {
		if err := redisotel.InstrumentTracing(db); err != nil {
			rdb.logger.Error().Err(err).Msg("Failed to instrument redis with tracing")
		} else {
			rdb.logger.Info().Msg("Instrumented redis with tracing")
		}
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

	cache.logger.Debug().Ctx(ctx).Str("instance", cache.instance).Msg("Getting item from cache")

	// do the lookup on the inner cache
	err := cache.cache.Get(ctx, key, item)

	// we got an error
	if err != nil {

		// was it a cache miss?
		if strings.Contains(err.Error(), cacheMissErr) {
			cache.logger.Debug().Ctx(ctx).Str("instance", cache.instance).Msg("Cache miss")
			cache.stats.incMisses()

			// temporarily downlevel this error
		} else if strings.Contains(err.Error(), decodeArrayErr1) || strings.Contains(err.Error(), decodeArrayErr2) {
			cache.logger.Warn().Ctx(ctx).Str("instance", cache.instance).Msg(err.Error())
			cache.stats.incErrors()

			// or an actual error
		} else {
			cache.logger.Error().Err(err).Ctx(ctx).Str("instance", cache.instance).Msg("Error getting item from cache")
			cache.stats.incErrors()
		}
		return err
	}

	// no error means cache hit
	cache.logger.Debug().Ctx(ctx).Str("instance", cache.instance).Msg("Cache hit")
	cache.stats.incHits()
	return nil
}

func (cache *Cache) set(ctx context.Context, item *cache.Item) error {

	cache.logger.Debug().Ctx(ctx).Str("instance", cache.instance).Msg("Setting item in cache")

	err := cache.cache.Set(item)
	if err != nil {
		cache.logger.Error().Err(err).Ctx(ctx).Str("instance", cache.instance).Msg("Failed to set item")
	}

	cache.logger.Debug().Ctx(ctx).Str("instance", cache.instance).Msg("Added item cache")
	return err
}

func (cache *Cache) delete(ctx context.Context, key string) error {

	cache.logger.Debug().Ctx(ctx).Str("instance", cache.instance).Msg("Deleting item from cache")

	err := cache.cache.Delete(ctx, key)
	if err != nil {
		cache.logger.Error().Err(err).Ctx(ctx).Str("instance", cache.instance).Msg("Failed to delete item")
	}

	cache.logger.Debug().Ctx(ctx).Str("instance", cache.instance).Msg("Deleted item from cache")
	return err
}
