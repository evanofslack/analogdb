package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/evanofslack/analogdb"
)

const (
	topLimit    = 20
	errorsLimit = 50
)

var clientExpr = fmt.Sprintf(
	`multiIf(startsWith(user_agent, '%s'), 'web', startsWith(user_agent, '%s'), 'scraper', 'other')`,
	analogdb.WebUserAgentPrefix, analogdb.ScraperUserAgentPrefix,
)

// no ? in these, the driver treats it as a bind parameter
const (
	routeExpr  = `replaceRegexpAll(path, '/[0-9]+', '/{id}')`
	postIDExpr = `toInt64OrZero(extract(replaceRegexpOne(path, '^/v1/', '/'), '^/post/([0-9]+)'))`
	legacyExpr = `match(path, '^/(posts|post|ids|films|film|cameras|camera|authors|keywords|scrape|encode)(/|$)')`
	timeExpr   = `toDateTime(intDiv(start_time, 1000), 'UTC')`
)

func bucketFor(r analogdb.TrafficRange, expr string) (string, string) {
	if r == analogdb.TrafficDay {
		return "hour", "toStartOfHour(" + expr + ")"
	}
	return "day", "toStartOfDay(" + expr + ")"
}

func (db *DB) Traffic(ctx context.Context, r analogdb.TrafficRange) (*analogdb.Traffic, error) {
	db.logger.DebugContext(ctx, "Starting traffic query", "range", r)
	defer db.logger.DebugContext(ctx, "Finished traffic query")

	window, ok := r.Duration()
	if !ok {
		return nil, &analogdb.Error{Code: analogdb.ERRBADREQUEST, Message: fmt.Sprintf("invalid range: %s", r)}
	}
	since := time.Now().Add(-window).UnixMilli()
	bucketName, bucketExpr := bucketFor(r, timeExpr)

	traffic := &analogdb.Traffic{Range: r, Bucket: bucketName}

	steps := []struct {
		name string
		run  func() error
	}{
		{"series", func() (err error) { traffic.Series, err = db.trafficSeries(ctx, since, bucketExpr); return }},
		{"totals", func() error { return db.trafficTotals(ctx, since, &traffic.Totals) }},
		{"routes", func() (err error) { traffic.Routes, err = db.trafficRoutes(ctx, since); return }},
		{"posts", func() (err error) { traffic.Posts, err = db.trafficPosts(ctx, since); return }},
		{"legacy", func() (err error) { traffic.Legacy, err = db.trafficLegacy(ctx, since); return }},
		{"params", func() (err error) { traffic.Params, err = db.trafficParams(ctx, since); return }},
		{"user agents", func() (err error) { traffic.UserAgents, err = db.trafficUserAgents(ctx, since); return }},
		{"ips", func() (err error) { traffic.IPs, err = db.trafficIPs(ctx, since); return }},
		{"errors", func() (err error) { traffic.Errors, err = db.trafficErrors(ctx, since); return }},
	}
	for _, step := range steps {
		if err := step.run(); err != nil {
			db.logger.ErrorContext(ctx, "Fail traffic query", "step", step.name, "error", err)
			return nil, fmt.Errorf("traffic %s: %w", step.name, err)
		}
	}
	return traffic, nil
}

func (db *DB) trafficSeries(ctx context.Context, since int64, bucketExpr string) ([]analogdb.TrafficBucket, error) {
	query := fmt.Sprintf(`
		SELECT
			%s AS bucket,
			toInt64(countIf(client = 'web')),
			toInt64(countIf(client = 'scraper')),
			toInt64(countIf(client = 'other')),
			toInt64(countIf(response_code >= 400 AND response_code < 500)),
			toInt64(countIf(response_code >= 500))
		FROM (SELECT start_time, response_code, %s AS client FROM %s WHERE start_time >= ?)
		GROUP BY bucket
		ORDER BY bucket`, bucketExpr, clientExpr, db.table)
	rows, done, err := db.query(ctx, query, since)
	if err != nil {
		return nil, err
	}
	defer done()

	series := make([]analogdb.TrafficBucket, 0)
	for rows.Next() {
		var b analogdb.TrafficBucket
		if err := rows.Scan(&b.Time, &b.Web, &b.Scraper, &b.Other, &b.Status4, &b.Status5); err != nil {
			return nil, err
		}
		series = append(series, b)
	}
	return series, rows.Err()
}

