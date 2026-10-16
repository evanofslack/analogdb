package clickhouse

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/logger"
	tcclickhouse "github.com/testcontainers/testcontainers-go/modules/clickhouse"
)

type testRow struct {
	ago        time.Duration
	requestID  string
	ip         string
	url        string
	path       string
	method     string
	userAgent  string
	status     int32
	authorized bool
	latencyMs  int64
}

const (
	semrush = "Mozilla/5.0 (compatible; SemrushBot/7~bl; +http://www.semrush.com/bot.html)"
	safari  = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15"
)

var testRows = []testRow{
	{ago: time.Hour, requestID: "r1", ip: "10.0.0.1", url: "/v1/posts?sort=time&nsfw=false", path: "/v1/posts", method: "GET", userAgent: "analogdb-web/1.0", status: 200, latencyMs: 10},
	{ago: time.Hour, requestID: "r2", ip: "10.0.0.1", url: "/v1/post/12", path: "/v1/post/12", method: "GET", userAgent: "analogdb-web/1.0", status: 200, latencyMs: 20},
	{ago: 2 * time.Hour, requestID: "r3", ip: "1.2.3.4", url: "/post/12", path: "/post/12", method: "GET", userAgent: "curl/8", status: 200, latencyMs: 5},
	{ago: 2 * time.Hour, requestID: "r4", ip: "1.2.3.4", url: "/v1/post/12", path: "/v1/post/12", method: "GET", userAgent: "curl/8", status: 500, latencyMs: 100},
	{ago: 3 * time.Hour, requestID: "r5", ip: "10.0.0.2", url: "/v1/post/12", path: "/v1/post/12", method: "PATCH", userAgent: "analogdb-scraper/2.0", status: 200, authorized: true, latencyMs: 30},
	{ago: 4 * time.Hour, requestID: "r6", ip: "5.6.7.8", url: "/v1/post/13", path: "/v1/post/13", method: "DELETE", userAgent: "curl/8", status: 401, latencyMs: 1},
	{ago: 5 * time.Hour, requestID: "r9", ip: "7.7.7.1", url: "/v1/post/15/similar", path: "/v1/post/15/similar", method: "GET", userAgent: semrush, status: 200, latencyMs: 400},
	{ago: 6 * time.Hour, requestID: "r10", ip: "7.7.7.2", url: "/v1/post/16/similar", path: "/v1/post/16/similar", method: "GET", userAgent: semrush, status: 200, latencyMs: 300},
	{ago: 6 * time.Hour, requestID: "r11", ip: "7.7.7.3", url: "/v1/post/99", path: "/v1/post/99", method: "GET", userAgent: semrush, status: 404, latencyMs: 2},
	{ago: 7 * time.Hour, requestID: "r12", ip: "8.8.8.8", url: "/v1/posts", path: "/v1/posts", method: "GET", userAgent: safari, status: 200, latencyMs: 15},
	{ago: 8 * time.Hour, requestID: "r13", ip: "4.4.4.4", url: "/v1/films", path: "/v1/films", method: "GET", userAgent: "", status: 200, latencyMs: 3},
	{ago: 9 * time.Hour, requestID: "r14", ip: "3.3.3.3", url: "/v1/cameras", path: "/v1/cameras", method: "GET", userAgent: "python-requests/2.31", status: 404, latencyMs: 2},
	{ago: 30 * time.Hour, requestID: "r15", ip: "10.0.0.1", url: "/v1/posts", path: "/v1/posts", method: "GET", userAgent: "analogdb-web/1.0", status: 200, latencyMs: 10},
	{ago: 30 * time.Hour, requestID: "r16", ip: "1.2.3.4", url: "/v1/post/12", path: "/v1/post/12", method: "GET", userAgent: "curl/8", status: 500, latencyMs: 50},
	{ago: 3 * 24 * time.Hour, requestID: "r7", ip: "10.0.0.1", url: "/v1/posts", path: "/v1/posts", method: "GET", userAgent: "analogdb-web/1.0", status: 200, latencyMs: 10},
	{ago: 40 * 24 * time.Hour, requestID: "r8", ip: "9.9.9.9", url: "/v1/post/14", path: "/v1/post/14", method: "DELETE", userAgent: "analogdb-web/1.0", status: 200, authorized: true, latencyMs: 15},
}

