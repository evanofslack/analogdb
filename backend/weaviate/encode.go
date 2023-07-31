package weaviate

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/evanofslack/analogdb"
	"github.com/weaviate/weaviate/entities/models"
	"go.opentelemetry.io/otel/codes"
)

func (ss SimilarityService) EncodePost(ctx context.Context, id int) error {

	ss.db.logger.Debug().Ctx(ctx).Int("postID", id).Msg("Starting encode post")

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

func (db *DB) downloadPostImage(ctx context.Context, post *analogdb.Post) (string, error) {

	db.logger.Debug().Ctx(ctx).Msg("Starting download post")

	ctx, span := db.tracer.Tracer.Start(ctx, "vector:download_post_image")
	defer span.End()

	var encode string
	url := post.Images[1].Url

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		err = fmt.Errorf("failed to create request: %w", err)
		span.SetStatus(codes.Error, "Create request failed")
		span.RecordError(err)
		return encode, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		resp.Body.Close()
		err = fmt.Errorf("failed to request post image: %w", err)
		span.SetStatus(codes.Error, "Request for post image failed")
		span.RecordError(err)
		return encode, err
	}
	defer resp.Body.Close()
	span.AddEvent("Downloaded post image")

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("failed to read post image: %w", err)
		span.SetStatus(codes.Error, "Reading of post image bytes failed")
		span.RecordError(err)
		return encode, err
	}
	encode = base64.StdEncoding.EncodeToString(data)
	span.AddEvent("Encoded to base64")
	return encode, nil
}

func (db *DB) postToPictureObject(ctx context.Context, post *analogdb.Post) (*models.Object, error) {

	db.logger.Debug().Ctx(ctx).Msg("Starting convert post to picture object")

	image, err := db.downloadPostImage(ctx, post)
	if err != nil {
		err = fmt.Errorf("failed to download post image: %w", err)
		return nil, err
	}
	pictureObject := newPictureObject(image, post.Id, post.Grayscale, post.Nsfw, post.Sprocket)
	return pictureObject, nil
}

func (db *DB) uploadObject(ctx context.Context, obj *models.Object) error {

	db.logger.Debug().Ctx(ctx).Msg("Starting upload object")

	ctx, span := db.startTrace(ctx, "vector:upload_object")
	defer span.End()

	batcher := db.db.Batch().ObjectsBatcher()
	_, err := batcher.WithObject(obj).Do(ctx)
	if err != nil {
		err = fmt.Errorf("failed to upload to vector DB: %w", err)
		db.logger.Error().Err(err).Ctx(ctx).Msg("Failed upload to vector DB")
		span.SetStatus(codes.Error, "Upload object failed")
		span.RecordError(err)
		return err
	}

	return nil
}
