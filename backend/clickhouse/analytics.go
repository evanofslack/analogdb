package clickhouse

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/evanofslack/analogdb"
)

const (
	uiTable        = "ui_events"
	liveWindow     = 30 * time.Minute
	campaignsLimit = 10
	browsersLimit  = 10
	vitalsLimit    = 10
)

// visitor_id rotates daily, so a visitor on two days counts as two visitor-days
const (
	uiTimeExpr   = `received_ts`
	uiWindow     = `received_ts >= fromUnixTimestamp64Milli(toInt64(?), 'UTC') AND received_ts < fromUnixTimestamp64Milli(toInt64(?), 'UTC')`
	pageViewCond = `event_name = 'page_view' AND NOT is_bot AND ` + uiWindow
	viewsExpr    = `uniqExact(event_id)`
	visitorsExpr = `uniq((toDate(received_ts), visitor_id))`
	uiPostIDExpr = `if(post_id > 0, toInt64(post_id), toInt64OrZero(extract(path, '^/post/([0-9]+)')))`
	referrerCond = `referrer_host != '' AND referrer_host != 'analogdb.com' AND NOT endsWith(referrer_host, '.analogdb.com')`
)

func emptyAnalytics(r analogdb.TrafficRange, bucket string) *analogdb.Analytics {
	return &analogdb.Analytics{
		Range:     r,
		Bucket:    bucket,
		Series:    []analogdb.AnalyticsBucket{},
		Pages:     []analogdb.AnalyticsCount{},
		Posts:     []analogdb.AnalyticsPost{},
		Referrers: []analogdb.AnalyticsCount{},
		Campaigns: []analogdb.AnalyticsCampaign{},
		Devices:   []analogdb.AnalyticsCount{},
		Browsers:  []analogdb.AnalyticsCount{},
		Countries: []analogdb.AnalyticsCount{},
		Vitals:    []analogdb.AnalyticsVital{},
	}
}

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

	analytics := emptyAnalytics(r, bucketName)

	exists, err := db.tableExists(ctx, uiTable)
	if err != nil {
		db.logger.ErrorContext(ctx, "Fail analytics query", "step", "table", "error", err)
		return nil, fmt.Errorf("analytics table: %w", err)
	}
	if !exists {
		return analytics, nil
	}
	analytics.Available = true

	steps := []struct {
		name string
		run  func() error
	}{
		{"totals", func() error { return db.analyticsTotals(ctx, since, until, &analytics.Totals) }},
		{"previous", func() error { return db.analyticsTotals(ctx, before, since, &analytics.Previous) }},
		{"series", func() (err error) { analytics.Series, err = db.analyticsSeries(ctx, since, until, bucketExpr); return }},
		{"live", func() error { return db.analyticsLive(ctx, live, until, &analytics.Live) }},
		{"pages", func() (err error) {
			analytics.Pages, err = db.analyticsCounts(ctx, "route", "1 = 1", topLimit, since, until)
			return
		}},
		{"posts", func() (err error) { analytics.Posts, err = db.analyticsPosts(ctx, since, until); return }},
		{"referrers", func() (err error) {
			analytics.Referrers, err = db.analyticsCounts(ctx, "referrer_host", referrerCond, topLimit, since, until)
			return
		}},
		{"campaigns", func() (err error) { analytics.Campaigns, err = db.analyticsCampaigns(ctx, since, until); return }},
		{"devices", func() (err error) {
			analytics.Devices, err = db.analyticsCounts(ctx, "device_type", "1 = 1", topLimit, since, until)
			return
		}},
		{"browsers", func() (err error) {
			analytics.Browsers, err = db.analyticsCounts(ctx, "browser", "1 = 1", browsersLimit, since, until)
			return
		}},
		{"countries", func() (err error) {
			analytics.Countries, err = db.analyticsCounts(ctx, "country", "country != ''", topLimit, since, until)
			return
		}},
		{"vitals", func() (err error) {
			analytics.Vitals, err = db.analyticsVitals(ctx, since, until, analytics.Pages)
			return
		}},
	}
	for _, step := range steps {
		if err := step.run(); err != nil {
			db.logger.ErrorContext(ctx, "Fail analytics query", "step", step.name, "error", err)
			return nil, fmt.Errorf("analytics %s: %w", step.name, err)
		}
	}
	return analytics, nil
}

func (db *DB) tableExists(ctx context.Context, table string) (bool, error) {
	rows, done, err := db.query(ctx, `
		SELECT toInt64(count())
		FROM system.tables
		WHERE database = currentDatabase() AND name = ?`, table)
	if err != nil {
		return false, err
	}
	defer done()

	var count int64
	if rows.Next() {
		if err := rows.Scan(&count); err != nil {
			return false, err
		}
	}
	return count > 0, rows.Err()
}

func (db *DB) analyticsTotals(ctx context.Context, from, to int64, t *analogdb.AnalyticsTotals) error {
	query := fmt.Sprintf(`
		SELECT
			toInt64(uniqExactIf(event_id, NOT is_bot)),
			toInt64(uniqIf((toDate(received_ts), visitor_id), NOT is_bot)),
			toInt64(uniqExactIf(event_id, is_bot))
		FROM %s
		WHERE event_name = 'page_view' AND %s`, uiTable, uiWindow)
	rows, done, err := db.query(ctx, query, from, to)
	if err != nil {
		return err
	}
	defer done()

	var bots int64
	if rows.Next() {
		if err := rows.Scan(&t.PageViews, &t.VisitorDays, &bots); err != nil {
			return err
		}
	}
	if t.VisitorDays > 0 {
		t.ViewsPerVisitor = float64(t.PageViews) / float64(t.VisitorDays)
	}
	if all := t.PageViews + bots; all > 0 {
		t.BotShare = float64(bots) / float64(all)
	}
	return rows.Err()
}

