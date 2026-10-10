package server

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"slices"
	"strconv"

	"github.com/evanofslack/analogdb"
)

type contextKey string

const (
	authStateKey contextKey = "auth_state"
	principalKey contextKey = "principal"
)

type role string

const (
	roleAdmin   role = "admin"
	roleScraper role = "scraper"
	roleWeb     role = "web"

	roleInvalid role = "invalid"
)

type principal struct {
	role   role
	legacy bool
}

// authState lets logRequests see whether auth passed further down the chain
type authState struct {
	ok bool
}

func withAuthState(r *http.Request) (*http.Request, *authState) {
	state := &authState{}
	return r.WithContext(context.WithValue(r.Context(), authStateKey, state)), state
}

func principalFrom(r *http.Request) *principal {
	p, _ := r.Context().Value(principalKey).(*principal)
	return p
}

// principal resolves basic auth to a role. Wrong credentials stay anonymous
func (s *Server) principal(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		// hash for consistent timing, and check every pair with no early return
		usernameHash := sha256.Sum256([]byte(username))
		passwordHash := sha256.Sum256([]byte(password))
		var match *principal
		for _, cred := range s.config.Auth.Credentials() {
			expectUsernameHash := sha256.Sum256([]byte(cred.Username))
			expectPasswordHash := sha256.Sum256([]byte(cred.Password))
			usernameMatch := subtle.ConstantTimeCompare(usernameHash[:], expectUsernameHash[:]) == 1
			passwordMatch := subtle.ConstantTimeCompare(passwordHash[:], expectPasswordHash[:]) == 1
			if usernameMatch && passwordMatch && match == nil {
				match = &principal{role: role(cred.Role), legacy: cred.Legacy}
			}
		}

		if match == nil {
			s.stats.authRequests.WithLabelValues(string(roleInvalid), "false").Inc()
			next.ServeHTTP(w, r)
			return
		}
		s.stats.authRequests.WithLabelValues(string(match.role), strconv.FormatBool(match.legacy)).Inc()
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey, match)))
	})
}

// require lets admin and the given roles through. Anonymous callers get 401,
// any other role 403
func (s *Server) require(roles ...role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := principalFrom(r)
			if p == nil {
				w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			if p.role != roleAdmin && !slices.Contains(roles, p.role) {
				s.writeError(w, r, &analogdb.Error{Code: analogdb.ERRFORBIDDEN, Message: "forbidden"})
				return
			}
			if state, ok := r.Context().Value(authStateKey).(*authState); ok {
				state.ok = true
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) logCredentials() {
	creds := s.config.Auth.Credentials()
	names := make([]string, 0, len(creds))
	hasAdmin := false
	for _, cred := range creds {
		names = append(names, cred.Name)
		hasAdmin = hasAdmin || role(cred.Role) == roleAdmin
	}
	s.logger.Info("Auth credentials enabled", "pairs", names)
	if !hasAdmin {
		s.logger.Error("Config admin auth username and password not set!")
	}
}
