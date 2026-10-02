package weaviate

import (
	"context"
	"slices"
	"strconv"
	"strings"

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
		batcher.WithObjects(obj)
	}
	resp, err := batcher.Do(ctx)
	if err != nil {
		db.logger.ErrorContext(ctx, "Fail batch upload to vector db", "error", err)
		return nil, err
	}

	var failed []strfmt.UUID
	for _, objErr := range batchObjectErrors(resp) {
		db.logger.ErrorContext(ctx, "Fail upload object to vector db", "id", objErr.id, "error", objErr.message)
		failed = append(failed, objErr.id)
	}
	return failed, nil
}

type objectError struct {
	id      strfmt.UUID
	message string
}

// batchObjectErrors returns the objects of a batch response that failed
func batchObjectErrors(resp []models.ObjectsGetResponse) []objectError {
	var errs []objectError
	for _, r := range resp {
		if r.Result == nil {
			continue
		}
		var messages []string
		if r.Result.Errors != nil {
			for _, item := range r.Result.Errors.Error {
				if item != nil {
					messages = append(messages, item.Message)
				}
			}
		}
		failed := r.Result.Status != nil && *r.Result.Status == models.ObjectsGetResponseAO2ResultStatusFAILED
		if len(messages) == 0 && !failed {
			continue
		}
		if len(messages) == 0 {
			messages = append(messages, "status failed")
		}
		errs = append(errs, objectError{id: r.ID, message: strings.Join(messages, ", ")})
	}
	return errs
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
		pictureObject := newPictureObject(post, result.image)
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

func newPictureObject(post *analogdb.Post, image string) *models.Object {
	caption := ""
	if post.Caption != nil {
		caption = *post.Caption
	}
	tags := make([]string, 0, len(post.Keywords))
	for _, keyword := range post.Keywords {
		tags = append(tags, keyword.Word)
	}
	object := models.Object{
		Class: PostImageClass,
		ID:    pictureID(post.Id),
		Properties: map[string]interface{}{
			"image":     image,
			"post_id":   post.Id,
			"title":     post.Title,
			"caption":   caption,
			"tags":      tags,
			"grayscale": post.Grayscale,
			"nsfw":      post.Nsfw,
			"sprocket":  post.Sprocket,
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
