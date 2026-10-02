package weaviate

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/weaviate/weaviate/entities/models"
	"go.opentelemetry.io/otel/codes"
)

func (ss SimilarityService) EncodePost(ctx context.Context, id int) error {
	ss.db.logger.DebugContext(ctx, "Start encode post", "post_id", id)

	ctx, span := ss.db.tracer.Tracer.Start(ctx, "vector:encode_post")
	defer span.End()

	post, err := ss.postService.FindPostByID(ctx, id)
	if err != nil {
		err = fmt.Errorf("failed to find post by ID: %w", err)
		return err
	}
	obj, err := ss.db.postToPictureObject(ctx, post)
	if err != nil {
		err = fmt.Errorf("failed to convert post to picture object: %w", err)
		return err
	}
	err = ss.db.uploadObject(ctx, obj)
	if err != nil {
		err = fmt.Errorf("failed to upload picture object: %w", err)
		return err
	}
	return nil
}

const (
	imageDownloadTimeout = 30 * time.Second
	maxImageBytes        = 20 << 20
)

var imageClient = &http.Client{Timeout: imageDownloadTimeout}

func (db *DB) downloadPostImage(ctx context.Context, post *analogdb.Post) (string, error) {
	db.logger.DebugContext(ctx, "Start download post", "post_id", post.Id)

	ctx, span := db.tracer.Tracer.Start(ctx, "vector:download_post_image")
	defer span.End()

	encode, err := downloadImage(ctx, post)
	if err != nil {
		span.SetStatus(codes.Error, "Download post image failed")
		span.RecordError(err)
		return "", err
	}
	span.AddEvent("Downloaded and encoded post image")
	return encode, nil
}

// downloadImage fetches the post's medium image and returns it base64 encoded
func downloadImage(ctx context.Context, post *analogdb.Post) (string, error) {
	if len(post.Images) < 2 {
		return "", fmt.Errorf("post %d has no medium image", post.Id)
	}
	url := post.Images[1].Url

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := imageClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request post image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("failed to request post image, status=%d", resp.StatusCode)
	}
	if contentType := resp.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "image/") {
		return "", fmt.Errorf("post image has unexpected content type %q", contentType)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes+1))
	if err != nil {
		return "", fmt.Errorf("failed to read post image: %w", err)
	}
	if len(data) > maxImageBytes {
		return "", fmt.Errorf("post image larger than %d bytes", maxImageBytes)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func (db *DB) postToPictureObject(ctx context.Context, post *analogdb.Post) (*models.Object, error) {
	db.logger.DebugContext(ctx, "Start convert post to picture object", "post_id", post.Id)

	image, err := db.downloadPostImage(ctx, post)
	if err != nil {
		err = fmt.Errorf("failed to download post image: %w", err)
		return nil, err
	}
	pictureObject := newPictureObject(post, image)
	return pictureObject, nil
}

func (db *DB) uploadObject(ctx context.Context, obj *models.Object) error {
	db.logger.DebugContext(ctx, "Start upload object")

	ctx, span := db.startTrace(ctx, "vector:upload_object")
	defer span.End()

	failed, err := db.batchUploadObjects(ctx, []*models.Object{obj})
	if err == nil && len(failed) != 0 {
		err = fmt.Errorf("vector DB rejected object %s", obj.ID)
	}
	if err != nil {
		err = fmt.Errorf("failed to upload to vector DB: %w", err)
		db.logger.ErrorContext(ctx, "Fail upload object", "error", err)
		span.SetStatus(codes.Error, "Upload object failed")
		span.RecordError(err)
		return err
	}

	return nil
}
