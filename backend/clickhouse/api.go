package clickhouse

import (
	"context"
	"fmt"
	"strings"

	"github.com/evanofslack/analogdb"
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
