package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

const totalAuthors = 4864

func TestFindAuthors(t *testing.T) {
	t.Run("Correct author length", func(t *testing.T) {
		s, db := mustOpen(t)
		defer mustClose(t, s, db)

		r := httptest.NewRequest(http.MethodGet, "/authors", nil)
		w := httptest.NewRecorder()

		s.router.ServeHTTP(w, r)

		if want, got := http.StatusOK, w.Code; got != want {
			t.Errorf("want status %d, got %d", want, got)
		}

		res := w.Result()
		defer res.Body.Close()

		data, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}

		var authors AuthorsResponse

		if err := json.Unmarshal(data, &authors); err != nil {
			t.Fatal(err)
		}

		if got, want := len(authors.Authors), totalAuthors; got != want {
			t.Errorf("invalid number of post IDs, want %d, got %d", want, got)
		}
	})
}
