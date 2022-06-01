package postgres

import (
	"context"
	"database/sql"
	"testing"

	"github.com/evanofslack/analogdb"
)

const (
	// number of posts matching each query from test DB
	totalPosts     = 51
	totalNsfw      = 4
	totalGrayscale = 7
	totalSprocket  = 2
	totalPortra    = 17
)

var (
	// reusable general filters
	limit       = 20
	limitFilter = &analogdb.PostFilter{Limit: &limit}
	nilFilter   = &analogdb.PostFilter{}

	// sample post
	postID     = 2066
	postTitle  = "Up on Melancholy Hill [Canon TLB / 28-55mm 3.5 / Portra 400]"
	postAuthor = "u/sunnyintheoffice"
)

func TestFindPosts(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		ctx, tx := setupTx(t)
		if posts, count, err := findPosts(ctx, tx, nilFilter); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), totalPosts; got != want {
			t.Fatalf("length of postsG %v, want %v", got, want)
		} else if got, want := count, totalPosts; got != want {
			t.Fatalf("total count %v, want %v", got, want)
		}
	})

	t.Run("Limit", func(t *testing.T) {
		ctx, tx := setupTx(t)
		if posts, count, err := findPosts(ctx, tx, limitFilter); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), limit; got != want {
			t.Fatalf("length of posts %v, want %v", got, want)
		} else if got, want := count, totalPosts; got != want {
			t.Fatalf("total count %v, want %v", got, want)
		}
	})

	t.Run("NoNSFW", func(t *testing.T) {
		ctx, tx := setupTx(t)
		nsfw := false
		if posts, count, err := findPosts(ctx, tx, &analogdb.PostFilter{Nsfw: &nsfw}); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), totalPosts-totalNsfw; got != want {
			t.Fatalf("length of posts %v, want %v", got, want)
		} else if got, want := count, totalPosts-totalNsfw; got != want {
			t.Fatalf("total count %v, want %v", got, want)
		}
	})

	t.Run("OnlyNSFW", func(t *testing.T) {
		ctx, tx := setupTx(t)
		nsfw := true
		if posts, count, err := findPosts(ctx, tx, &analogdb.PostFilter{Nsfw: &nsfw}); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), totalNsfw; got != want {
			t.Fatalf("length of posts %v, want %v", got, want)
		} else if got, want := count, totalNsfw; got != want {
			t.Fatalf("total count %v, want %v", got, want)
		}
	})

	t.Run("NoBW", func(t *testing.T) {
		ctx, tx := setupTx(t)
		grayscale := false
		if posts, count, err := findPosts(ctx, tx, &analogdb.PostFilter{Grayscale: &grayscale}); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), totalPosts-totalGrayscale; got != want {
			t.Fatalf("length of posts %v, want %v", got, want)
		} else if got, want := count, totalPosts-totalGrayscale; got != want {
			t.Fatalf("total count %v, want %v", got, want)
		}
	})

	t.Run("OnlyBW", func(t *testing.T) {
		ctx, tx := setupTx(t)
		grayscale := true
		if posts, count, err := findPosts(ctx, tx, &analogdb.PostFilter{Grayscale: &grayscale}); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), totalGrayscale; got != want {
			t.Fatalf("length of posts %v, want %v", got, want)
		} else if got, want := count, totalGrayscale; got != want {
			t.Fatalf("total count %v, want %v", got, want)
		}
	})

	t.Run("NoSprocket", func(t *testing.T) {
		ctx, tx := setupTx(t)
		sprocket := false
		if posts, count, err := findPosts(ctx, tx, &analogdb.PostFilter{Sprocket: &sprocket}); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), totalPosts-totalSprocket; got != want {
			t.Fatalf("length of posts %v, want %v", got, want)
		} else if got, want := count, totalPosts-totalSprocket; got != want {
			t.Fatalf("total count %v, want %v", got, want)
		}
	})

	t.Run("OnlySprocket", func(t *testing.T) {
		ctx, tx := setupTx(t)
		sprocket := true
		if posts, count, err := findPosts(ctx, tx, &analogdb.PostFilter{Sprocket: &sprocket}); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), totalSprocket; got != want {
			t.Fatalf("length of posts %v, want %v", got, want)
		} else if got, want := count, totalSprocket; got != want {
			t.Fatalf("total count %v, want %v", got, want)
		}
	})

	t.Run("ByAuthor", func(t *testing.T) {
		ctx, tx := setupTx(t)
		if posts, count, err := findPosts(ctx, tx, &analogdb.PostFilter{Author: &postAuthor}); err != nil {
			t.Fatal(err)
		} else if len(posts) != 1 || count != 1 {
			t.Fatal("must be one matching post")
		} else if got, want := posts[0].Title, postTitle; got != want {
			t.Fatalf("Post title does not match, got %v, want %v", got, want)
		}
	})

	t.Run("ByAuthorAddPrefix", func(t *testing.T) {
		ctx, tx := setupTx(t)
		noPrefixAuthor := postAuthor[2:]
		if posts, count, err := findPosts(ctx, tx, &analogdb.PostFilter{Author: &noPrefixAuthor}); err != nil {
			t.Fatal(err)
		} else if len(posts) != 1 || count != 1 {
			t.Fatal("must be one matching post")
		} else if got, want := posts[0].Title, postTitle; got != want {
			t.Fatalf("Post title does not match, got %v, want %v", got, want)
		}
	})

	t.Run("SearchTitleOne", func(t *testing.T) {
		ctx, tx := setupTx(t)
		keyword := "Melancholy"
		if posts, count, err := findPosts(ctx, tx, &analogdb.PostFilter{Title: &keyword}); err != nil {
			t.Fatal(err)
		} else if len(posts) != 1 || count != 1 {
			t.Fatal("must be one matching post")
		} else if got, want := posts[0].Title, postTitle; got != want {
			t.Fatalf("Post title does not match, got %v, want %v", got, want)
		}
	})

	t.Run("SearchTitleMultiple", func(t *testing.T) {
		ctx, tx := setupTx(t)
		keyword := "Portra"
		if posts, _, err := findPosts(ctx, tx, &analogdb.PostFilter{Title: &keyword}); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), totalPortra; got != want {
			t.Fatalf("number of matching titles not equal, got %v, want %v", got, want)
		}
	})
}

