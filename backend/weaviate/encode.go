package weaviate

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/evanofslack/analogdb"
	"github.com/weaviate/weaviate/entities/models"
)

func (ss SimilarityService) EncodePost(ctx context.Context, id int) error {
	post, err := ss.postService.FindPostByID(ctx, id)
	if err != nil {
		err = fmt.Errorf("failed to find post by ID: %w", err)
		return err
	}
	obj, err := postToPictureObject(post)
	if err != nil {
		err = fmt.Errorf("failed to convert post to picture object: %w", err)
		return err
	}
	err = ss.db.uploadObject(ctx,  obj)
	if err != nil {
		err = fmt.Errorf("failed to upload picture object: %w", err)
		return err
	}
	return nil
}


func downloadPostImage(post *analogdb.Post) (string, error) {

	var encode string

	url := post.Images[1].Url
	resp, err := http.Get(url)
	if err != nil {
		resp.Body.Close()
		err = fmt.Errorf("failed to request post image: %w", err)
		return encode, err
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("failed to read post image: %w", err)
		return encode, err
	}
	encode = base64.StdEncoding.EncodeToString(data)
	return encode, nil
}

func postToPictureObject(post *analogdb.Post) (*models.Object, error) {
	image, err := downloadPostImage(post)
	if err != nil {
		err = fmt.Errorf("failed to download post image: %w", err)
		return nil, err
	}
	pictureObject := newPictureObject(image, post.Id, post.Grayscale, post.Nsfw, post.Sprocket)
	return pictureObject, nil
}

func (db *DB) uploadObject(ctx context.Context, obj *models.Object) error {

	db.logger.Debug().Msg("Starting upload object to vector DB")

	batcher := db.db.Batch().ObjectsBatcher()
	_, err := batcher.WithObject(obj).Do(ctx)
	if err != nil {
		err = fmt.Errorf("failed to upload to vector DB: %w", err)
		db.logger.Error().Err(err).Msg("Failed upload to vector DB")
		return err
	}
	return nil
}
