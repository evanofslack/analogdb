package weaviate

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/evanofslack/analogdb"
	"github.com/weaviate/weaviate/entities/models"
)

func (ss SimilarityService) BatchEncodePosts(ctx context.Context, ids []int) error {
	fmt.Println("start batch encode")
	filter := analogdb.PostFilter{IDs: &ids}
	posts, _, err := ss.postService.FindPosts(ctx, &filter)
	if err != nil {
		return err
	}
	fmt.Println("got batch encode posts")
	fmt.Println(len(posts))
	pictureObjects := postsToPictureObjects(posts)
	fmt.Println("got batch encode objects")
	err = ss.db.batchUploadObjects(ctx, pictureObjects)
	if err != nil {
		return err
	}
	fmt.Println("batch encode success")

	return nil
}

func (db *DB) batchUploadObjects(ctx context.Context, objects []*models.Object) error {
	batcher := db.db.Batch().ObjectsBatcher()
	for _, obj := range objects {
		batcher.WithObject(obj)
	}
	_, err := batcher.Do(ctx)
	if err != nil {
		return err
	}
	return nil
}

func downloadAndEncodePosts(posts []*analogdb.Post) ([]string, []*analogdb.Post, []int) {
	var wg sync.WaitGroup

	encodesChan := make(chan string)
	postsChan := make(chan *analogdb.Post)
	failedChan := make(chan int)

	for _, post := range posts {
		wg.Add(1)
		fmt.Println("add WG")
		go downloadAndEncodePost(post, &wg, encodesChan, postsChan, failedChan)
	}

	go func() {
		wg.Wait()
		fmt.Println("done waiting WG")
		close(encodesChan)
		close(postsChan)
		close(failedChan)
	}()

	var encodedImages []string
	var successPosts []*analogdb.Post
	var failedIDs []int

	for {
		select {
		case encoded, ok := <-encodesChan:
			if ok {
				encodedImages = append(encodedImages, encoded)
			} else {
				return encodedImages, successPosts, failedIDs
			}
		case post, ok := <-postsChan:
			if ok {
				successPosts = append(successPosts, post)
			} else {
				return encodedImages, successPosts, failedIDs
			}
		case id, ok := <-failedChan:
			if ok {
				failedIDs = append(failedIDs, id)
			} else {
				return encodedImages, successPosts, failedIDs
			}
		}
	}
}

func downloadAndEncodePost(post *analogdb.Post, wg *sync.WaitGroup, encodes chan string, posts chan *analogdb.Post, failed chan int) {
	defer wg.Done()

	url := post.Images[1].Url
	id := post.Id
	resp, err := http.Get(url)
	fmt.Println(resp)
	if err != nil {
		fmt.Println("request errored")
		failed <- id
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		failed <- id
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	encodes <- encoded
	posts <- post
	fmt.Println("close WG")
	return
}

func postsToPictureObjects(posts []*analogdb.Post) []*models.Object {
	encodedImages, successPosts, failedIDs := downloadAndEncodePosts(posts)
	var pictureObjects []*models.Object

	for i := range encodedImages {
		image := encodedImages[i]
		post := successPosts[i]
		pictureObject := newPictureObject(image, post.Id, post.Grayscale, post.Nsfw, post.Sprocket)
		pictureObjects = append(pictureObjects, pictureObject)

	}

	if len(failedIDs) != 0 {
		fmt.Println(fmt.Sprintf("failed to download/encode post ids: %v", failedIDs))
	}

	return pictureObjects
}

func newPictureObject(image string, postID int, grayscale bool, nsfw bool, sprocket bool) *models.Object {
	object := models.Object{
		Class: "Picture",
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
