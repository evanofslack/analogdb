package clickhouse

import (
	"context"
	"log/slog"
	"testing"
	"time"

	tcclickhouse "github.com/testcontainers/testcontainers-go/modules/clickhouse"

	v1 "github.com/evanofslack/analogdb-consumer/internal/gen/proto/analytics/v1"
	"github.com/evanofslack/analogdb-consumer/internal/metrics"
)

func TestInsertAfterMigrations(t *testing.T) {
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

	tests := []struct {
		name          string
		migrationPath string
	}{
		{name: "embedded", migrationPath: ""},
		{name: "path", migrationPath: "migrations"},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := metrics.New(slog.Default())
			if err != nil {
				t.Fatal(err)
			}
			c, err := New(slog.Default(), m, host, int(port.Num()), "analytics", "test", "test",
				"httprequests", "analogdb-consumer-test", "test", true, tt.migrationPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := c.Open(); err != nil {
				t.Fatalf("Open, err=%v", err)
			}
			t.Cleanup(func() { _ = c.Close() })

			if err := c.HealthCheck(ctx); err != nil {
				t.Fatalf("HealthCheck after migrations, err=%v", err)
			}

			now := time.Now().UnixMilli()
			event := &v1.Event{
				RequestId:    tt.name,
				RemoteIp:     "1.2.3.4",
				Path:         "/v1/posts",
				Method:       "GET",
				ResponseCode: 200,
				StartTime:    now,
				EndTime:      now + 5,
			}
			if err := c.Insert(ctx, []*v1.Event{event}); err != nil {
				t.Fatalf("Insert after migrations, err=%v", err)
			}

			var count uint64
			if err := c.conn.QueryRow(ctx, "SELECT count() FROM httprequests").Scan(&count); err != nil {
				t.Fatal(err)
			}
			if want := uint64(i + 1); count != want {
				t.Fatalf("count = %d, want %d", count, want)
			}
		})
	}
}