func (db *DB) analyticsSeries(ctx context.Context, from, to int64, bucketExpr string) ([]analogdb.AnalyticsBucket, error) {
	query := fmt.Sprintf(`
		SELECT %s AS bucket, toInt64(%s), toInt64(%s)
		FROM %s
		WHERE %s
		GROUP BY bucket
		ORDER BY bucket`, bucketExpr, viewsExpr, visitorsExpr, uiTable, pageViewCond)
	rows, done, err := db.query(ctx, query, from, to)
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

func (db *DB) analyticsLive(ctx context.Context, from, to int64, live *analogdb.AnalyticsLive) error {
	query := fmt.Sprintf(`
		SELECT toInt64(%s), toInt64(%s)
		FROM %s
		WHERE %s`, viewsExpr, visitorsExpr, uiTable, pageViewCond)
	rows, done, err := db.query(ctx, query, from, to)
	if err != nil {
		return err
	}
	defer done()

	if rows.Next() {
		if err := rows.Scan(&live.PageViews, &live.Visitors); err != nil {
			return err
		}
	}
	return rows.Err()
}

// analyticsCounts groups page views by column, cond must not contain bind parameters
func (db *DB) analyticsCounts(ctx context.Context, column, cond string, limit int, from, to int64) ([]analogdb.AnalyticsCount, error) {
	query := fmt.Sprintf(`
		SELECT toString(%s) AS name, toInt64(%s) AS views, toInt64(%s)
		FROM %s
		WHERE %s AND %s
		GROUP BY name
		ORDER BY views DESC, name
		LIMIT ?`, column, viewsExpr, visitorsExpr, uiTable, pageViewCond, cond)
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
		SELECT %s AS pid, toInt64(%s) AS views, toInt64(%s)
		FROM %s
		WHERE %s AND (route = '/post/[id]' OR match(path, '^/post/[0-9]+'))
		GROUP BY pid
		HAVING pid > 0
		ORDER BY views DESC, pid
		LIMIT ?`, uiPostIDExpr, viewsExpr, visitorsExpr, uiTable, pageViewCond)
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

func (db *DB) analyticsCampaigns(ctx context.Context, from, to int64) ([]analogdb.AnalyticsCampaign, error) {
	query := fmt.Sprintf(`
		SELECT toString(utm_source) AS source, toString(utm_campaign) AS campaign, toInt64(%s) AS views, toInt64(%s)
		FROM %s
		WHERE %s AND (utm_source != '' OR utm_campaign != '')
		GROUP BY source, campaign
		ORDER BY views DESC, source, campaign
		LIMIT ?`, viewsExpr, visitorsExpr, uiTable, pageViewCond)
	rows, done, err := db.query(ctx, query, from, to, campaignsLimit)
	if err != nil {
		return nil, err
	}
	defer done()

	campaigns := make([]analogdb.AnalyticsCampaign, 0)
	for rows.Next() {
		var c analogdb.AnalyticsCampaign
		if err := rows.Scan(&c.Source, &c.Campaign, &c.PageViews, &c.Visitors); err != nil {
			return nil, err
		}
		campaigns = append(campaigns, c)
	}
	return campaigns, rows.Err()
}

// analyticsVitals reports p75 web vitals for the most viewed routes, in page order
func (db *DB) analyticsVitals(ctx context.Context, from, to int64, pages []analogdb.AnalyticsCount) ([]analogdb.AnalyticsVital, error) {
	routes := make([]string, 0, vitalsLimit)
	for _, p := range pages {
		if p.Name != "" && len(routes) < vitalsLimit {
			routes = append(routes, p.Name)
		}
	}
	vitals := make([]analogdb.AnalyticsVital, 0, len(routes))
	if len(routes) == 0 {
		return vitals, nil
	}

	query := fmt.Sprintf(`
		SELECT
			toString(route),
			quantileIf(0.75)(value, metric = 'LCP'),
			quantileIf(0.75)(value, metric = 'INP'),
			quantileIf(0.75)(value, metric = 'CLS'),
			toInt64(countIf(metric IN ('LCP', 'INP', 'CLS')))
		FROM (
			SELECT
				event_id,
				route,
				JSONExtractString(props, 'metric') AS metric,
				JSONExtractFloat(props, 'value') AS value
			FROM %s
			WHERE event_name = 'web_vital' AND NOT is_bot AND %s AND has(?, toString(route))
			LIMIT 1 BY event_id
		)
		GROUP BY route`, uiTable, uiWindow)
	rows, done, err := db.query(ctx, query, from, to, routes)
	if err != nil {
		return nil, err
	}
	defer done()

	byRoute := map[string]analogdb.AnalyticsVital{}
	for rows.Next() {
		var v analogdb.AnalyticsVital
		var lcp, inp, cls float64
		if err := rows.Scan(&v.Route, &lcp, &inp, &cls, &v.Samples); err != nil {
			return nil, err
		}
		v.LCP, v.INP, v.CLS = finite(lcp), finite(inp), finite(cls)
		byRoute[v.Route] = v
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, route := range routes {
		if v, ok := byRoute[route]; ok {
			vitals = append(vitals, v)
		}
	}
	return vitals, nil
}

func finite(f float64) *float64 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return nil
	}
	return &f
}
