package postgres

import (
	"context"
	"database/sql"
	"testing"

	"github.com/evanofslack/analogdb"
)

const (
	// number of posts matching each query from test DB
	totalPosts     = 4864
	totalNsfw      = 276
	totalGrayscale = 932
	totalSprocket  = 204
	totalPortra    = 1463
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

		db := mustOpen(t)
		defer mustClose(t, db)
		ctx, tx := setupTx(t, db)

		if posts, count, err := db.findPosts(ctx, tx, nilFilter); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), totalPosts; got != want {
			t.Fatalf("length of posts %v, want %v", got, want)
		} else if got, want := count, totalPosts; got != want {
			t.Fatalf("total count %v, want %v", got, want)
		}
	})

	t.Run("Limit", func(t *testing.T) {

		db := mustOpen(t)
		defer mustClose(t, db)
		ctx, tx := setupTx(t, db)

		if posts, count, err := db.findPosts(ctx, tx, limitFilter); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), limit; got != want {
			t.Fatalf("length of posts %v, want %v", got, want)
		} else if got, want := count, totalPosts; got != want {
			t.Fatalf("total count %v, want %v", got, want)
		}
	})

	t.Run("NoNSFW", func(t *testing.T) {

		db := mustOpen(t)
		defer mustClose(t, db)
		ctx, tx := setupTx(t, db)

		nsfw := false
		if posts, count, err := db.findPosts(ctx, tx, &analogdb.PostFilter{Nsfw: &nsfw}); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), totalPosts-totalNsfw; got != want {
			t.Fatalf("length of posts %v, want %v", got, want)
		} else if got, want := count, totalPosts-totalNsfw; got != want {
			t.Fatalf("total count %v, want %v", got, want)
		}
	})

	t.Run("OnlyNSFW", func(t *testing.T) {

		db := mustOpen(t)
		defer mustClose(t, db)
		ctx, tx := setupTx(t, db)

		nsfw := true
		if posts, count, err := db.findPosts(ctx, tx, &analogdb.PostFilter{Nsfw: &nsfw}); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), totalNsfw; got != want {
			t.Fatalf("length of posts %v, want %v", got, want)
		} else if got, want := count, totalNsfw; got != want {
			t.Fatalf("total count %v, want %v", got, want)
		}
	})

	t.Run("NoBW", func(t *testing.T) {

		db := mustOpen(t)
		defer mustClose(t, db)
		ctx, tx := setupTx(t, db)

		grayscale := false
		if posts, count, err := db.findPosts(ctx, tx, &analogdb.PostFilter{Grayscale: &grayscale}); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), totalPosts-totalGrayscale; got != want {
			t.Fatalf("length of posts %v, want %v", got, want)
		} else if got, want := count, totalPosts-totalGrayscale; got != want {
			t.Fatalf("total count %v, want %v", got, want)
		}
	})

	t.Run("OnlyBW", func(t *testing.T) {

		db := mustOpen(t)
		defer mustClose(t, db)
		ctx, tx := setupTx(t, db)

		grayscale := true
		if posts, count, err := db.findPosts(ctx, tx, &analogdb.PostFilter{Grayscale: &grayscale}); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), totalGrayscale; got != want {
			t.Fatalf("length of posts %v, want %v", got, want)
		} else if got, want := count, totalGrayscale; got != want {
			t.Fatalf("total count %v, want %v", got, want)
		}
	})

	t.Run("NoSprocket", func(t *testing.T) {

		db := mustOpen(t)
		defer mustClose(t, db)
		ctx, tx := setupTx(t, db)

		sprocket := false
		if posts, count, err := db.findPosts(ctx, tx, &analogdb.PostFilter{Sprocket: &sprocket}); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), totalPosts-totalSprocket; got != want {
			t.Fatalf("length of posts %v, want %v", got, want)
		} else if got, want := count, totalPosts-totalSprocket; got != want {
			t.Fatalf("total count %v, want %v", got, want)
		}
	})

	t.Run("OnlySprocket", func(t *testing.T) {

		db := mustOpen(t)
		defer mustClose(t, db)
		ctx, tx := setupTx(t, db)

		sprocket := true
		if posts, count, err := db.findPosts(ctx, tx, &analogdb.PostFilter{Sprocket: &sprocket}); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), totalSprocket; got != want {
			t.Fatalf("length of posts %v, want %v", got, want)
		} else if got, want := count, totalSprocket; got != want {
			t.Fatalf("total count %v, want %v", got, want)
		}
	})

	t.Run("ByAuthor", func(t *testing.T) {

		db := mustOpen(t)
		defer mustClose(t, db)
		ctx, tx := setupTx(t, db)

		if posts, count, err := db.findPosts(ctx, tx, &analogdb.PostFilter{Author: &postAuthor}); err != nil {
			t.Fatal(err)
		} else if len(posts) != 1 || count != 1 {
			t.Fatal("must be one matching post")
		} else if got, want := posts[0].Title, postTitle; got != want {
			t.Fatalf("Post title does not match, got %v, want %v", got, want)
		}
	})

	t.Run("ByAuthorAddPrefix", func(t *testing.T) {

		db := mustOpen(t)
		defer mustClose(t, db)
		ctx, tx := setupTx(t, db)

		noPrefixAuthor := postAuthor[2:]
		if posts, count, err := db.findPosts(ctx, tx, &analogdb.PostFilter{Author: &noPrefixAuthor}); err != nil {
			t.Fatal(err)
		} else if len(posts) != 1 || count != 1 {
			t.Fatal("must be one matching post")
		} else if got, want := posts[0].Title, postTitle; got != want {
			t.Fatalf("Post title does not match, got %v, want %v", got, want)
		}
	})

	t.Run("SearchTitleOne", func(t *testing.T) {

		db := mustOpen(t)
		defer mustClose(t, db)
		ctx, tx := setupTx(t, db)

		keyword := postTitle
		if posts, count, err := db.findPosts(ctx, tx, &analogdb.PostFilter{Title: &keyword}); err != nil {
			t.Fatal(err)
		} else if len(posts) != 1 || count != 1 {
			t.Fatal("must be one matching post")
		} else if got, want := posts[0].Title, postTitle; got != want {
			t.Fatalf("Post title does not match, got %v, want %v", got, want)
		}
	})

	t.Run("SearchTitleMultiple", func(t *testing.T) {

		db := mustOpen(t)
		defer mustClose(t, db)
		ctx, tx := setupTx(t, db)

		keyword := "Portra"
		if posts, _, err := db.findPosts(ctx, tx, &analogdb.PostFilter{Title: &keyword}); err != nil {
			t.Fatal(err)
		} else if got, want := len(posts), totalPortra; got != want {
			t.Fatalf("number of matching titles not equal, got %v, want %v", got, want)
		}
	})
}

