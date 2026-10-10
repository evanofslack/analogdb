package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/evanofslack/analogdb"
	"golang.org/x/sync/errgroup"
)

const (
	topLimit    = 20
	errorsLimit = 50
)

var clientExpr = fmt.Sprintf(
	`multiIf(startsWith(user_agent, '%s'), 'web', startsWith(user_agent, '%s'), 'scraper', 'direct')`,
	analogdb.WebUserAgentPrefix, analogdb.ScraperUserAgentPrefix,
)

// no ? in these, the driver treats it as a bind parameter
const (
	routeExpr    = `replaceRegexpAll(path, '/[0-9]+', '/{id}')`
	timeExpr     = `toDateTime(intDiv(start_time, 1000), 'UTC')`
	boundExpr    = `toDateTime(intDiv(toInt64(?), 1000), 'UTC')`
	botTokenExpr = `extract(lower(user_agent), '([a-z0-9._-]*(bot|crawler|spider|externalagent|externalhit)[a-z0-9._-]*)')`
	toolExpr     = `match(lower(user_agent), '^(python|curl|wget|go-http-client|axios|node-fetch|node|undici|okhttp|java|scrapy|httpx|aiohttp|libwww-perl|apache-httpclient|postmanruntime|insomnia|ruby|php|dart|reqwest|deno|bun)')`
)

// bots are named by their token in the user agent, tools by the token before the first slash
var (
	callerKindExpr = fmt.Sprintf(
		`multiIf(user_agent = '', 'empty', %s != '', 'bot', %s, 'tool', 'browser')`,
		botTokenExpr, toolExpr,
	)
	callerNameExpr = fmt.Sprintf(
		`multiIf(%[1]s = 'bot', substring(user_agent, positionCaseInsensitive(user_agent, %[2]s), length(%[2]s)), %[1]s = 'tool', extract(user_agent, '^([^/ ]+)'), user_agent)`,
		callerKindExpr, botTokenExpr,
	)
)

func bucketFor(r analogdb.TrafficRange, expr string) (string, string) {
	if r == analogdb.TrafficMonth {
		return "day", "toStartOfDay(" + expr + ")"
	}
	return "hour", "toStartOfHour(" + expr + ")"
}

// fillFor zero fills buckets from since up to until, both bound as unix milliseconds
func fillFor(r analogdb.TrafficRange) string {
	name, bound := bucketFor(r, boundExpr)
	unit := strings.ToUpper(name)
	return fmt.Sprintf(`WITH FILL FROM %s TO %s + INTERVAL 1 %s STEP INTERVAL 1 %s`, bound, bound, unit, unit)
}

