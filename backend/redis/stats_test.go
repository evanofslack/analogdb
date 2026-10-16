package redis

import (
	"context"
	"testing"

	"github.com/evanofslack/analogdb"
)

type fakeStatsService struct {
	analogdb.StatsService
	snapshots []*analogdb.StatsSnapshot
	reads     int
}

func (f *fakeStatsService) FindLatestSnapshot(ctx context.Context) (*analogdb.StatsSnapshot, error) {
	f.reads++
	if len(f.snapshots) == 0 {
		return nil, &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "stats not built yet"}
	}
	return f.snapshots[len(f.snapshots)-1], nil
}

func (f *fakeStatsService) CreateSnapshot(ctx context.Context, snapshot *analogdb.StatsSnapshot) error {
	snapshot.ID = int64(len(f.snapshots) + 1)
	f.snapshots = append(f.snapshots, snapshot)
	return nil
}

func TestCacheStatsService(t *testing.T) {
	rdb, _ := newMiniRDB(t)
	db := &fakeStatsService{}
	s := NewCacheStatsService(rdb, db)
	ctx := context.Background()

	if _, err := s.FindLatestSnapshot(ctx); analogdb.ErrorCode(err) != analogdb.ERRNOTFOUND {
		t.Fatalf("want not found before a build, got %v", err)
	}
	if err := s.CreateSnapshot(ctx, &analogdb.StatsSnapshot{SeedVersion: "v1", Body: []byte(`{"version":1}`)}); err != nil {
		t.Fatal(err)
	}

	for range 3 {
		got, err := s.FindLatestSnapshot(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if got.ID != 1 || string(got.Body) != `{"version":1}` || got.Size != len(got.Body) {
			t.Errorf("want snapshot 1 with its body, got %+v", got)
		}
	}
	if db.reads != 2 {
		t.Errorf("want one read after the miss, got %d", db.reads-1)
	}

	if err := s.CreateSnapshot(ctx, &analogdb.StatsSnapshot{SeedVersion: "v1", Body: []byte(`{"version":2}`)}); err != nil {
		t.Fatal(err)
	}
	got, err := s.FindLatestSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 2 {
		t.Errorf("want the new snapshot after a build, got %d", got.ID)
	}
}
