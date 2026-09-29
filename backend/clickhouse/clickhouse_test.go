package clickhouse

import (
	"context"
	"os"
	"path/filepath"
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

var testRows = []testRow{
	{ago: time.Hour, requestID: "r1", ip: "10.0.0.1", url: "/v1/posts?sort=time&nsfw=false", path: "/v1/posts", method: "GET", userAgent: "analogdb-web/1.0", status: 200, latencyMs: 10},
	{ago: time.Hour, requestID: "r2", ip: "10.0.0.1", url: "/v1/post/12", path: "/v1/post/12", method: "GET", userAgent: "analogdb-web/1.0", status: 200, latencyMs: 20},
	{ago: 2 * time.Hour, requestID: "r3", ip: "1.2.3.4", url: "/post/12", path: "/post/12", method: "GET", userAgent: "curl/8", status: 200, latencyMs: 5},
	{ago: 2 * time.Hour, requestID: "r4", ip: "1.2.3.4", url: "/v1/post/12", path: "/v1/post/12", method: "GET", userAgent: "curl/8", status: 500, latencyMs: 100},
	{ago: 3 * time.Hour, requestID: "r5", ip: "10.0.0.2", url: "/v1/post/12", path: "/v1/post/12", method: "PATCH", userAgent: "analogdb-scraper/2.0", status: 200, authorized: true, latencyMs: 30},
	{ago: 4 * time.Hour, requestID: "r6", ip: "5.6.7.8", url: "/v1/post/13", path: "/v1/post/13", method: "DELETE", userAgent: "curl/8", status: 401, latencyMs: 1},
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
	db, err := NewDB(host, port.Int(), "analytics", "test", "test", "", l)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Open(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

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

	t.Run("traffic day", func(t *testing.T) {
		traffic, err := db.Traffic(ctx, analogdb.TrafficDay)
		if err != nil {
			t.Fatal(err)
		}
		if traffic.Bucket != "hour" {
			t.Errorf("want hour buckets, got %s", traffic.Bucket)
		}
		wantTotals := analogdb.TrafficTotals{Requests: 6, UniqueIPs: 4, Status2: 4, Status4: 1, Status5: 1}
		if traffic.Totals != wantTotals {
			t.Errorf("want totals %+v, got %+v", wantTotals, traffic.Totals)
		}

		var web, scraper, other int64
		for _, b := range traffic.Series {
			web, scraper, other = web+b.Web, scraper+b.Scraper, other+b.Other
		}
		if web != 2 || scraper != 1 || other != 3 {
			t.Errorf("want series web 2 scraper 1 other 3, got %d %d %d", web, scraper, other)
		}

		if len(traffic.Routes) == 0 || traffic.Routes[0].Route != "/v1/post/{id}" || traffic.Routes[0].Requests != 4 || traffic.Routes[0].Errors != 1 {
			t.Errorf("unexpected top route %+v", traffic.Routes)
		}
		if len(traffic.Posts) != 2 || traffic.Posts[0] != (analogdb.TrafficPost{PostID: 12, Requests: 4}) {
			t.Errorf("unexpected posts %+v", traffic.Posts)
		}

		legacy := map[string]analogdb.TrafficLegacy{}
		for _, l := range traffic.Legacy {
			legacy[l.Client] = l
		}
		if legacy["other"].Legacy != 1 || legacy["web"].Legacy != 0 {
			t.Errorf("unexpected legacy %+v", traffic.Legacy)
		}

		params := map[string]int64{}
		for _, p := range traffic.Params {
			params[p.Name] = p.Requests
		}
		if params["sort"] != 1 || params["nsfw"] != 1 || len(params) != 2 {
			t.Errorf("unexpected params %+v", traffic.Params)
		}

		if len(traffic.IPs) == 0 || traffic.IPs[0].Name != "1.2.3.4" || traffic.IPs[0].Requests != 2 {
			t.Errorf("unexpected ips %+v", traffic.IPs)
		}
		for _, ip := range traffic.IPs {
			if ip.Name == "10.0.0.1" {
				t.Errorf("web ip in other ips %+v", traffic.IPs)
			}
		}
		if len(traffic.UserAgents) == 0 || traffic.UserAgents[0].Name != "curl/8" || traffic.UserAgents[0].Client != "other" {
			t.Errorf("unexpected user agents %+v", traffic.UserAgents)
		}
		if len(traffic.Errors) != 1 || traffic.Errors[0].RequestID != "r4" || traffic.Errors[0].Status != 500 {
			t.Errorf("unexpected errors %+v", traffic.Errors)
		}
	})

	t.Run("traffic week", func(t *testing.T) {
		traffic, err := db.Traffic(ctx, analogdb.TrafficWeek)
		if err != nil {
			t.Fatal(err)
		}
		if traffic.Bucket != "day" || traffic.Totals.Requests != 7 {
			t.Errorf("want 7 requests in day buckets, got %d in %s", traffic.Totals.Requests, traffic.Bucket)
		}
	})

	t.Run("traffic bad range", func(t *testing.T) {
		if _, err := db.Traffic(ctx, "1y"); analogdb.ErrorCode(err) != analogdb.ERRBADREQUEST {
			t.Errorf("want bad request, got %v", err)
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
	})
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