func (db *DB) Traffic(ctx context.Context, r analogdb.TrafficRange) (*analogdb.Traffic, error) {
	db.logger.DebugContext(ctx, "Starting traffic query", "range", r)
	defer db.logger.DebugContext(ctx, "Finished traffic query")

	window, ok := r.Duration()
	if !ok {
		return nil, &analogdb.Error{Code: analogdb.ERRBADREQUEST, Message: fmt.Sprintf("invalid range: %s", r)}
	}
	now := time.Now()
	until := now.UnixMilli()
	since := now.Add(-window).UnixMilli()
	before := now.Add(-2 * window).UnixMilli()
	bucketName, bucketExpr := bucketFor(r, timeExpr)
	fill := fillFor(r)

	traffic := &analogdb.Traffic{Range: r, Bucket: bucketName}
	summary := &traffic.Summary
	g, gctx := errgroup.WithContext(ctx)

	steps := []struct {
		name string
		run  func() error
	}{
		{"summary", func() error { return db.trafficSummary(gctx, before, since, until, summary) }},
		{"page views", func() error { return db.trafficPageViews(gctx, before, since, until, summary) }},
		{"series", func() (err error) {
			traffic.Series, err = db.trafficSeries(gctx, since, until, bucketExpr, fill)
			return
		}},
		{"callers", func() (err error) { traffic.Callers, err = db.trafficCallers(gctx, since, until); return }},
		{"routes", func() (err error) { traffic.Routes, err = db.trafficRoutes(gctx, since, until); return }},
		{"errors by status", func() (err error) {
			traffic.Errors.ByStatus, err = db.trafficErrorsByStatus(gctx, since, until)
			return
		}},
		{"errors", func() (err error) { traffic.Errors.Recent, err = db.trafficErrors(gctx, since, until); return }},
	}
	for _, step := range steps {
		g.Go(func() error {
			if err := step.run(); err != nil {
				if gctx.Err() == nil {
					db.logger.ErrorContext(ctx, "Fail traffic query", "step", step.name, "error", err)
				}
				return fmt.Errorf("traffic %s: %w", step.name, err)
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return traffic, nil
}

// trafficSummary reads the current window and the one before it in one pass
func (db *DB) trafficSummary(ctx context.Context, before, since, until int64, s *analogdb.TrafficSummary) error {
	query := fmt.Sprintf(`
		SELECT
			toInt64(countIf(cur)),
			toInt64(countIf(cur AND client = 'web')),
			toInt64(countIf(cur AND client = 'scraper')),
			toInt64(countIf(cur AND client = 'direct')),
			toInt64(countIf(cur AND client = 'direct' AND kind = 'bot')),
			toInt64(countIf(cur AND response_code >= 400 AND response_code < 500)),
			toInt64(countIf(cur AND response_code >= 500)),
			ifNotFinite(quantileIf(0.95)(request_time_ms, cur), 0),
			toInt64(countIf(NOT cur)),
			toInt64(countIf(NOT cur AND client = 'web')),
			toInt64(countIf(NOT cur AND client = 'scraper')),
			toInt64(countIf(NOT cur AND client = 'direct')),
			toInt64(countIf(NOT cur AND client = 'direct' AND kind = 'bot')),
			toInt64(countIf(NOT cur AND response_code >= 400 AND response_code < 500)),
			toInt64(countIf(NOT cur AND response_code >= 500)),
			ifNotFinite(quantileIf(0.95)(request_time_ms, NOT cur), 0)
		FROM (
			SELECT
				start_time >= toInt64(?) AS cur,
				%s AS client,
				%s AS kind,
				response_code,
				request_time_ms
			FROM %s
			WHERE start_time >= ? AND start_time < ?
		)`, clientExpr, callerKindExpr, db.table)
	rows, done, err := db.query(ctx, query, since, before, until)
	if err != nil {
		return err
	}
	defer done()

	cur, prev := &s.Current, &s.Previous
	if rows.Next() {
		if err := rows.Scan(
			&cur.Requests, &cur.Web, &cur.Scraper, &cur.Direct, &cur.Bots, &cur.Status4, &cur.Status5, &cur.P95Ms,
			&prev.Requests, &prev.Web, &prev.Scraper, &prev.Direct, &prev.Bots, &prev.Status4, &prev.Status5, &prev.P95Ms,
		); err != nil {
			return err
		}
	}
	return rows.Err()
}

// trafficPageViews counts human page views from UI events in the current and previous window
func (db *DB) trafficPageViews(ctx context.Context, before, since, until int64, s *analogdb.TrafficSummary) error {
	query := fmt.Sprintf(`
		WITH fromUnixTimestamp64Milli(toInt64(?), 'UTC') AS since_ts
		SELECT
			toInt64(uniqExactIf(event_id, received_ts >= since_ts)),
			toInt64(uniqExactIf(event_id, received_ts < since_ts))
		FROM %s
		WHERE %s`, uiTable, pageViewCond)
	rows, done, err := db.query(ctx, query, since, before, until)
	if err != nil {
		return err
	}
	defer done()

	if rows.Next() {
		if err := rows.Scan(&s.Current.PageViews, &s.Previous.PageViews); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (db *DB) trafficSeries(ctx context.Context, since, until int64, bucketExpr, fill string) ([]analogdb.TrafficBucket, error) {
	query := fmt.Sprintf(`
		SELECT
			%s AS bucket,
			toInt64(countIf(client = 'web')),
			toInt64(countIf(client = 'scraper')),
			toInt64(countIf(client = 'direct')),
			toInt64(countIf(response_code >= 400 AND response_code < 500)),
			toInt64(countIf(response_code >= 500))
		FROM (SELECT start_time, response_code, %s AS client FROM %s WHERE start_time >= ? AND start_time < ?)
		GROUP BY bucket
		ORDER BY bucket %s`, bucketExpr, clientExpr, db.table, fill)
	rows, done, err := db.query(ctx, query, since, until, since, until)
	if err != nil {
		return nil, err
	}
	defer done()

	series := make([]analogdb.TrafficBucket, 0)
	for rows.Next() {
		var b analogdb.TrafficBucket
		if err := rows.Scan(&b.Time, &b.Web, &b.Scraper, &b.Direct, &b.Status4, &b.Status5); err != nil {
			return nil, err
		}
		series = append(series, b)
	}
	return series, rows.Err()
}

// trafficCallers groups direct requests, bots and tools by family and browsers by full user agent
func (db *DB) trafficCallers(ctx context.Context, since, until int64) ([]analogdb.TrafficCaller, error) {
	query := fmt.Sprintf(`
		SELECT name, kind, toInt64(count()) AS requests, toInt64(uniq(remote_ip))
		FROM (
			SELECT %s AS kind, %s AS name, remote_ip
			FROM %s
			WHERE start_time >= ? AND start_time < ? AND %s = 'direct'
		)
		GROUP BY kind, name
		ORDER BY requests DESC, kind, name
		LIMIT ?`, callerKindExpr, callerNameExpr, db.table, clientExpr)
	rows, done, err := db.query(ctx, query, since, until, topLimit)
	if err != nil {
		return nil, err
	}
	defer done()

	callers := make([]analogdb.TrafficCaller, 0)
	for rows.Next() {
		var c analogdb.TrafficCaller
		if err := rows.Scan(&c.Name, &c.Kind, &c.Requests, &c.IPs); err != nil {
			return nil, err
		}
		callers = append(callers, c)
	}
	return callers, rows.Err()
}

func (db *DB) trafficRoutes(ctx context.Context, since, until int64) ([]analogdb.TrafficRoute, error) {
	query := fmt.Sprintf(`
		SELECT
			%s AS route,
			toInt64(count()),
			toInt64(sum(request_time_ms)) AS total_ms,
			quantile(0.5)(request_time_ms),
			quantile(0.95)(request_time_ms),
			toInt64(countIf(response_code >= 400 AND response_code < 500)),
			toInt64(countIf(response_code >= 500))
		FROM %s
		WHERE start_time >= ? AND start_time < ?
		GROUP BY route
		ORDER BY total_ms DESC, route
		LIMIT ?`, routeExpr, db.table)
	rows, done, err := db.query(ctx, query, since, until, topLimit)
	if err != nil {
		return nil, err
	}
	defer done()

	routes := make([]analogdb.TrafficRoute, 0)
	for rows.Next() {
		var route analogdb.TrafficRoute
		if err := rows.Scan(&route.Route, &route.Requests, &route.TotalMs, &route.P50Ms, &route.P95Ms, &route.Status4, &route.Status5); err != nil {
			return nil, err
		}
		routes = append(routes, route)
	}
	return routes, rows.Err()
}

func (db *DB) trafficErrorsByStatus(ctx context.Context, since, until int64) ([]analogdb.TrafficStatus, error) {
	query := fmt.Sprintf(`
		SELECT response_code, %s AS route, toInt64(count()) AS requests
		FROM %s
		WHERE start_time >= ? AND start_time < ? AND response_code >= 400
		GROUP BY response_code, route
		ORDER BY requests DESC, response_code, route
		LIMIT ?`, routeExpr, db.table)
	rows, done, err := db.query(ctx, query, since, until, topLimit)
	if err != nil {
		return nil, err
	}
	defer done()

	statuses := make([]analogdb.TrafficStatus, 0)
	for rows.Next() {
		var s analogdb.TrafficStatus
		if err := rows.Scan(&s.Status, &s.Route, &s.Requests); err != nil {
			return nil, err
		}
		statuses = append(statuses, s)
	}
	return statuses, rows.Err()
}

func (db *DB) trafficErrors(ctx context.Context, since, until int64) ([]analogdb.TrafficError, error) {
	query := fmt.Sprintf(`
		SELECT %s, method, path, response_code, request_id
		FROM %s
		WHERE start_time >= ? AND start_time < ? AND response_code >= 500
		ORDER BY start_time DESC
		LIMIT ?`, timeExpr, db.table)
	rows, done, err := db.query(ctx, query, since, until, errorsLimit)
	if err != nil {
		return nil, err
	}
	defer done()

	errs := make([]analogdb.TrafficError, 0)
	for rows.Next() {
		var e analogdb.TrafficError
		if err := rows.Scan(&e.Time, &e.Method, &e.Path, &e.Status, &e.RequestID); err != nil {
			return nil, err
		}
		errs = append(errs, e)
	}
	return errs, rows.Err()
}
