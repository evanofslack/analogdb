package clickhouse

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/evanofslack/analogdb"
	"golang.org/x/sync/errgroup"
)

const (
	uiTable       = "ui_events"
	liveWindow    = 30 * time.Minute
	browsersLimit = 10
	vitalsLimit   = 10
)

// visitor_id rotates daily, so a visitor on two days counts as two visitor-days
const (
	uiTimeExpr   = `toDateTime(received_ts, 'UTC')`
	uiWindow     = `received_ts >= fromUnixTimestamp64Milli(toInt64(?), 'UTC') AND received_ts < fromUnixTimestamp64Milli(toInt64(?), 'UTC')`
	pageViewCond = `event_name = 'page_view' AND NOT is_bot AND ` + uiWindow
	viewsExpr    = `uniqExact(event_id)`
	visitorsExpr = `uniq((toDate(received_ts), visitor_id))`
	referrerCond = `referrer_host != '' AND referrer_host != 'analogdb.com' AND NOT endsWith(referrer_host, '.analogdb.com')`
)

func (db *DB) Analytics(ctx context.Context, r analogdb.TrafficRange) (*analogdb.Analytics, error) {
	db.logger.DebugContext(ctx, "Starting analytics query", "range", r)
	defer db.logger.DebugContext(ctx, "Finished analytics query")

	window, ok := r.Duration()
	if !ok {
		return nil, &analogdb.Error{Code: analogdb.ERRBADREQUEST, Message: fmt.Sprintf("invalid range: %s", r)}
	}
	now := time.Now()
	until := now.UnixMilli()
	since := now.Add(-window).UnixMilli()
	before := now.Add(-2 * window).UnixMilli()
	live := now.Add(-liveWindow).UnixMilli()
	bucketName, bucketExpr := bucketFor(r, uiTimeExpr)

	fill := fillFor(r)

	analytics := &analogdb.Analytics{Range: r, Bucket: bucketName}
	top := &analytics.Top
	g, gctx := errgroup.WithContext(ctx)
	counts := func(dst *[]analogdb.AnalyticsCount, column, filter string, limit int) func() error {
		return func() (err error) {
			*dst, err = db.analyticsCounts(gctx, column, filter, limit, since, until)
			return
		}
	}

	steps := []struct {
		name string
		run  func() error
	}{
		{"summary", func() error { return db.analyticsSummary(gctx, before, since, live, until, &analytics.Summary) }},
		{"series", func() (err error) {
			analytics.Series, err = db.analyticsSeries(gctx, since, until, bucketExpr, fill)
			return
		}},
		{"pages", counts(&top.Pages, "route", "", topLimit)},
		{"referrers", counts(&top.Referrers, "referrer_host", referrerCond, topLimit)},
		{"sources", counts(&top.Sources, "utm_source", "utm_source != ''", topLimit)},
		{"campaigns", counts(&top.Campaigns, "utm_campaign", "utm_campaign != ''", topLimit)},
		{"devices", counts(&top.Devices, "device_type", "", topLimit)},
		{"browsers", counts(&top.Browsers, "browser", "", browsersLimit)},
		{"posts", func() (err error) { analytics.Posts, err = db.analyticsPosts(gctx, since, until); return }},
		{"vitals", func() (err error) { analytics.Vitals, err = db.analyticsVitals(gctx, since, until); return }},
	}
	for _, step := range steps {
		g.Go(func() error {
			if err := step.run(); err != nil {
				if gctx.Err() == nil {
					db.logger.ErrorContext(ctx, "Fail analytics query", "step", step.name, "error", err)
				}
				return fmt.Errorf("analytics %s: %w", step.name, err)
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return analytics, nil
}

// analyticsSummary reads the current window, the one before it and the live window in one pass
func (db *DB) analyticsSummary(ctx context.Context, before, since, live, until int64, s *analogdb.AnalyticsSummary) error {
	query := fmt.Sprintf(`
		WITH
			fromUnixTimestamp64Milli(toInt64(?), 'UTC') AS since_ts,
			fromUnixTimestamp64Milli(toInt64(?), 'UTC') AS live_ts
		SELECT
			toInt64(uniqExactIf(event_id, NOT is_bot AND received_ts >= since_ts)),
			toInt64(uniqIf((toDate(received_ts), visitor_id), NOT is_bot AND received_ts >= since_ts)),
			toInt64(uniqExactIf(event_id, is_bot AND received_ts >= since_ts)),
			toInt64(uniqExactIf(event_id, NOT is_bot AND received_ts < since_ts)),
			toInt64(uniqIf((toDate(received_ts), visitor_id), NOT is_bot AND received_ts < since_ts)),
			toInt64(uniqExactIf(event_id, is_bot AND received_ts < since_ts)),
			toInt64(uniqExactIf(event_id, NOT is_bot AND received_ts >= live_ts)),
			toInt64(uniqIf((toDate(received_ts), visitor_id), NOT is_bot AND received_ts >= live_ts))
		FROM %s
		WHERE event_name = 'page_view' AND %s`, uiTable, uiWindow)
	rows, done, err := db.query(ctx, query, since, live, before, until)
	if err != nil {
		return err
	}
	defer done()

	cur, prev := &s.Current, &s.Previous
	if rows.Next() {
		if err := rows.Scan(
			&cur.PageViews, &cur.Visitors, &cur.BotViews,
			&prev.PageViews, &prev.Visitors, &prev.BotViews,
			&s.Live.PageViews, &s.Live.Visitors,
		); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (db *DB) analyticsSeries(ctx context.Context, from, to int64, bucketExpr, fill string) ([]analogdb.AnalyticsBucket, error) {
	query := fmt.Sprintf(`
		SELECT %s AS bucket, toInt64(%s), toInt64(%s)
		FROM %s
		WHERE %s
		GROUP BY bucket
		ORDER BY bucket %s`, bucketExpr, viewsExpr, visitorsExpr, uiTable, pageViewCond, fill)
	rows, done, err := db.query(ctx, query, from, to, from, to)
	if err != nil {
		return nil, err
	}
	defer done()

	series := make([]analogdb.AnalyticsBucket, 0)
	for rows.Next() {
		var b analogdb.AnalyticsBucket
		if err := rows.Scan(&b.Time, &b.PageViews, &b.Visitors); err != nil {
			return nil, err
		}
		series = append(series, b)
	}
	return series, rows.Err()
}

// analyticsCounts groups page views by column, filter is an optional condition without bind parameters
func (db *DB) analyticsCounts(ctx context.Context, column, filter string, limit int, from, to int64) ([]analogdb.AnalyticsCount, error) {
	if filter != "" {
		filter = " AND " + filter
	}
	query := fmt.Sprintf(`
		SELECT toString(%s) AS name, toInt64(%s) AS views, toInt64(%s)
		FROM %s
		WHERE %s%s
		GROUP BY name
		ORDER BY views DESC, name
		LIMIT ?`, column, viewsExpr, visitorsExpr, uiTable, pageViewCond, filter)
	rows, done, err := db.query(ctx, query, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer done()

	counts := make([]analogdb.AnalyticsCount, 0)
	for rows.Next() {
		var c analogdb.AnalyticsCount
		if err := rows.Scan(&c.Name, &c.PageViews, &c.Visitors); err != nil {
			return nil, err
		}
		counts = append(counts, c)
	}
	return counts, rows.Err()
}

func (db *DB) analyticsPosts(ctx context.Context, from, to int64) ([]analogdb.AnalyticsPost, error) {
	query := fmt.Sprintf(`
		SELECT toInt64(post_id) AS pid, toInt64(%s) AS views, toInt64(%s)
		FROM %s
		WHERE %s AND post_id > 0
		GROUP BY pid
		ORDER BY views DESC, pid
		LIMIT ?`, viewsExpr, visitorsExpr, uiTable, pageViewCond)
	rows, done, err := db.query(ctx, query, from, to, topLimit)
	if err != nil {
		return nil, err
	}
	defer done()

	posts := make([]analogdb.AnalyticsPost, 0)
	for rows.Next() {
		var p analogdb.AnalyticsPost
		if err := rows.Scan(&p.PostID, &p.PageViews, &p.Visitors); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

// analyticsVitals reports p75 web vitals for the routes with the most samples
func (db *DB) analyticsVitals(ctx context.Context, from, to int64) ([]analogdb.AnalyticsVital, error) {
	query := fmt.Sprintf(`
		SELECT
			toString(route) AS r,
			quantileIf(0.75)(value, metric = 'LCP'),
			quantileIf(0.75)(value, metric = 'INP'),
			quantileIf(0.75)(value, metric = 'CLS'),
			toInt64(countIf(metric IN ('LCP', 'INP', 'CLS'))) AS samples
		FROM (
			SELECT
				event_id,
				route,
				JSONExtractString(props, 'metric') AS metric,
				JSONExtractFloat(props, 'value') AS value
			FROM %s
			WHERE event_name = 'web_vital' AND NOT is_bot AND %s
			LIMIT 1 BY event_id
		)
		WHERE r != ''
		GROUP BY r
		ORDER BY samples DESC, r
		LIMIT ?`, uiTable, uiWindow)
	rows, done, err := db.query(ctx, query, from, to, vitalsLimit)
	if err != nil {
		return nil, err
	}
	defer done()

	vitals := make([]analogdb.AnalyticsVital, 0)
	for rows.Next() {
		var v analogdb.AnalyticsVital
		var lcp, inp, cls float64
		if err := rows.Scan(&v.Route, &lcp, &inp, &cls, &v.Samples); err != nil {
			return nil, err
		}
		v.LCP, v.INP, v.CLS = finite(lcp), finite(inp), finite(cls)
		vitals = append(vitals, v)
	}
	return vitals, rows.Err()
}

func finite(f float64) *float64 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return nil
	}
	return &f
}