func TestLatestPost(t *testing.T) {
	t.Run("PostSequential", func(t *testing.T) {
		db := MustOpen(t)
		defer MustClose(t, db)
		ps := NewPostService(db)

		sort := time
		filter := &analogdb.PostFilter{Limit: &limit, Sort: &sort}

		posts, _, err := ps.FindPosts(context.Background(), filter)
		if err != nil {
			t.Fatal(err)
		}

		newest := posts[0].Time
		oldest := posts[limit-1].Time
		for _, p := range posts {
			if p.Time > newest {
				t.Fatalf("posts not sorted newest to oldest")
			}
		}

		filter.Keyset = &oldest
		posts, _, err = ps.FindPosts(context.Background(), filter)
		if err != nil {
			t.Fatal(err)
		}

		for _, p := range posts {
			if p.Time > oldest {
				t.Fatalf("posts not sorted newest to oldest with keyset")
			}
		}

	})
}

func TestTopPost(t *testing.T) {
	t.Run("PostTop", func(t *testing.T) {
		db := MustOpen(t)
		defer MustClose(t, db)
		ps := NewPostService(db)

		sort := score
		filter := &analogdb.PostFilter{Limit: &limit, Sort: &sort}

		posts, _, err := ps.FindPosts(context.Background(), filter)
		if err != nil {
			t.Fatal(err)
		}

		top := posts[0].Score
		bottom := posts[limit-1].Score
		for _, p := range posts {
			if p.Score > top {
				t.Fatalf("posts not sorted most to least votes")
			}
		}
		filter.Keyset = &bottom
		posts, _, err = ps.FindPosts(context.Background(), filter)
		if err != nil {
			t.Fatal(err)
		}

		for _, p := range posts {
			if p.Score > bottom {
				t.Fatalf("posts not sorted most to least votes with keyset")
			}
		}
	})
}

func TestRandomPost(t *testing.T) {
	t.Run("PostRandom", func(t *testing.T) {
		db := MustOpen(t)
		defer MustClose(t, db)
		ps := NewPostService(db)

		sort := random
		filter := &analogdb.PostFilter{Limit: &limit, Sort: &sort}

		if seed := filter.Seed; seed != nil {
			t.Fatal("unset seed must be nil")
		}

		posts, _, err := ps.FindPosts(context.Background(), filter)
		if err != nil {
			t.Fatal(err)
		}

		seen := make(map[int]bool)
		for _, p := range posts {
			seen[p.Id] = true
		}

		if seed := filter.Seed; seed == nil {
			t.Fatal("assigned seed must not be nil")
		}

		filter.Keyset = &posts[limit-1].Time

		posts, _, err = ps.FindPosts(context.Background(), filter)
		if err != nil {
			t.Fatal(err)
		}

		for _, p := range posts {
			if seen[p.Id] == true {
				t.Fatal("random posts must not repeat")
			}
		}

	})
}

func TestFindPost(t *testing.T) {
	t.Run("ErrNoPost", func(t *testing.T) {
		db := MustOpen(t)
		defer MustClose(t, db)
		ps := NewPostService(db)

		if _, err := ps.FindPostByID(context.Background(), 69); err == nil {
			t.Fatal("error should be returned when no matching post is found")
		}
	})

	t.Run("ByID", func(t *testing.T) {
		db := MustOpen(t)
		defer MustClose(t, db)
		ps := NewPostService(db)

		post, err := ps.FindPostByID(context.Background(), postID)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := post.Title, postTitle; got != want {
			t.Fatalf("Post title does not match, got %v, want %v", got, want)
		}
	})
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
