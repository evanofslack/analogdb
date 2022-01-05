package main

import (
	"encoding/json"
	mw "go-reddit/middleware"
	"go-reddit/models"
	"log"
	"net/http"
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
