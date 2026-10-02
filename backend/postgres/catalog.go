package postgres

import (
	"context"

	"github.com/evanofslack/analogdb"
)

func catalogKey(make, name string) string {
	return make + "\x00" + name
}

// findCatalogTopPosts runs a top posts query and groups the posts by make and name
func (db *DB) findCatalogTopPosts(ctx context.Context, query string, args ...any) (map[string][]analogdb.CatalogPost, error) {
	rows, err := db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	top := make(map[string][]analogdb.CatalogPost)
	for rows.Next() {
		var make, name, title, lowUrl, medUrl string
		var id, score, lowWidth, lowHeight, medWidth, medHeight int

		if err := rows.Scan(&make, &name, &id, &title, &score, &lowUrl, &lowWidth, &lowHeight, &medUrl, &medWidth, &medHeight); err != nil {
			return nil, err
		}

		key := catalogKey(make, name)
		top[key] = append(top[key], analogdb.CatalogPost{
			Id:    id,
			Title: title,
			Score: score,
			Images: []analogdb.Image{
				{Label: "low", Url: lowUrl, Width: lowWidth, Height: lowHeight},
				{Label: "medium", Url: medUrl, Width: medWidth, Height: medHeight},
			},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return top, nil
}
