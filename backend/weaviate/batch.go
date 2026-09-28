package weaviate

import (
	"context"
	"slices"
	"strconv"

	"github.com/evanofslack/analogdb"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"github.com/weaviate/weaviate/entities/models"
	"golang.org/x/sync/errgroup"
)

const maxConcurrentDownloads = 10

var pictureNamespace = uuid.NewSHA1(uuid.NameSpaceURL, []byte("https://analogdb.com/picture"))

// pictureID is the deterministic weaviate object ID for a post
func pictureID(postID int) strfmt.UUID {
	return strfmt.UUID(uuid.NewSHA1(pictureNamespace, []byte(strconv.Itoa(postID))).String())
}

// BatchEncodePosts encodes posts in batches and returns the IDs that failed to encode
func (ss SimilarityService) BatchEncodePosts(ctx context.Context, ids []int, batchSize int) ([]int, error) {
	failedIDs := []int{}
	if len(ids) == 0 {
		return failedIDs, nil
	}

	batches := batchBy(ids, batchSize)
	for _, batch := range batches {
		filter := analogdb.PostFilter{IDs: &batch}
		posts, _, err := ss.postService.FindPosts(ctx, &filter)
		if err != nil {
			return failedIDs, err
		}
		failedIDs = append(failedIDs, missingIDs(batch, posts)...)

		pictureObjects, failedDownloads := ss.db.postsToPictureObjects(ctx, posts)
		failedIDs = append(failedIDs, failedDownloads...)
		if len(pictureObjects) == 0 {
			continue
		}

		failedUploads, err := ss.db.batchUploadObjects(ctx, pictureObjects)
		if err != nil {
			return failedIDs, err
		}
		for _, post := range posts {
			if slices.Contains(failedUploads, pictureID(post.Id)) {
				failedIDs = append(failedIDs, post.Id)
			}
		}
	}

	if len(failedIDs) != 0 {
		ss.db.logger.WarnContext(ctx, "Fail encode some posts", "failed_ids", failedIDs, "total", len(ids))
	}
	return failedIDs, nil
}

// batchUploadObjects upserts objects and returns the IDs of objects that failed
func (db *DB) batchUploadObjects(ctx context.Context, objects []*models.Object) ([]strfmt.UUID, error) {
	db.logger.DebugContext(ctx, "Start batch upload to vector db")

	batcher := db.db.Batch().ObjectsBatcher()
	for _, obj := range objects {
		batcher.WithObject(obj)
	}
	resp, err := batcher.Do(ctx)
	if err != nil {
		db.logger.ErrorContext(ctx, "Fail batch upload to vector db", "error", err)
		return nil, err
	}

	var failed []strfmt.UUID
	for _, r := range resp {
		if r.Result == nil || r.Result.Errors == nil {
			continue
		}
		db.logger.ErrorContext(ctx, "Fail upload object to vector db", "id", r.ID, "error", r.Result.Errors)
		failed = append(failed, r.ID)
	}
	return failed, nil
}

type encodeResult struct {
	post  *analogdb.Post
	image string
	err   error
}

// downloadAndEncodePosts downloads each post's image, writing results by index so posts and images stay paired
func downloadAndEncodePosts(ctx context.Context, posts []*analogdb.Post) []encodeResult {
	results := make([]encodeResult, len(posts))

	var g errgroup.Group
	g.SetLimit(maxConcurrentDownloads)
	for i, post := range posts {
		g.Go(func() error {
			image, err := downloadImage(ctx, post)
			results[i] = encodeResult{post: post, image: image, err: err}
			return nil
		})
	}
	_ = g.Wait()

	return results
}

func (db *DB) postsToPictureObjects(ctx context.Context, posts []*analogdb.Post) ([]*models.Object, []int) {
	results := downloadAndEncodePosts(ctx, posts)

	var pictureObjects []*models.Object
	var failedIDs []int

	for _, result := range results {
		post := result.post
		if result.err != nil {
			db.logger.ErrorContext(ctx, "Fail download and encode post image", "post_id", post.Id, "error", result.err)
			failedIDs = append(failedIDs, post.Id)
			continue
		}
		pictureObject := newPictureObject(result.image, post.Id, post.Grayscale, post.Nsfw, post.Sprocket)
		pictureObjects = append(pictureObjects, pictureObject)
	}

	return pictureObjects, failedIDs
}

func missingIDs(ids []int, posts []*analogdb.Post) []int {
	found := make(map[int]bool, len(posts))
	for _, post := range posts {
		found[post.Id] = true
	}
	var missing []int
	for _, id := range ids {
		if !found[id] {
			missing = append(missing, id)
		}
	}
	return missing
}

func newPictureObject(image string, postID int, grayscale bool, nsfw bool, sprocket bool) *models.Object {
	object := models.Object{
		Class: PictureClass,
		ID:    pictureID(postID),
		Properties: map[string]interface{}{
			"image":     image,
			"post_id":   postID,
			"grayscale": grayscale,
			"nsfw":      nsfw,
			"sprocket":  sprocket,
		},
	}
	return &object
}

func batchBy[T any](items []T, batchSize int) (batchs [][]T) {
	if batchSize <= 0 {
		batchSize = len(items)
	}
	for batchSize < len(items) {
		items, batchs = items[batchSize:], append(batchs, items[0:batchSize:batchSize])
	}
	return append(batchs, items)
}
