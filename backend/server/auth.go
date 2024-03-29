package server

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
)

type contextKey string

const authKey contextKey = "authorized"

func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		username := s.config.Auth.Username
		password := s.config.Auth.Password
		authenticated := s.passBasicAuth(username, password, r)

		if authenticated {
			ctx := context.WithValue(r.Context(), authKey, true)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

func (s *Server) passBasicAuth(username, password string, r *http.Request) bool {
	username, password, ok := r.BasicAuth()
	if !ok {
		return false
	}

	usernameHash := sha256.Sum256([]byte(username))
	passwordHash := sha256.Sum256([]byte(password))
	expectedUsernameHash := sha256.Sum256([]byte(username))
	expectedPasswordHash := sha256.Sum256([]byte(password))

	usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1)
	passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)

	return usernameMatch && passwordMatch
}