func (db *DB) trafficTotals(ctx context.Context, since int64, t *analogdb.TrafficTotals) error {
	query := fmt.Sprintf(`
		SELECT
			toInt64(count()),
			toInt64(uniq(remote_ip)),
			toInt64(countIf(response_code >= 200 AND response_code < 300)),
			toInt64(countIf(response_code >= 300 AND response_code < 400)),
			toInt64(countIf(response_code >= 400 AND response_code < 500)),
			toInt64(countIf(response_code >= 500))
		FROM %s
		WHERE start_time >= ?`, db.table)
	rows, done, err := db.query(ctx, query, since)
	if err != nil {
		return err
	}
	defer done()

	if rows.Next() {
		if err := rows.Scan(&t.Requests, &t.UniqueIPs, &t.Status2, &t.Status3, &t.Status4, &t.Status5); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (db *DB) trafficRoutes(ctx context.Context, since int64) ([]analogdb.TrafficRoute, error) {
	query := fmt.Sprintf(`
		SELECT
			%s AS route,
			toInt64(count()) AS requests,
			quantile(0.5)(request_time_ms),
			quantile(0.95)(request_time_ms),
			toInt64(countIf(response_code >= 500))
		FROM %s
		WHERE start_time >= ?
		GROUP BY route
		ORDER BY requests DESC, route
		LIMIT ?`, routeExpr, db.table)
	rows, done, err := db.query(ctx, query, since, topLimit)
	if err != nil {
		return nil, err
	}
	defer done()

	routes := make([]analogdb.TrafficRoute, 0)
	for rows.Next() {
		var route analogdb.TrafficRoute
		if err := rows.Scan(&route.Route, &route.Requests, &route.P50Ms, &route.P95Ms, &route.Errors); err != nil {
			return nil, err
		}
		routes = append(routes, route)
	}
	return routes, rows.Err()
}

func (db *DB) trafficPosts(ctx context.Context, since int64) ([]analogdb.TrafficPost, error) {
	query := fmt.Sprintf(`
		SELECT %s AS post_id, toInt64(count()) AS requests
		FROM %s
		WHERE start_time >= ? AND post_id > 0
		GROUP BY post_id
		ORDER BY requests DESC, post_id
		LIMIT ?`, postIDExpr, db.table)
	rows, done, err := db.query(ctx, query, since, topLimit)
	if err != nil {
		return nil, err
	}
	defer done()

	posts := make([]analogdb.TrafficPost, 0)
	for rows.Next() {
		var p analogdb.TrafficPost
		if err := rows.Scan(&p.PostID, &p.Requests); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

func (db *DB) trafficLegacy(ctx context.Context, since int64) ([]analogdb.TrafficLegacy, error) {
	query := fmt.Sprintf(`
		SELECT %s AS client, toInt64(count()), toInt64(countIf(%s))
		FROM %s
		WHERE start_time >= ?
		GROUP BY client
		ORDER BY client`, clientExpr, legacyExpr, db.table)
	rows, done, err := db.query(ctx, query, since)
	if err != nil {
		return nil, err
	}
	defer done()

	legacy := make([]analogdb.TrafficLegacy, 0)
	for rows.Next() {
		var l analogdb.TrafficLegacy
		if err := rows.Scan(&l.Client, &l.Requests, &l.Legacy); err != nil {
			return nil, err
		}
		legacy = append(legacy, l)
	}
	return legacy, rows.Err()
}

func (db *DB) trafficParams(ctx context.Context, since int64) ([]analogdb.TrafficCount, error) {
	query := fmt.Sprintf(`
		SELECT arrayJoin(extractURLParameterNames(url)) AS param, toInt64(count()) AS requests
		FROM %s
		WHERE start_time >= ? AND path IN ('/v1/posts', '/posts')
		GROUP BY param
		ORDER BY requests DESC, param
		LIMIT ?`, db.table)
	return db.trafficCounts(ctx, query, false, since, topLimit)
}

func (db *DB) trafficUserAgents(ctx context.Context, since int64) ([]analogdb.TrafficCount, error) {
	query := fmt.Sprintf(`
		SELECT user_agent, %s AS client, toInt64(count()) AS requests
		FROM %s
		WHERE start_time >= ?
		GROUP BY user_agent, client
		ORDER BY requests DESC, user_agent
		LIMIT ?`, clientExpr, db.table)
	return db.trafficCounts(ctx, query, true, since, topLimit)
}

func (db *DB) trafficIPs(ctx context.Context, since int64) ([]analogdb.TrafficCount, error) {
	query := fmt.Sprintf(`
		SELECT remote_ip, 'other' AS client, toInt64(count()) AS requests
		FROM %s
		WHERE start_time >= ? AND %s = 'other'
		GROUP BY remote_ip
		ORDER BY requests DESC, remote_ip
		LIMIT ?`, db.table, clientExpr)
	return db.trafficCounts(ctx, query, true, since, topLimit)
}

func (db *DB) trafficCounts(ctx context.Context, query string, withClient bool, args ...any) ([]analogdb.TrafficCount, error) {
	rows, done, err := db.query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer done()

	counts := make([]analogdb.TrafficCount, 0)
	for rows.Next() {
		var c analogdb.TrafficCount
		dest := []any{&c.Name, &c.Requests}
		if withClient {
			dest = []any{&c.Name, &c.Client, &c.Requests}
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		counts = append(counts, c)
	}
	return counts, rows.Err()
}

func (db *DB) trafficErrors(ctx context.Context, since int64) ([]analogdb.TrafficError, error) {
	query := fmt.Sprintf(`
		SELECT %s, method, path, response_code, request_id
		FROM %s
		WHERE start_time >= ? AND response_code >= 500
		ORDER BY start_time DESC
		LIMIT ?`, timeExpr, db.table)
	rows, done, err := db.query(ctx, query, since, errorsLimit)
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
