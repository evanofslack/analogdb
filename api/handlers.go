package main

import (
	"encoding/json"
	"go-reddit/models"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func getLatestPost(w http.ResponseWriter, r *http.Request) {

	numPosts := queryParamInt(r, "num")
	latest, err := models.LatestPost(numPosts)

	if err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(latest); err != nil {
		log.Fatal(err)
	}
}

func getTopPost(w http.ResponseWriter, r *http.Request) {

	numPosts := queryParamInt(r, "num")
	top, err := models.TopPost(numPosts)

	if err != nil {
		log.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(top); err != nil {
		log.Fatal(err)
	}
}

func getRandomPost(w http.ResponseWriter, r *http.Request) {

	numPosts := queryParamInt(r, "num")
	random, err := models.RandomPost(numPosts)
	if err != nil {
		log.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(random); err != nil {
		log.Fatal(err)
	}
}

func helloWorld(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World!"))
}

func queryParamInt(r *http.Request, name string) int {
	if num := chi.URLParam(r, name); num != "" {
		result, err := strconv.Atoi(num)
		if err != nil {
			return 1
		}
		return result
	} else {
		return 1
	}
}
