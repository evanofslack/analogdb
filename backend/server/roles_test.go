package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/config"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

type pair struct{ username, password string }

var (
	adminPair         = pair{"admin", "admin-pass"}
	scraperPair       = pair{"scraper", "scraper-pass"}
	webPair           = pair{"web", "web-pass"}
	legacyAdminPair   = pair{testUsername, testPassword}
	legacyWebPair     = pair{"legacy-web", "legacy-web-pass"}
	wrongPasswordPair = pair{"admin", "wrong"}
)

func newRolesServer(t *testing.T, cfg *config.Config) *Server {
	t.Helper()
	cfg.Auth.AdminUsername, cfg.Auth.AdminPassword = adminPair.username, adminPair.password
	cfg.Auth.ScraperUsername, cfg.Auth.ScraperPassword = scraperPair.username, scraperPair.password
	cfg.Auth.WebUsername, cfg.Auth.WebPassword = webPair.username, webPair.password
	cfg.Auth.RateLimitUsername, cfg.Auth.RateLimitPassword = legacyWebPair.username, legacyWebPair.password
	return newTestServer(t, cfg)
}

func roleRequest(s *Server, method, path string, creds *pair, remote string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, nil)
	if creds != nil {
		r.SetBasicAuth(creds.username, creds.password)
	}
	if remote != "" {
		r.RemoteAddr = remote
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)
	return w
}

func TestPrincipal(t *testing.T) {
	s := newRolesServer(t, &config.Config{})

	var got *principal
	handler := s.principal(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = principalFrom(r)
	}))

	tests := []struct {
		name   string
		creds  *pair
		role   role
		legacy bool
	}{
		{name: "admin", creds: &adminPair, role: roleAdmin},
		{name: "scraper", creds: &scraperPair, role: roleScraper},
		{name: "web", creds: &webPair, role: roleWeb},
		{name: "legacy admin", creds: &legacyAdminPair, role: roleAdmin, legacy: true},
		{name: "legacy rate limit", creds: &legacyWebPair, role: roleWeb, legacy: true},
		{name: "wrong password", creds: &wrongPasswordPair},
		{name: "empty pair never matches", creds: &pair{"", ""}},
		{name: "no credentials"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got = nil
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.creds != nil {
				r.SetBasicAuth(tt.creds.username, tt.creds.password)
			}
			handler.ServeHTTP(httptest.NewRecorder(), r)
			if tt.role == "" {
				if got != nil {
					t.Fatalf("want anonymous, got %+v", got)
				}
				return
			}
			if got == nil || got.role != tt.role || got.legacy != tt.legacy {
				t.Fatalf("want role %s legacy %v, got %+v", tt.role, tt.legacy, got)
			}
		})
	}

	if n := testutil.ToFloat64(s.stats.authRequests.WithLabelValues("admin", "true")); n != 1 {
		t.Errorf("want 1 legacy admin request counted, got %v", n)
	}
	if n := testutil.ToFloat64(s.stats.authRequests.WithLabelValues("invalid", "false")); n != 2 {
		t.Errorf("want 2 invalid requests counted, got %v", n)
	}
}

func TestRequireRoles(t *testing.T) {
	s := newRolesServer(t, &config.Config{})
	s.AdminService = mockAdmin{}
	s.VectorCounter = mockVectors{}
	s.AnalyticsService = mockAnalytics{}
	s.ExtractionService = &mockExtractions{}

	tests := []struct {
		name   string
		method string
		path   string
		creds  *pair
		want   int
	}{
		{name: "anonymous delete", method: http.MethodDelete, path: "/v1/post/1", want: http.StatusUnauthorized},
		{name: "wrong password delete", method: http.MethodDelete, path: "/v1/post/1", creds: &wrongPasswordPair, want: http.StatusUnauthorized},
		{name: "web delete", method: http.MethodDelete, path: "/v1/post/1", creds: &webPair, want: http.StatusForbidden},
		{name: "scraper delete", method: http.MethodDelete, path: "/v1/post/1", creds: &scraperPair, want: http.StatusForbidden},
		{name: "web patch", method: http.MethodPatch, path: "/v1/post/1", creds: &webPair, want: http.StatusForbidden},
		{name: "web scrape route", method: http.MethodGet, path: "/v1/scrape/vectors/missing", creds: &webPair, want: http.StatusForbidden},
		{name: "scraper extractions", method: http.MethodGet, path: "/v1/admin/extractions", creds: &scraperPair, want: http.StatusOK},
		{name: "web extractions", method: http.MethodGet, path: "/v1/admin/extractions", creds: &webPair, want: http.StatusForbidden},
		{name: "scraper overview", method: http.MethodGet, path: "/v1/admin/overview", creds: &scraperPair, want: http.StatusForbidden},
		{name: "admin overview", method: http.MethodGet, path: "/v1/admin/overview", creds: &adminPair, want: http.StatusOK},
		{name: "web analytics", method: http.MethodGet, path: "/v1/admin/analytics", creds: &webPair, want: http.StatusForbidden},
		{name: "admin analytics", method: http.MethodGet, path: "/v1/admin/analytics", creds: &adminPair, want: http.StatusOK},
		{name: "admin extractions", method: http.MethodGet, path: "/v1/admin/extractions", creds: &adminPair, want: http.StatusOK},
		{name: "legacy admin overview", method: http.MethodGet, path: "/v1/admin/overview", creds: &legacyAdminPair, want: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := roleRequest(s, tt.method, tt.path, tt.creds, "")
			if w.Code != tt.want {
				t.Fatalf("want %d, got %d: %s", tt.want, w.Code, w.Body.String())
			}
			if tt.want == http.StatusUnauthorized && w.Header().Get("WWW-Authenticate") == "" {
				t.Error("want WWW-Authenticate on 401")
			}
		})
	}
}

