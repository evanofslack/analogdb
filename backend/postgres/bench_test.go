package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/logger"
	"github.com/lib/pq"
)

const (
	benchDSNEnv      = "ANALOGDB_BENCH_DSN"
	benchExplainEnv  = "ANALOGDB_BENCH_EXPLAIN"
	benchReadOnlyEnv = "ANALOGDB_BENCH_READONLY"
	benchMigrateEnv  = "ANALOGDB_BENCH_MIGRATE"
)

// TestBenchMigrate applies the migrations to the benchmark database.
func TestBenchMigrate(t *testing.T) {
	dsn := os.Getenv(benchDSNEnv)
	if dsn == "" || os.Getenv(benchMigrateEnv) == "" {
		t.Skipf("set %s and %s to migrate the benchmark database", benchDSNEnv, benchMigrateEnv)
	}
	if os.Getenv(benchReadOnlyEnv) != "" {
		t.Skip("read only mode")
	}
	log, err := logger.New("error", "debug", "analogdb_bench")
	if err != nil {
		t.Fatal(err)
	}
	db := NewDB(dsn, log, true, "", false)
	if err := db.Open(); err != nil {
		t.Fatal(err)
	}
	mustClose(t, db)
}

// BenchmarkFindPosts runs representative queries against a seeded database.
// See bench/README.md.
func BenchmarkFindPosts(b *testing.B) {
	db, rec := openBench(b)
	defer db.Close()
	ctx := context.Background()

	cases := benchCases(b, db)

	if path := os.Getenv(benchExplainEnv); path != "" {
		if err := writeExplain(ctx, db, rec, cases, path); err != nil {
			b.Fatal(err)
		}
	}

	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			if err := c.run(ctx); err != nil {
				b.Fatal(err)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := c.run(ctx); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

type benchCase struct {
	name string
	run  func(ctx context.Context) error
}

func benchCases(b *testing.B, db *DB) []benchCase {
	b.Helper()
	ctx := context.Background()
	posts := NewPostService(db)
	cameras := NewCameraService(db)

	postFilter := func(sort analogdb.PostSort) *analogdb.PostFilter {
		limit := 20
		return analogdb.NewPostFilter(&limit, &sort, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	}
	findPosts := func(build func() *analogdb.PostFilter) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			_, _, err := posts.FindPosts(ctx, build())
			return err
		}
	}

	// keyset for the fifth page of the score sort, found by paging like a client
	var scoreKeyset int
	for page := 1; page < 5; page++ {
		filter := postFilter(analogdb.PostSortScore)
		if page > 1 {
			filter.Keyset = &scoreKeyset
		}
		found, _, err := posts.FindPosts(ctx, filter)
		if err != nil {
			b.Fatal(err)
		}
		if len(found) == 0 {
			b.Fatal("benchmark database has no posts, run make bench-seed")
		}
		scoreKeyset = found[len(found)-1].Score
	}

	randomSQL := func(order string) func(ctx context.Context) error {
		query := "SELECT * FROM pictures ORDER BY " + order + " LIMIT 20"
		return func(ctx context.Context) error {
			rows, err := db.db.QueryContext(ctx, query)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
			}
			return rows.Err()
		}
	}

	return []benchCase{
		{"posts_default", findPosts(func() *analogdb.PostFilter {
			return postFilter(analogdb.PostSortTime)
		})},
		{"sort_score_page5", findPosts(func() *analogdb.PostFilter {
			filter := postFilter(analogdb.PostSortScore)
			keyset := scoreKeyset
			filter.Keyset = &keyset
			return filter
		})},
		{"sort_random", findPosts(func() *analogdb.PostFilter {
			filter := postFilter(analogdb.PostSortRandom)
			seed := 37
			filter.Seed = &seed
			return filter
		})},
		{"sort_random_mod_sql", randomSQL("MOD(time, 37), time DESC")},
		{"sort_random_md5_sql", randomSQL("md5(id::text || '42'), id")},
		{"keyword_portrait", findPosts(func() *analogdb.PostFilter {
			filter := postFilter(analogdb.PostSortTime)
			filter.Keywords = &[]string{"portrait"}
			return filter
		})},
		{"color_teal_min_0.2", findPosts(func() *analogdb.PostFilter {
			filter := postFilter(analogdb.PostSortTime)
			filter.Colors = &[]string{"teal"}
			filter.ColorPercents = &[]float64{0.2}
			filter.SetMinColorPercent()
			return filter
		})},
		{"camera_nikon_fm2", findPosts(func() *analogdb.PostFilter {
			filter := postFilter(analogdb.PostSortTime)
			make, model := "nikon", "fm2"
			filter.CameraMake = &make
			filter.CameraModel = &model
			return filter
		})},
		{"title_beach", findPosts(func() *analogdb.PostFilter {
			filter := postFilter(analogdb.PostSortTime)
			title := "beach"
			filter.Title = &title
			return filter
		})},
		{"cameras_with_counts", func(ctx context.Context) error {
			sort := analogdb.CameraSortAlphabetical
			include, exclude := true, true
			filter := analogdb.NewCameraFilter(nil, &sort, nil, nil, nil, nil, nil, &include, &exclude)
			_, err := cameras.FindCameras(ctx, filter)
			return err
		}},
	}
}

