package postgres

import (
	"context"
	"testing"
)

const (
	totalAuthors = 4864
	author1      = "u/photocactus"
	author2      = "u/_markforbes_"
	author3      = "u/26Point2"
)

func TestFindAuthors(t *testing.T) {
	t.Run("Number of authors", func(t *testing.T) {
		db := mustOpen(t)
		defer mustClose(t, db)
		as := NewAuthorService(db)

		authors, err := as.FindAuthors(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		numAuthors := len(authors)
		if numAuthors != totalAuthors {
			t.Fatalf("wrong number of total authors, wanted %d, got %d", totalAuthors, numAuthors)
		}
	})
	t.Run("Authors are correct", func(t *testing.T) {
		db := mustOpen(t)
		defer mustClose(t, db)
		as := NewAuthorService(db)

		authors, err := as.FindAuthors(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if authors[0] != author1 || authors[1] != author2 || authors[2] != author3 {
			t.Fatalf("wrong values of authors, want %s, %s, %s, got %s, %s, %s", author1, author2, author3, authors[0], authors[1], authors[2])
		}
	})
}