func mustOpen(t *testing.T) *DB {
	t.Helper()
	ctx := context.Background()

	container, err := tcclickhouse.Run(ctx, "clickhouse/clickhouse-server:24.8",
		tcclickhouse.WithUsername("test"),
		tcclickhouse.WithPassword("test"),
		tcclickhouse.WithDatabase("analytics"),
	)
	if err != nil {
		t.Fatalf("Start clickhouse container, err=%v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Terminate clickhouse container, err=%v", err)
		}
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "9000/tcp")
	if err != nil {
		t.Fatal(err)
	}

	l, err := logger.New("error", "debug", "analogdb-test")
	if err != nil {
		t.Fatal(err)
	}
	db, err := NewDB(host, int(port.Num()), "analytics", "test", "test", "", l)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Open(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	schema, err := os.ReadFile(filepath.Join("testdata", "httprequests.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.conn.Exec(ctx, string(schema)); err != nil {
		t.Fatal(err)
	}

	batch, err := db.conn.PrepareBatch(ctx, `INSERT INTO httprequests (request_id, remote_ip, url, path, method, user_agent, response_code, authorized, start_time, end_time, request_time_ms)`)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for _, row := range testRows {
		start := now.Add(-row.ago).UnixMilli()
		if err := batch.Append(row.requestID, row.ip, row.url, row.path, row.method, row.userAgent, row.status, row.authorized, start, start+row.latencyMs, row.latencyMs); err != nil {
			t.Fatal(err)
		}
	}
	if err := batch.Send(); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestClickhouse(t *testing.T) {
	db := mustOpen(t)
	ctx := context.Background()

	t.Run("readyz", func(t *testing.T) {
		if err := db.Readyz(ctx); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("audit", func(t *testing.T) {
		entries, err := db.Audit(ctx, &analogdb.AuditFilter{Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		got := []string{}
		for _, e := range entries {
			got = append(got, e.RequestID)
		}
		if len(got) != 3 || got[0] != "r5" || got[1] != "r6" || got[2] != "r8" {
			t.Fatalf("want audit r5 r6 r8, got %v", got)
		}
		if entries[0].Client != "scraper" || entries[1].Status != 401 {
			t.Errorf("unexpected audit entries %+v %+v", entries[0], entries[1])
		}

		before := entries[1].StartMs
		entries, err = db.Audit(ctx, &analogdb.AuditFilter{Limit: 10, Before: &before})
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].RequestID != "r8" {
			t.Errorf("want only r8 before r6, got %+v", entries)
		}

		for client, want := range map[string][]string{"web": {"r8"}, "scraper": {"r5"}, "direct": {"r6"}} {
			entries, err = db.Audit(ctx, &analogdb.AuditFilter{Limit: 10, Client: client})
			if err != nil {
				t.Fatal(err)
			}
			got := []string{}
			for _, e := range entries {
				got = append(got, e.RequestID)
			}
			if !slices.Equal(got, want) {
				t.Errorf("want %s audit %v, got %v", client, want, got)
			}
		}
	})
}

// mustFilled checks the series has one bucket per step from the window start to now
func mustFilled(t *testing.T, times []time.Time, step time.Duration, buckets int) {
	t.Helper()
	if len(times) != buckets && len(times) != buckets+1 {
		t.Fatalf("want %d or %d buckets, got %d", buckets, buckets+1, len(times))
	}
	for i := 1; i < len(times); i++ {
		if got := times[i].Sub(times[i-1]); got != step {
			t.Fatalf("want step %s at bucket %d, got %s", step, i, got)
		}
	}
	// the query may have run in the previous bucket if a boundary passed since
	last := times[len(times)-1]
	now := time.Now().UTC().Truncate(step)
	if !last.Equal(now) && !last.Equal(now.Add(-step)) {
		t.Errorf("want last bucket %s, got %s", now, last)
	}
}

func TestNewDBRejectsBadTable(t *testing.T) {
	l, err := logger.New("error", "debug", "analogdb-test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewDB("localhost", 9000, "db", "u", "p", "requests; DROP TABLE x", l); err == nil {
		t.Error("want error for bad table name")
	}
}