var routeParam = regexp.MustCompile(`\{[^}]+\}`)

// TestProtectedRoutes walks the router so a new write, admin or scrape route
// can't ship without a role check
func TestProtectedRoutes(t *testing.T) {
	s := newRolesServer(t, &config.Config{})

	public := map[string]bool{
		"POST /v1/search/image":  true,
		"POST /search/image":     true,
		"POST /v1/post/1/report": true,
		"POST /post/1/report":    true,
	}
	webAllowed := map[string]bool{
		"POST /v1/events": true,
	}
	err := chi.Walk(s.router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if strings.Contains(route, "*") || method == http.MethodOptions || method == http.MethodHead {
			return nil
		}
		path := strings.TrimSuffix(routeParam.ReplaceAllString(route, "1"), "/")
		if path == "" {
			path = "/"
		}
		sensitive := method != http.MethodGet || strings.Contains(path, "/admin") || strings.Contains(path, "/scrape")
		if !sensitive || public[method+" "+path] {
			return nil
		}

		if w := roleRequest(s, method, path, nil, ""); w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: want 401 without credentials, got %d", method, route, w.Code)
		}
		forbidden := &webPair
		if webAllowed[method+" "+path] {
			forbidden = &scraperPair
		}
		if w := roleRequest(s, method, path, forbidden, ""); w.Code != http.StatusForbidden {
			t.Errorf("%s %s: want 403 for %s, got %d", method, route, forbidden.username, w.Code)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRateLimitByRole(t *testing.T) {
	cfg := &config.Config{}
	cfg.App.RateLimitEnabled = true
	cfg.App.RateLimitWebPerMinute = 5
	s := newRolesServer(t, cfg)

	for i := 0; i < 5; i++ {
		remote := []string{"1.1.1.1:1234", "2.2.2.2:1234"}[i%2]
		if code := roleRequest(s, http.MethodGet, "/ping", &webPair, remote).Code; code != http.StatusOK {
			t.Fatalf("web request %d: want 200, got %d", i, code)
		}
	}
	if code := roleRequest(s, http.MethodGet, "/ping", &webPair, "3.3.3.3:1234").Code; code != http.StatusTooManyRequests {
		t.Errorf("want 429 once web passes its shared cap, got %d", code)
	}
	if code := roleRequest(s, http.MethodGet, "/ping", &legacyWebPair, "4.4.4.4:1234").Code; code != http.StatusTooManyRequests {
		t.Errorf("want legacy rate limit pair to share the web cap, got %d", code)
	}
	if code := roleRequest(s, http.MethodGet, "/ping", nil, "3.3.3.3:1234").Code; code != http.StatusOK {
		t.Errorf("want anonymous unaffected by the web cap, got %d", code)
	}
	for i := 0; i < rateLimit*2; i++ {
		if code := roleRequest(s, http.MethodGet, "/ping", &scraperPair, "5.5.5.5:1234").Code; code != http.StatusOK {
			t.Fatalf("scraper request %d: want 200, got %d", i, code)
		}
	}
	if n := testutil.ToFloat64(s.stats.rateLimited.WithLabelValues("web")); n != 2 {
		t.Errorf("want 2 web rejections counted, got %v", n)
	}
}

func TestRateLimitWebOff(t *testing.T) {
	cfg := &config.Config{}
	cfg.App.RateLimitEnabled = true
	cfg.App.RateLimitWebPerMinute = 0
	s := newRolesServer(t, cfg)

	for i := 0; i < rateLimit*2; i++ {
		if code := roleRequest(s, http.MethodGet, "/ping", &webPair, "1.1.1.1:1234").Code; code != http.StatusOK {
			t.Fatalf("web request %d: want 200 with the cap off, got %d", i, code)
		}
	}
}

type blockingSearch struct {
	analogdb.SearchService
	started chan struct{}
	block   chan struct{}
}

func (b *blockingSearch) SearchText(ctx context.Context, filter *analogdb.SearchFilter) ([]analogdb.SearchHit, bool, error) {
	b.started <- struct{}{}
	<-b.block
	return nil, false, nil
}

func TestTextSearchSlots(t *testing.T) {
	s := newTestServer(t, &config.Config{})
	s.textWait = 50 * time.Millisecond
	inner := &blockingSearch{started: make(chan struct{}, searchTextSlots), block: make(chan struct{})}
	search := s.LimitTextSearch(inner)

	done := make(chan error, searchTextSlots)
	for range searchTextSlots {
		go func() {
			_, _, err := search.SearchText(context.Background(), &analogdb.SearchFilter{})
			done <- err
		}()
	}
	for range searchTextSlots {
		<-inner.started
	}

	_, _, err := search.SearchText(context.Background(), &analogdb.SearchFilter{})
	if code := analogdb.ErrorCode(err); code != analogdb.ERRUNAVAILABLE {
		t.Errorf("want %s with all slots busy, got %v", analogdb.ERRUNAVAILABLE, err)
	}

	close(inner.block)
	for range searchTextSlots {
		if err := <-done; err != nil {
			t.Errorf("want blocked searches to finish, got %v", err)
		}
	}

	// a free slot passes through
	inner.started = make(chan struct{}, 1)
	if _, _, err := search.SearchText(context.Background(), &analogdb.SearchFilter{}); err != nil {
		t.Errorf("want search with a free slot to pass, got %v", err)
	}
}
