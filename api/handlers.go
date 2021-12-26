package main

import (
	"encoding/json"
	"go-reddit/models"
	"log"
	"net/http"
)

func getLatest(w http.ResponseWriter, r *http.Request) {
	latest, err := models.LatestPost()
	if err != nil {
		log.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(latest); err != nil {
		log.Fatal(err)
	}
}

func getRandom(w http.ResponseWriter, r *http.Request) {
	random, err := models.RandomPost()
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
