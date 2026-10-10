package redis

import (
	"bytes"
	"context"
	"testing"
)

func TestDailySalt(t *testing.T) {
	rdb, mr := newMiniRDB(t)
	ctx := context.Background()

	first, err := rdb.DailySalt(ctx, "2026-10-10", []byte("first"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, []byte("first")) {
		t.Fatalf("want first salt stored, got %q", first)
	}

	second, err := rdb.DailySalt(ctx, "2026-10-10", []byte("second"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(second, []byte("first")) {
		t.Errorf("want the stored salt to win, got %q", second)
	}

	if ttl := mr.TTL(saltKey("2026-10-10")); ttl != saltTTL {
		t.Errorf("want ttl %s, got %s", saltTTL, ttl)
	}

	next, err := rdb.DailySalt(ctx, "2026-10-11", []byte("next"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(next, []byte("next")) {
		t.Errorf("want a new salt for the next day, got %q", next)
	}
}
