package redis

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	postsEntity   = "posts"
	filmsEntity   = "films"
	camerasEntity = "cameras"

	// how long a generation read from redis is reused
	genMemoTTL = time.Second
	// timeout for generation increments after a write
	genIncrTimeout = time.Second
)

// generation is an in process copy of an entity's generation counter
type generation struct {
	value atomic.Int64
	at    atomic.Int64
}

func newGenerations() map[string]*generation {
	return map[string]*generation{
		postsEntity:   {},
		filmsEntity:   {},
		camerasEntity: {},
	}
}

func genKey(entity string) string {
	return fmt.Sprintf("gen:%s", entity)
}

// genCacheKey versions a list cache key with the entity's generation
func (rdb *RDB) genCacheKey(ctx context.Context, entity string, key string) string {
	return fmt.Sprintf("%s:g%d", key, rdb.gen(ctx, entity))
}

// gen returns the entity's generation, read from redis at most once per genMemoTTL
func (rdb *RDB) gen(ctx context.Context, entity string) int64 {
	g := rdb.generations[entity]
	now := rdb.now().UnixNano()
	if at := g.at.Load(); at != 0 && now-at < genMemoTTL.Nanoseconds() {
		return g.value.Load()
	}

	value, err := rdb.readGen(ctx, entity)
	if err != nil {
		rdb.logger.WarnContext(ctx, "Fail get cache generation", "entity", entity, "error", err)
		return g.value.Load()
	}
	g.value.Store(value)
	g.at.Store(now)
	return value
}

// readGen reads the entity's generation from redis, skipping the in process copy
func (rdb *RDB) readGen(ctx context.Context, entity string) (int64, error) {
	value, err := rdb.db.Get(ctx, genKey(entity)).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	return value, err
}

// bumpGen increments the generation of each entity so older list keys are no longer read.
// Errors are logged and counted but not returned, the ttl still bounds staleness.
func (rdb *RDB) bumpGen(ctx context.Context, entities ...string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), genIncrTimeout)
	defer cancel()

	for _, entity := range entities {
		value, err := rdb.db.Incr(ctx, genKey(entity)).Result()
		if err != nil {
			rdb.logger.ErrorContext(ctx, "Fail increment cache generation", "entity", entity, "error", err)
			rdb.stats.incrErrors.Inc()
			continue
		}
		g := rdb.generations[entity]
		g.value.Store(value)
		g.at.Store(rdb.now().UnixNano())
		rdb.stats.invalidations.WithLabelValues(entity).Inc()
	}
}
