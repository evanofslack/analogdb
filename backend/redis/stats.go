package redis

import (
	"context"
	"time"

	"github.com/evanofslack/analogdb"
)

const (
	statsInstance  = "stats"
	statsLocalSize = 4
	statsTTL       = time.Hour
	statsLatestKey = "stats:latest"
)

// ensure interface is implemented
var _ analogdb.StatsService = (*StatsService)(nil)

// StatsService caches the latest snapshot, a new snapshot bumps the stats
// generation so every instance reads it on its next request
type StatsService struct {
	rdb        *RDB
	statsCache *Cache
	dbService  analogdb.StatsService
}

func NewCacheStatsService(rdb *RDB, dbService analogdb.StatsService) *StatsService {
	return &StatsService{
		rdb:        rdb,
		statsCache: rdb.NewCache(statsInstance, statsLocalSize, statsTTL),
		dbService:  dbService,
	}
}

// cachedSnapshot carries the body, which the snapshot itself keeps out of json
type cachedSnapshot struct {
	ID          int64
	Created     time.Time
	SeedVersion string
	BuildMs     int
	Body        []byte
}

func (s *StatsService) FindLatestSnapshot(ctx context.Context) (*analogdb.StatsSnapshot, error) {
	key := s.rdb.genCacheKey(ctx, statsEntity, statsLatestKey)
	cached, err := fetch(ctx, s.statsCache, key, statsTTL, func(ctx context.Context) (cachedSnapshot, error) {
		snapshot, err := s.dbService.FindLatestSnapshot(ctx)
		if err != nil {
			return cachedSnapshot{}, err
		}
		return cachedSnapshot{
			ID:          snapshot.ID,
			Created:     snapshot.Created,
			SeedVersion: snapshot.SeedVersion,
			BuildMs:     snapshot.BuildMs,
			Body:        snapshot.Body,
		}, nil
	})
	if err != nil {
		return nil, err
	}
	return &analogdb.StatsSnapshot{
		ID:          cached.ID,
		Created:     cached.Created,
		SeedVersion: cached.SeedVersion,
		BuildMs:     cached.BuildMs,
		Size:        len(cached.Body),
		Body:        cached.Body,
	}, nil
}

func (s *StatsService) CreateSnapshot(ctx context.Context, snapshot *analogdb.StatsSnapshot) error {
	if err := s.dbService.CreateSnapshot(ctx, snapshot); err != nil {
		return err
	}
	s.rdb.bumpGen(ctx, statsEntity)
	return nil
}

func (s *StatsService) ReplaceSeed(ctx context.Context, seed *analogdb.StatsSeed) error {
	return s.dbService.ReplaceSeed(ctx, seed)
}

func (s *StatsService) FindSeed(ctx context.Context) (*analogdb.StatsSeed, error) {
	return s.dbService.FindSeed(ctx)
}

func (s *StatsService) LoadStatsInput(ctx context.Context, since time.Time, subjects []string) (*analogdb.StatsInput, error) {
	return s.dbService.LoadStatsInput(ctx, since, subjects)
}

func (s *StatsService) FindSnapshots(ctx context.Context) ([]*analogdb.StatsSnapshot, error) {
	return s.dbService.FindSnapshots(ctx)
}
