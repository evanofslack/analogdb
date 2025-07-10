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
		authenticated := s.passBasicAuth(s.config.Auth.Username, s.config.Auth.Password, r)

		if authenticated {
			ctx := context.WithValue(r.Context(), authKey, true)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

func (s *Server) passBasicAuth(expectUsername, expectPassword string, r *http.Request) bool {
	username, password, ok := r.BasicAuth()
	if !ok {
		return false
	}

	if expectUsername == "" && expectPassword == "" {
		s.logger.Warn().Msg("Auth expected username and password not set, not authenticating")
		return false
	}

	// Hash for consistent timing
	usernameHash := sha256.Sum256([]byte(username))
	passwordHash := sha256.Sum256([]byte(password))
	expectUsernameHash := sha256.Sum256([]byte(expectUsername))
	expectPasswordHash := sha256.Sum256([]byte(expectPassword))

	usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectUsernameHash[:]) == 1)
	passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectPasswordHash[:]) == 1)

	return usernameMatch && passwordMatch
}
