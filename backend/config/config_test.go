package config

import (
	"strings"
	"testing"
)

func TestAuthCredentials(t *testing.T) {
	a := Auth{
		AdminUsername:     "admin",
		AdminPassword:     "a",
		ScraperUsername:   "scraper",
		WebUsername:       "web",
		WebPassword:       "w",
		RateLimitUsername: "rl",
		RateLimitPassword: "r",
	}
	var names []string
	for _, c := range a.Credentials() {
		names = append(names, c.Name)
	}
	// scraper has no password and legacy admin is unset, so both are disabled
	if got := strings.Join(names, ","); got != "admin,web,legacy rate limit" {
		t.Errorf("want admin,web,legacy rate limit, got %s", got)
	}
}

func TestAuthValidate(t *testing.T) {
	tests := []struct {
		name     string
		auth     Auth
		wantErr  string
		warnings int
	}{
		{
			name: "distinct pairs",
			auth: Auth{AdminUsername: "admin", AdminPassword: "a", WebUsername: "web", WebPassword: "w"},
		},
		{
			name:    "web equals admin",
			auth:    Auth{AdminUsername: "admin", AdminPassword: "a", WebUsername: "admin", WebPassword: "a"},
			wantErr: "auth: admin and web credentials are the same",
		},
		{
			name:    "legacy rate limit equals legacy admin",
			auth:    Auth{Username: "u", Password: "p", RateLimitUsername: "u", RateLimitPassword: "p"},
			wantErr: "auth: legacy admin and legacy rate limit credentials are the same",
		},
		{
			name:     "same username different passwords",
			auth:     Auth{AdminUsername: "analogdb", AdminPassword: "a", ScraperUsername: "analogdb", ScraperPassword: "s"},
			warnings: 1,
		},
		{
			name: "empty pairs ignored",
			auth: Auth{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, err := tt.auth.Validate()
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("want error %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("want no error, got %v", err)
			}
			if len(warnings) != tt.warnings {
				t.Errorf("want %d warnings, got %v", tt.warnings, warnings)
			}
		})
	}
}