func openBench(b *testing.B) (*DB, *queryRecorder) {
	b.Helper()
	dsn := os.Getenv(benchDSNEnv)
	if dsn == "" {
		b.Skipf("set %s to run database benchmarks", benchDSNEnv)
	}
	if os.Getenv(benchReadOnlyEnv) != "" {
		dsn = withRuntimeParam(dsn, "default_transaction_read_only", "on")
	}
	connector, err := pq.NewConnector(dsn)
	if err != nil {
		b.Fatal(err)
	}
	log, err := logger.New("error", "debug", "analogdb_bench")
	if err != nil {
		b.Fatal(err)
	}
	rec := &queryRecorder{}
	db := NewDB(dsn, log, false, "", false)
	db.db = sql.OpenDB(&recordingConnector{Connector: connector, rec: rec})
	if err := db.db.PingContext(context.Background()); err != nil {
		b.Fatal(err)
	}
	return db, rec
}

func withRuntimeParam(dsn, key, value string) string {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, err := url.Parse(dsn)
		if err == nil {
			q := u.Query()
			q.Set(key, value)
			u.RawQuery = q.Encode()
			return u.String()
		}
	}
	return dsn + " " + key + "=" + value
}

// writeExplain runs each case once, then writes EXPLAIN (ANALYZE, BUFFERS)
// for every statement the case sent.
func writeExplain(ctx context.Context, db *DB, rec *queryRecorder, cases []benchCase, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, c := range cases {
		rec.start()
		err := c.run(ctx)
		queries := rec.stop()
		if err != nil {
			return fmt.Errorf("%s: %w", c.name, err)
		}
		for i, q := range queries {
			fmt.Fprintf(f, "=== %s (statement %d of %d)\n%s\nargs: %v\n\n", c.name, i+1, len(queries), strings.TrimSpace(q.query), q.args)
			rows, err := db.db.QueryContext(ctx, "EXPLAIN (ANALYZE, BUFFERS) "+q.query, q.args...)
			if err != nil {
				return fmt.Errorf("%s: %w", c.name, err)
			}
			for rows.Next() {
				var line string
				if err := rows.Scan(&line); err != nil {
					rows.Close()
					return err
				}
				fmt.Fprintln(f, line)
			}
			if err := rows.Err(); err != nil {
				rows.Close()
				return err
			}
			rows.Close()
			fmt.Fprintln(f)
		}
	}
	return nil
}

type recordedQuery struct {
	query string
	args  []any
}

type queryRecorder struct {
	mu        sync.Mutex
	recording bool
	queries   []recordedQuery
}

func (r *queryRecorder) start() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recording = true
	r.queries = nil
}

func (r *queryRecorder) stop() []recordedQuery {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recording = false
	return r.queries
}

func (r *queryRecorder) add(query string, args []driver.NamedValue) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.recording {
		return
	}
	values := make([]any, len(args))
	for i, a := range args {
		values[i] = a.Value
	}
	r.queries = append(r.queries, recordedQuery{query: query, args: values})
}

// recordingConnector wraps the pq connector to capture the SQL each case sends.
type recordingConnector struct {
	driver.Connector
	rec *queryRecorder
}

func (c *recordingConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &recordingConn{Conn: conn, rec: c.rec}, nil
}

type recordingConn struct {
	driver.Conn
	rec *queryRecorder
}

func (c *recordingConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.rec.add(query, args)
	return c.Conn.(driver.QueryerContext).QueryContext(ctx, query, args)
}

func (c *recordingConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.rec.add(query, args)
	return c.Conn.(driver.ExecerContext).ExecContext(ctx, query, args)
}

func (c *recordingConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	return c.Conn.(driver.ConnPrepareContext).PrepareContext(ctx, query)
}

func (c *recordingConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return c.Conn.(driver.ConnBeginTx).BeginTx(ctx, opts)
}

func (c *recordingConn) Ping(ctx context.Context) error {
	return c.Conn.(driver.Pinger).Ping(ctx)
}

func (c *recordingConn) ResetSession(ctx context.Context) error {
	return c.Conn.(driver.SessionResetter).ResetSession(ctx)
}

func (c *recordingConn) IsValid() bool {
	return c.Conn.(driver.Validator).IsValid()
}
