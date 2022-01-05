package middleware

import (
	"context"
	"log"
	"net/http"
	"strconv"
)

type CustomKey string

const PageIDKey CustomKey = "page_id"
const PageSizeKey CustomKey = "page_size"
const SeedKey CustomKey = "seed"
const NsfwKey CustomKey = "nsfw"
const GrayscaleKey CustomKey = "bw"

func Pagination(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		PageID := r.URL.Query().Get(string(PageIDKey))
		PageSize := r.URL.Query().Get(string(PageSizeKey))
		Seed := r.URL.Query().Get(string(SeedKey))
		Nsfw := r.URL.Query().Get(string(NsfwKey))
		Grayscale := r.URL.Query().Get(string(GrayscaleKey))

		intPageID := 0
		intPageSize := 10
		intSeed := 0
		boolNsfw := false
		boolGrayscale := true

		var err error
		if PageID != "" {
			intPageID, err = strconv.Atoi(PageID)
			if err != nil {
				log.Println(err)
				return
			}
		}
		if PageSize != "" {
			intPageSize, err = strconv.Atoi(PageSize)
			if err != nil {
				log.Println(err)
				return
			}
		}
		if Seed != "" {
			intSeed, err = strconv.Atoi(Seed)
			if err != nil {
				log.Println(err)
				return
			}
		}
		if Nsfw == "true" || Nsfw == "1" || Nsfw == "t" {
			boolNsfw = true
		}
		if Grayscale == "false" || Grayscale == "0" || Grayscale == "f" {
			boolGrayscale = false
		}
		ctx := context.WithValue(r.Context(), PageIDKey, intPageID)
		ctx = context.WithValue(ctx, PageSizeKey, intPageSize)
		ctx = context.WithValue(ctx, SeedKey, intSeed)
		ctx = context.WithValue(ctx, NsfwKey, boolNsfw)
		ctx = context.WithValue(ctx, GrayscaleKey, boolGrayscale)
		next.ServeHTTP(w, r.WithContext(ctx))

	})
}
