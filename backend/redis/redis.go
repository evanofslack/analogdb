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
	logger.Debug("Initializing cache instance")

	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	db := redis.NewClient(opt)
	logger.Debug("Created new redis client")

	// prometheus metrics for redis
	redisCollector := newRedisCollector(db)
	metrics.Registry.MustRegister(redisCollector)
	logger.Info("Registered redis collector with prometheus")

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
	rdb.logger.Info("Registered cache collector with prometheus")

	// otel instrumentation of redis
	if tracingEnabled {
		if err := redisotel.InstrumentTracing(db); err != nil {
			rdb.logger.Error("Fail instrument redis with tracing", "error", err)
		} else {
			rdb.logger.Info("Instrumented redis with tracing")
		}
	}

	rdb.logger.Info("Initialized cache instance")

	return rdb, nil
}

func (rdb *RDB) Open() error {
	if err := rdb.db.Ping(rdb.ctx).Err(); err != nil {
		return err
	}
	return nil
}

func (rdb *RDB) Close() error {
	rdb.logger.Debug("Starting redis server close")
	defer rdb.logger.Info("Closed redis server")

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
	rdb.logger.Debug("Initializing new cache", "instance", instance)

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
	rdb.logger.Info("Registered cache instance with prometheus", "instance", instance)
	rdb.logger.Info("Initialized new cache", "instance", instance)

	return cache
}

func (cache *Cache) get(ctx context.Context, key string, item interface{}) error {
	cache.logger.DebugContext(ctx, "Getting item from cache", "instance", cache.instance)

	// do the lookup on the inner cache
	err := cache.cache.Get(ctx, key, item)
	// we got an error
	if err != nil {

		// was it a cache miss?
		if strings.Contains(err.Error(), cacheMissErr) {
			cache.logger.DebugContext(ctx, "Cache miss", "instance", cache.instance)
			cache.stats.incMisses()

			// temporarily downlevel this error
		} else if strings.Contains(err.Error(), decodeArrayErr1) || strings.Contains(err.Error(), decodeArrayErr2) {
			cache.logger.WarnContext(ctx, "Cache decode error", "instance", cache.instance, "error", err)
			cache.stats.incErrors()

			// or an actual error
		} else {
			cache.logger.WarnContext(ctx, "Fail get item from cache", "instance", cache.instance, "error", err)
			cache.stats.incErrors()
		}
		return err
	}

	// no error means cache hit
	cache.logger.DebugContext(ctx, "Cache hit", "instance", cache.instance)
	cache.stats.incHits()
	return nil
}

func (cache *Cache) set(ctx context.Context, item *cache.Item) error {
	cache.logger.DebugContext(ctx, "Set item in cache", "instance", cache.instance)

	err := cache.cache.Set(item)
	if err != nil {
		cache.logger.ErrorContext(ctx, "Fail set item in cache", "instance", cache.instance, "error", err)
	}

	cache.logger.DebugContext(ctx, "Add item in cache", "instance", cache.instance)
	return err
}

func (cache *Cache) delete(ctx context.Context, key string) error {
	cache.logger.DebugContext(ctx, "Delete item in cache", "instance", cache.instance)

	err := cache.cache.Delete(ctx, key)
	if err != nil {
		cache.logger.ErrorContext(ctx, "Fail delete item in cache", "instance", cache.instance, "error", err)
	}
	cache.logger.ErrorContext(ctx, "Finish delete item in cache", "instance", cache.instance)
	return err
}
