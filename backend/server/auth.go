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
		authenticated := s.passBasicAuth(r)

		if authenticated {
			ctx := context.WithValue(r.Context(), authKey, true)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

func (s *Server) passBasicAuth(r *http.Request) bool {
	username, password, ok := r.BasicAuth()
	if !ok {
		return false
	}

	if s.username == "" && s.password == "" {
		s.logger.Warn().Msg("Config auth username and password not set, not authenticating")
		return false
	}

	// Hash for consistent timing
	usernameHash := sha256.Sum256([]byte(username))
	passwordHash := sha256.Sum256([]byte(password))
	expectedUsernameHash := sha256.Sum256([]byte(s.username))
	expectedPasswordHash := sha256.Sum256([]byte(s.password))

	usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1)
	passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)

	return usernameMatch && passwordMatch
}