func TestLatestPost(t *testing.T) {
	t.Run("PostSequential", func(t *testing.T) {
		db := mustOpen(t)
		defer mustClose(t, db)
		ps := NewPostService(db)

		sort := analogdb.SortTime
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
		db := mustOpen(t)
		defer mustClose(t, db)
		ps := NewPostService(db)

		sort := analogdb.SortScore
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
		db := mustOpen(t)
		defer mustClose(t, db)
		ps := NewPostService(db)

		sort := analogdb.SortRandom
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
		db := mustOpen(t)
		defer mustClose(t, db)
		ps := NewPostService(db)

		if _, err := ps.FindPostByID(context.Background(), 69); err == nil {
			t.Fatal("error should be returned when no matching post is found")
		}
	})

	t.Run("ByID", func(t *testing.T) {
		db := mustOpen(t)
		defer mustClose(t, db)
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

func TestCreateAndDeletePost(t *testing.T) {
	t.Run("valid post", func(t *testing.T) {
		db := mustOpen(t)
		defer mustClose(t, db)
		ps := NewPostService(db)

		testImage := analogdb.Image{
			Label:  "test",
			Url:    "test.com",
			Width:  0,
			Height: 0,
		}
		fourImages := []analogdb.Image{testImage, testImage, testImage, testImage}

		testColor := analogdb.Color{
			Hex:     "#000000",
			Css:     "Black",
			Percent: 0.2500000,
		}
		fiveColors := []analogdb.Color{testColor, testColor, testColor, testColor, testColor}

		keyword := analogdb.Keyword{Word: "keyword", Weight: 0.1}
		keywords := []analogdb.Keyword{keyword, keyword, keyword}

		testTitle := "test title"

		createPost := analogdb.CreatePost{
			Title:     testTitle,
			Author:    "test author",
			Permalink: "test.permalink.com",
			Score:     0,
			Nsfw:      false,
			Grayscale: false,
			Time:      0,
			Sprocket:  false,
			Images:    fourImages,
			Colors:    fiveColors,
			Keywords:  keywords,
		}

		ctx := context.Background()

		created, err := ps.CreatePost(ctx, &createPost)
		if err != nil {
			t.Fatalf("valid post should be created, error: %s", err)
		}

		if created.Title != testTitle {
			t.Fatalf("created post has invalid title, got %v, want %v", created.Title, testTitle)
		}

		if err := ps.DeletePost(ctx, created.Id); err != nil {
			t.Fatalf("unable to delete post created to test create post, error: %s", err)
		}
	})

	t.Run("3 images is an invalid post", func(t *testing.T) {
		db := mustOpen(t)
		defer mustClose(t, db)
		ps := NewPostService(db)

		testImage := analogdb.Image{
			Label:  "test",
			Url:    "test.com",
			Width:  0,
			Height: 0,
		}
		var threeImages []analogdb.Image

		threeImages = append(threeImages, testImage, testImage, testImage)

		testColor := analogdb.Color{
			Hex:     "#000000",
			Css:     "Black",
			Percent: 0.2500000,
		}
		fiveColors := []analogdb.Color{testColor, testColor, testColor, testColor, testColor}

		testTitle := "test title"

		createPost := analogdb.CreatePost{
			Title:     testTitle,
			Author:    "test author",
			Permalink: "test.permalink.com",
			Score:     0,
			Nsfw:      false,
			Grayscale: false,
			Time:      0,
			Sprocket:  false,
			Images:    threeImages,
			Colors:    fiveColors,
		}

		ctx := context.Background()

		_, err := ps.CreatePost(ctx, &createPost)
		if err == nil {
			t.Fatal("invalid post should not be created")
		}
		if err != nil {
			if err.Error() != "analogdb error: code: unprocessable message: Unable to create post, expected 4 images (low, medium, high, raw)" {
				t.Fatal("expected analogdb error message")
			}
		}

	})
}

func TestAllPostIDs(t *testing.T) {
	t.Run("Number of IDs", func(t *testing.T) {
		db := mustOpen(t)
		defer mustClose(t, db)
		ps := NewPostService(db)

		ids, err := ps.AllPostIDs(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		numIDs := len(ids)
		if numIDs != totalPosts {
			t.Fatalf("wrong number of total post IDs, wanted %d, got %d", totalPosts, numIDs)
		}
	})
	t.Run("IDs are correct", func(t *testing.T) {
		db := mustOpen(t)
		defer mustClose(t, db)
		ps := NewPostService(db)

		ids, err := ps.AllPostIDs(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if ids[0] != 1 || ids[1] != 2 || ids[2] != 3 {
			t.Fatalf("wrong values of post IDs, wanted %d, %d, %d, got %d, %d, %d", 1, 2, 3, ids[0], ids[1], ids[2])
		}
	})
}

func TestPatchPost(t *testing.T) {
	t.Run("ErrNoFields", func(t *testing.T) {
		db := mustOpen(t)
		defer mustClose(t, db)
		ps := NewPostService(db)

		patch := analogdb.PatchPost{}

		if err := ps.PatchPost(context.Background(), &patch, postID); err == nil {
			t.Fatal("error should be returned when no patch fields are provided")
		}
	})

	t.Run("UpdateKeywords", func(t *testing.T) {
		db := mustOpen(t)
		defer mustClose(t, db)
		ps := NewPostService(db)

		og, err := ps.FindPostByID(context.Background(), postID)
		if err != nil {
			t.Fatal(err)
		}

		keyword := analogdb.Keyword{Word: "keyword", Weight: 0.1}
		keywords := []analogdb.Keyword{keyword, keyword, keyword}
		patch := analogdb.PatchPost{
			Keywords: &keywords,
		}
		err = ps.PatchPost(context.Background(), &patch, postID)
		if err != nil {
			t.Fatal(err)
		}

		updated, err := ps.FindPostByID(context.Background(), postID)
		if err != nil {
			t.Fatal(err)
		}

		if len(og.Keywords) == len(updated.Keywords) {
			t.Fatalf("updated keywords should have different length than original, original: %d, updated: %d", len(og.Keywords), len(updated.Keywords))
		}

		// remove added keywords for idempotency
		keywords = []analogdb.Keyword{}
		patch = analogdb.PatchPost{
			Keywords: &keywords,
		}
		err = ps.PatchPost(context.Background(), &patch, postID)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("UpdateScore", func(t *testing.T) {
		db := mustOpen(t)
		defer mustClose(t, db)
		ps := NewPostService(db)

		og, err := ps.FindPostByID(context.Background(), postID)
		if err != nil {
			t.Fatal(err)
		}

		newScore := og.Score + 1
		patch := analogdb.PatchPost{
			Score: &newScore,
		}
		err = ps.PatchPost(context.Background(), &patch, postID)
		if err != nil {
			t.Fatal(err)
		}

		updated, err := ps.FindPostByID(context.Background(), postID)
		if err != nil {
			t.Fatal(err)
		}

		if og.Score == updated.Score {
			t.Fatalf("updated post should have different score than original post, original: %d, updated: %d", og.Score, updated.Score)
		}
	})
}

func setupTx(t *testing.T, db *DB) (context.Context, *sql.Tx) {
	t.Helper()
	ctx := context.Background()
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	return ctx, tx
}
