package weaviate

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/weaviate/weaviate/entities/models"
)

func (ss SimilarityService) BatchEncodePosts(ctx context.Context, ids []int, batchSize int) error {
	fmt.Println("entering BatchEncodePosts")

	batches := batchBy(ids, batchSize)
	for _, batch := range batches {
		fmt.Println("looping batches")
		filter := analogdb.PostFilter{IDs: &batch}
		posts, _, err := ss.postService.FindPosts(ctx, &filter)
		fmt.Println("found post by id")
		if err != nil {
			return err
		}
		fmt.Println("convert post to pic object")
		pictureObjects := postsToPictureObjects(posts)
		fmt.Println("start put to img2vec")
		err = ss.db.batchUploadObjects(ctx, pictureObjects)
		if err != nil {
			return err
		}
		fmt.Println("img2vec success")
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

func maxThreadsDownload(maxGoroutines int, posts []*analogdb.Post, wg *sync.WaitGroup, encodesChan chan string, postsChan chan *analogdb.Post, failedChan chan int) {
	// limit max concurrent goroutines
	guard := make(chan int, maxGoroutines)

	for _, post := range posts {
		wg.Add(1)
		guard <- 1
		go func(post *analogdb.Post, wg *sync.WaitGroup, encodesChan chan string, postsChan chan *analogdb.Post, failedChan chan int) {
			downloadAndEncodePost(post, wg, encodesChan, postsChan, failedChan)
			<-guard
		}(post, wg, encodesChan, postsChan, failedChan)
	}
}

func downloadAndEncodePosts(posts []*analogdb.Post) ([]string, []*analogdb.Post, []int) {
	var wg sync.WaitGroup

	encodesChan := make(chan string)
	postsChan := make(chan *analogdb.Post)
	failedChan := make(chan int)

	maxGoroutines := 10
	go maxThreadsDownload(maxGoroutines, posts, &wg, encodesChan, postsChan, failedChan)

	go func() {
		time.Sleep(time.Second * 2)
		wg.Wait()
		fmt.Println("waitgroup finished")
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
				fmt.Println("encodedChan closed")
				return encodedImages, successPosts, failedIDs
			}
		case post, ok := <-postsChan:
			if ok {
				successPosts = append(successPosts, post)
			} else {
				fmt.Println("successPosts closed")
				return encodedImages, successPosts, failedIDs
			}
		case id, ok := <-failedChan:
			if ok {
				failedIDs = append(failedIDs, id)
			} else {
				fmt.Println("failedChan closed")
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
	return
}

func postsToPictureObjects(posts []*analogdb.Post) []*models.Object {
	fmt.Println("starting download pictures")
	encodedImages, successPosts, failedIDs := downloadAndEncodePosts(posts)
	fmt.Println("converting encoded images to weaviate objects")
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

func batchBy[T any](items []T, batchSize int) (batchs [][]T) {
	for batchSize < len(items) {
		items, batchs = items[batchSize:], append(batchs, items[0:batchSize:batchSize])
	}
	return append(batchs, items)
}
