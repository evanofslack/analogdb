package clickhouse

import (
	"context"
	"fmt"
	"math"

	"github.com/evanofslack/analogdb"
)

func (db *DB) Audit(ctx context.Context, filter *analogdb.AuditFilter) ([]*analogdb.AuditEntry, error) {
	db.logger.DebugContext(ctx, "Starting audit query")
	defer db.logger.DebugContext(ctx, "Finished audit query")

	before := int64(math.MaxInt64)
	if filter.Before != nil {
		before = *filter.Before
	}

	query := fmt.Sprintf(`
		SELECT %s, start_time, method, path, response_code, remote_ip, user_agent, request_id, %s
		FROM %s
		WHERE method IN ('POST', 'PUT', 'PATCH', 'DELETE')
			AND (authorized OR response_code = 401)
			AND start_time < ?
		ORDER BY start_time DESC
		LIMIT ?`, timeExpr, clientExpr, db.table)
	rows, cancel, err := db.query(ctx, query, before, filter.Limit)
	if err != nil {
		db.logger.ErrorContext(ctx, "Fail audit query", "error", err)
		return nil, err
	}
	defer cancel()
	defer rows.Close()

	entries := make([]*analogdb.AuditEntry, 0)
	for rows.Next() {
		var e analogdb.AuditEntry
		if err := rows.Scan(&e.Time, &e.StartMs, &e.Method, &e.Path, &e.Status, &e.RemoteIP, &e.UserAgent, &e.RequestID, &e.Client); err != nil {
			return nil, err
		}
		entries = append(entries, &e)
	}
	return entries, rows.Err()
}
