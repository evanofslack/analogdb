package main

import (
	"encoding/json"
	"fmt"
	mw "go-reddit/middleware"
	"go-reddit/models"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func listLatest(w http.ResponseWriter, r *http.Request) {

	pageSize := r.Context().Value(mw.PageSizeKey)
	pageID := r.Context().Value(mw.PageIDKey)
	nsfw := r.Context().Value(mw.NsfwKey)
	grayscale := r.Context().Value(mw.GrayscaleKey)

	latest, err := models.LatestPost(pageSize.(int), pageID.(int), nsfw.(bool), grayscale.(bool))

	if err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(latest); err != nil {
		log.Fatal(err)
	}
}

func listTop(w http.ResponseWriter, r *http.Request) {

	pageSize := r.Context().Value(mw.PageSizeKey)
	pageID := r.Context().Value(mw.PageIDKey)
	nsfw := r.Context().Value(mw.NsfwKey)
	grayscale := r.Context().Value(mw.GrayscaleKey)

	top, err := models.TopPost(pageSize.(int), pageID.(int), nsfw.(bool), grayscale.(bool))

	if err != nil {
		log.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(top); err != nil {
		log.Fatal(err)
	}
}

func listRandom(w http.ResponseWriter, r *http.Request) {

	pageSize := r.Context().Value(mw.PageSizeKey)
	pageID := r.Context().Value(mw.PageIDKey)
	nsfw := r.Context().Value(mw.NsfwKey)
	grayscale := r.Context().Value(mw.GrayscaleKey)
	seed := r.Context().Value(mw.SeedKey)
	random, err := models.RandomPost(pageSize.(int), pageID.(int), nsfw.(bool), grayscale.(bool), seed.(int))
	if err != nil {
		log.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(random); err != nil {
		log.Fatal(err)
	}
}

func listNsfw(w http.ResponseWriter, r *http.Request) {

	pageSize := r.Context().Value(mw.PageSizeKey)
	pageID := r.Context().Value(mw.PageIDKey)

	nsfw, err := models.NsfwPost(pageSize.(int), pageID.(int))

	if err != nil {
		log.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(nsfw); err != nil {
		log.Fatal(err)
	}
}

func listBw(w http.ResponseWriter, r *http.Request) {

	pageSize := r.Context().Value(mw.PageSizeKey)
	pageID := r.Context().Value(mw.PageIDKey)

	bw, err := models.BwPost(pageSize.(int), pageID.(int))

	if err != nil {
		log.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(bw); err != nil {
		log.Fatal(err)
	}
}

func listSprocket(w http.ResponseWriter, r *http.Request) {

	pageSize := r.Context().Value(mw.PageSizeKey)
	pageID := r.Context().Value(mw.PageIDKey)

	sprocket, err := models.SprocketPost(pageSize.(int), pageID.(int))

	if err != nil {
		log.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(sprocket); err != nil {
		log.Fatal(err)
	}
}

func findPost(w http.ResponseWriter, r *http.Request) {
	var post models.Post
	var err error

	if id := chi.URLParam(r, "id"); id != "" {
		fmt.Println(id)
		intId, _ := strconv.Atoi(id)
		post, err = models.FindPost(intId)
	}

	if err != nil {
		// TODO Check that item exists in DB, return http.error message if it does not exist
		fmt.Println(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	if err := enc.Encode(post); err != nil {
		log.Fatal(err)
	}
}
