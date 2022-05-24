package postgres

import (
	"context"
	"database/sql"
	"testing"

	"github.com/evanofslack/analogdb"
)

// number of posts matching each criteria in test DB
const (
	totalPosts     = 51
	totalNsfw      = 4
	totalGrayscale = 7
	totalSprocket  = 2
)

func TestFindPosts(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		ctx, tx := setupTx(t)
		if posts, count, err := findPosts(ctx, tx, analogdb.PostFilter{}, "time"); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), totalPosts; got != want {
			t.Fatalf("length of posts = %v, want %v", got, want)
		} else if got, want := count, totalPosts; got != want {
			t.Fatalf("total count = %v, want %v", got, want)
		}
	})

	t.Run("Limit", func(t *testing.T) {
		ctx, tx := setupTx(t)
		limit := 20
		if posts, count, err := findPosts(ctx, tx, analogdb.PostFilter{Limit: &limit}, "time"); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), limit; got != want {
			t.Fatalf("length of posts = %v, want %v", got, want)
		} else if got, want := count, totalPosts; got != want {
			t.Fatalf("total count = %v, want %v", got, want)
		}
	})

	t.Run("NoNSFW", func(t *testing.T) {
		ctx, tx := setupTx(t)
		nsfw := false
		if posts, count, err := findPosts(ctx, tx, analogdb.PostFilter{Nsfw: &nsfw}, "time"); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), totalPosts-totalNsfw; got != want {
			t.Fatalf("length of posts = %v, want %v", got, want)
		} else if got, want := count, totalPosts-totalNsfw; got != want {
			t.Fatalf("total count = %v, want %v", got, want)
		}
	})

	t.Run("OnlyNSFW", func(t *testing.T) {
		ctx, tx := setupTx(t)
		nsfw := true
		if posts, count, err := findPosts(ctx, tx, analogdb.PostFilter{Nsfw: &nsfw}, "time"); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), totalNsfw; got != want {
			t.Fatalf("length of posts = %v, want %v", got, want)
		} else if got, want := count, totalNsfw; got != want {
			t.Fatalf("total count = %v, want %v", got, want)
		}
	})

	t.Run("NoBW", func(t *testing.T) {
		ctx, tx := setupTx(t)
		grayscale := false
		if posts, count, err := findPosts(ctx, tx, analogdb.PostFilter{Grayscale: &grayscale}, "time"); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), totalPosts-totalGrayscale; got != want {
			t.Fatalf("length of posts = %v, want %v", got, want)
		} else if got, want := count, totalPosts-totalGrayscale; got != want {
			t.Fatalf("total count = %v, want %v", got, want)
		}
	})

	t.Run("OnlyBW", func(t *testing.T) {
		ctx, tx := setupTx(t)
		grayscale := true
		if posts, count, err := findPosts(ctx, tx, analogdb.PostFilter{Grayscale: &grayscale}, "time"); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), totalGrayscale; got != want {
			t.Fatalf("length of posts = %v, want %v", got, want)
		} else if got, want := count, totalGrayscale; got != want {
			t.Fatalf("total count = %v, want %v", got, want)
		}
	})

	t.Run("NoSprocket", func(t *testing.T) {
		ctx, tx := setupTx(t)
		sprocket := false
		if posts, count, err := findPosts(ctx, tx, analogdb.PostFilter{Sprocket: &sprocket}, "time"); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), totalPosts-totalSprocket; got != want {
			t.Fatalf("length of posts = %v, want %v", got, want)
		} else if got, want := count, totalPosts-totalSprocket; got != want {
			t.Fatalf("total count = %v, want %v", got, want)
		}
	})

	t.Run("OnlySprocket", func(t *testing.T) {
		ctx, tx := setupTx(t)
		sprocket := true
		if posts, count, err := findPosts(ctx, tx, analogdb.PostFilter{Sprocket: &sprocket}, "time"); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), totalSprocket; got != want {
			t.Fatalf("length of posts = %v, want %v", got, want)
		} else if got, want := count, totalSprocket; got != want {
			t.Fatalf("total count = %v, want %v", got, want)
		}
	})

}

func TestLatestPost(t *testing.T) {
	db := MustOpen(t)
	defer MustClose(t, db)
	psql := NewPostService(db)

	limit := 20
	filter := analogdb.PostFilter{
		Limit: &limit,
	}

	if posts, count, err := psql.LatestPosts(context.Background(), filter); err != nil {
		t.Fatal(err)
	} else if got, want := len(posts), limit; got != want {
		t.Fatalf("length of posts = %v, want %v", got, want)
	} else if got, want := count, 51; got != want {
		t.Fatalf("total count = %v, want %v", got, want)
	}

}

func setupTx(t *testing.T) (context.Context, *sql.Tx) {
	t.Helper()
	ctx := context.Background()
	db := MustOpen(t)
	defer MustClose(t, db)
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	return ctx, tx
}
