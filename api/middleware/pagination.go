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

func Pagination(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		PageID := r.URL.Query().Get(string(PageIDKey))
		PageSize := r.URL.Query().Get(string(PageSizeKey))
		Seed := r.URL.Query().Get(string(SeedKey))
		intPageID := 0
		intPageSize := 10
		intSeed := 0
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
		ctx := context.WithValue(r.Context(), PageIDKey, intPageID)
		ctx = context.WithValue(ctx, PageSizeKey, intPageSize)
		ctx = context.WithValue(ctx, SeedKey, intSeed)
		next.ServeHTTP(w, r.WithContext(ctx))

	})
}
