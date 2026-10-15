package redis

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/evanofslack/analogdb/logger"
	"github.com/evanofslack/analogdb/metrics"
	rediscache "github.com/go-redis/cache/v9"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"

	"github.com/redis/go-redis/extra/redisotel/v9"
)

const (
	// max ttl for the in memory cache of each instance
	localTTL = time.Minute
)

type RDB struct {
	db          *redis.Client
	ctx         context.Context
	cancel      func()
	logger      *logger.Logger
	metrics     *metrics.Metrics
	collector   *cacheCollector
	stats       *genStats
	generations map[string]*generation
	now         func() time.Time
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
		db:          db,
		ctx:         ctx,
		cancel:      cancel,
		logger:      logger,
		metrics:     metrics,
		collector:   collector,
		stats:       newGenStats(),
		generations: newGenerations(),
		now:         time.Now,
	}

	// prometheus metrics for redis based caches
	rdb.metrics.Registry.MustRegister(rdb.collector)
	rdb.stats.register(rdb.metrics.Registry)
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

func (rdb *RDB) Readyz(ctx context.Context) error {
	return rdb.db.Ping(ctx).Err()
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
	cache    *rediscache.Cache
	local    rediscache.LocalCache
	redis    *redis.Client
	instance string
	stats    *cacheStats
	logger   *logger.Logger
	group    singleflight.Group
	// counts deletes, a read that raced one drops what it put in the local cache
	deletes atomic.Int64
}

// create a new cache backed by redis
func (rdb *RDB) NewCache(instance string, size int, ttl time.Duration) *Cache {
	rdb.logger.Debug("Initializing new cache", "instance", instance)

	local := rediscache.NewTinyLFU(size, min(ttl, localTTL))
	inner := rediscache.New(&rediscache.Options{
		Redis:        rdb.db,
		LocalCache:   local,
		StatsEnabled: true,
	})

	stats := newCacheStats()

	cache := &Cache{
		cache:    inner,
		local:    local,
		redis:    rdb.db,
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

// get looks up a key and decodes it into item.
// Returns cache.ErrCacheMiss when the key is missing or can't be decoded.
func (cache *Cache) get(ctx context.Context, key string, item interface{}) error {
	cache.logger.DebugContext(ctx, "Getting item from cache", "instance", cache.instance)

	err := cache.lookup(ctx, key, item)
	if errors.Is(err, errDecode) {
		cache.logger.WarnContext(ctx, "Cache decode error", "instance", cache.instance, "error", err)
		cache.stats.incErrors(errorKindDecode)
		_ = cache.delete(ctx, key)
		return rediscache.ErrCacheMiss
	}
	if errors.Is(err, rediscache.ErrCacheMiss) {
		cache.logger.DebugContext(ctx, "Cache miss", "instance", cache.instance)
		cache.stats.incMisses()
		return err
	}
	if err != nil {
		cache.logger.WarnContext(ctx, "Fail get item from cache", "instance", cache.instance, "error", err)
		cache.stats.incErrors(errorKindRedis)
		return err
	}

	cache.logger.DebugContext(ctx, "Cache hit", "instance", cache.instance)
	cache.stats.incHits()
	return nil
}

var errDecode = errors.New("cache decode error")

// lookup is get without logging or stats, it returns errDecode when the value can't be decoded
func (cache *Cache) lookup(ctx context.Context, key string, item interface{}) error {
	b, err := cache.getBytes(ctx, key)
	if err != nil {
		return err
	}
	if err := cache.cache.Unmarshal(b, item); err != nil {
		return fmt.Errorf("%w: %w", errDecode, err)
	}
	return nil
}

func (cache *Cache) getBytes(ctx context.Context, key string) ([]byte, error) {
	if b, ok := cache.local.Get(key); ok {
		return b, nil
	}
	deletes := cache.deletes.Load()
	b, err := cache.redis.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, rediscache.ErrCacheMiss
	}
	if err != nil {
		return nil, err
	}
	// check after the set, a delete either comes after it or bumped the count before this check
	cache.local.Set(key, b)
	if cache.deletes.Load() != deletes {
		cache.local.Del(key)
	}
	return b, nil
}

func (cache *Cache) set(ctx context.Context, key string, value interface{}, ttl time.Duration) {
	cache.logger.DebugContext(ctx, "Set item in cache", "instance", cache.instance)

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cacheOpTimeout)
	defer cancel()

	err := cache.cache.Set(&rediscache.Item{
		Ctx:   ctx,
		Key:   key,
		Value: value,
		TTL:   ttl,
	})
	if err != nil {
		cache.logger.ErrorContext(ctx, "Fail set item in cache", "instance", cache.instance, "error", err)
		cache.stats.incErrors(errorKindRedis)
	}
}

func (cache *Cache) delete(ctx context.Context, key string) error {
	cache.logger.DebugContext(ctx, "Delete item in cache", "instance", cache.instance)

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cacheOpTimeout)
	defer cancel()

	cache.deletes.Add(1)
	err := cache.cache.Delete(ctx, key)
	if err != nil {
		cache.logger.ErrorContext(ctx, "Fail delete item in cache", "instance", cache.instance, "error", err)
		cache.stats.incErrors(errorKindRedis)
	}
	cache.logger.DebugContext(ctx, "Finish delete item in cache", "instance", cache.instance)
	return err
}

// deleteMany removes keys from redis and the local cache in one call
func (cache *Cache) deleteMany(ctx context.Context, keys []string) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cacheOpTimeout)
	defer cancel()

	cache.deletes.Add(1)
	for _, key := range keys {
		cache.local.Del(key)
	}
	if err := cache.redis.Del(ctx, keys...).Err(); err != nil {
		cache.logger.ErrorContext(ctx, "Fail delete items in cache", "instance", cache.instance, "count", len(keys), "error", err)
		cache.stats.incErrors(errorKindRedis)
		return err
	}
	return nil
}

// once runs load for a key at most once at a time and stores the result.
// Concurrent misses on the same key share one call to load.
func (cache *Cache) once(ctx context.Context, key string, load func(ctx context.Context) (any, error)) (any, error) {
	v, err, _ := cache.group.Do(key, func() (any, error) {
		return load(context.WithoutCancel(ctx))
	})
	return v, err
}

// fetch returns the value under key or loads, stores and returns it.
func fetch[T any](ctx context.Context, cache *Cache, key string, ttl time.Duration, load func(ctx context.Context) (T, error)) (T, error) {
	var value T
	if err := cache.get(ctx, key, &value); err == nil {
		return value, nil
	}

	v, err := cache.once(ctx, key, func(ctx context.Context) (any, error) {
		// a load that finished after our miss may have stored it already
		var value T
		if err := cache.lookup(ctx, key, &value); err == nil {
			return value, nil
		}
		value, err := load(ctx)
		if err != nil {
			return nil, err
		}
		cache.set(ctx, key, value, ttl)
		return value, nil
	})
	if err != nil {
		var zero T
		return zero, err
	}
	return v.(T), nil
}
