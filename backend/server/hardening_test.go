package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/config"
	"github.com/evanofslack/analogdb/events"
	"github.com/evanofslack/analogdb/logger"
	"github.com/evanofslack/analogdb/metrics"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

const (
	testUsername = "user"
	testPassword = "pass"
)

func newTestServer(t *testing.T, cfg *config.Config) *Server {
	t.Helper()
	logger, err := logger.New("error", "debug", "analogdb-test")
	if err != nil {
		t.Fatal(err)
	}
	metrics, err := metrics.New(logger)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Auth.Username = testUsername
	cfg.Auth.Password = testPassword
	s := New("0", logger, metrics, cfg)
	s.ReadyService = &mockReady{}
	s.EventService = events.NewNoop(logger)
	return s
}

func TestResolveClientIP(t *testing.T) {
	trusted, invalid := parseTrustedProxies([]string{"172.16.0.0/12", "127.0.0.1", "bogus"})
	if len(invalid) != 1 || invalid[0] != "bogus" {
		t.Fatalf("want bogus to be invalid, got %v", invalid)
	}

	tests := []struct {
		name   string
		remote string
		xff    []string
		realIP string
		want   string
	}{
		{"untrusted remote ignores xff", "203.0.113.9:1234", []string{"1.1.1.1"}, "", "203.0.113.9"},
		{"trusted proxy single hop", "172.18.0.2:1234", []string{"1.1.1.1"}, "", "1.1.1.1"},
		{"trusted proxy takes rightmost untrusted hop", "172.18.0.2:1234", []string{"6.6.6.6, 1.1.1.1, 172.18.0.5"}, "", "1.1.1.1"},
		{"trusted proxy multiple headers", "127.0.0.1:1234", []string{"6.6.6.6", "2.2.2.2"}, "", "2.2.2.2"},
		{"trusted proxy without xff", "172.18.0.2:1234", nil, "", "172.18.0.2"},
		{"trusted proxy invalid hop", "172.18.0.2:1234", []string{"1.1.1.1, garbage"}, "", "172.18.0.2"},
		{"x-real-ip is ignored", "203.0.113.9:1234", nil, "6.6.6.6", "203.0.113.9"},
		{"x-real-ip ignored behind proxy", "172.18.0.2:1234", []string{"1.1.1.1"}, "6.6.6.6", "1.1.1.1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/ping", nil)
			r.RemoteAddr = tt.remote
			for _, v := range tt.xff {
				r.Header.Add("X-Forwarded-For", v)
			}
			if tt.realIP != "" {
				r.Header.Set("X-Real-IP", tt.realIP)
			}
			if got := resolveClientIP(trusted, r); got != tt.want {
				t.Errorf("want %s, got %s", tt.want, got)
			}
		})
	}
}

func TestRateLimitPerClientIP(t *testing.T) {
	cfg := &config.Config{}
	cfg.App.RateLimitEnabled = true
	cfg.HTTP.TrustedProxies = []string{"172.16.0.0/12"}
	s := newTestServer(t, cfg)

	do := func(remote, xff string) int {
		r := httptest.NewRequest(http.MethodGet, "/ping", nil)
		r.RemoteAddr = remote
		if xff != "" {
			r.Header.Set("X-Forwarded-For", xff)
		}
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, r)
		return w.Code
	}

	for i := 0; i < rateLimit; i++ {
		if code := do("172.18.0.2:1234", "1.1.1.1"); code != http.StatusOK {
			t.Fatalf("request %d: want 200, got %d", i, code)
		}
	}
	if code := do("172.18.0.2:1234", "1.1.1.1"); code != http.StatusTooManyRequests {
		t.Errorf("want 429 for first client, got %d", code)
	}
	if code := do("172.18.0.2:1234", "2.2.2.2"); code != http.StatusOK {
		t.Errorf("want 200 for second client, got %d", code)
	}
	if got := testutil.ToFloat64(s.stats.rateLimited); got != 1 {
		t.Errorf("want 1 rate limited request, got %v", got)
	}
}

func TestCORSPreflightAnyOrigin(t *testing.T) {
	s := newTestServer(t, &config.Config{})
	r := httptest.NewRequest(http.MethodOptions, "/v1/posts", nil)
	r.Header.Set("Origin", "https://evil.example")
	r.Header.Set("Access-Control-Request-Method", http.MethodGet)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("want allow origin *, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("want no allow credentials header, got %q", got)
	}
}

func TestBodyTooLarge(t *testing.T) {
	s := newTestServer(t, &config.Config{})
	s.PostService = &mockPostService{}
	s.SimilarityService = &mockFailSimilarityService{}

	body := `{"title":"` + strings.Repeat("a", maxBodyBytes) + `"}`
	r := httptest.NewRequest(http.MethodPost, "/v1/post", strings.NewReader(body))
	r.SetBasicAuth(testUsername, testPassword)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)

	if want, got := http.StatusRequestEntityTooLarge, w.Code; got != want {
		t.Errorf("want status %d, got %d", want, got)
	}
}

func TestUnknownJSONFieldIsCounted(t *testing.T) {
	s := newTestServer(t, &config.Config{})
	postService := &mockPostService{}
	s.PostService = postService
	s.SimilarityService = &mockFailSimilarityService{}

	body, err := json.Marshal(map[string]any{"title": "known", "not_a_field": 1})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "/v1/post", bytes.NewReader(body))
	r.SetBasicAuth(testUsername, testPassword)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)

	if want, got := http.StatusCreated, w.Code; got != want {
		t.Fatalf("want status %d, got %d", want, got)
	}
	if len(postService.created) != 1 || postService.created[0].Title != "known" {
		t.Errorf("want post created with known fields")
	}
	if got := testutil.ToFloat64(s.stats.unknownJSONFields.WithLabelValues("/v1/post")); got != 1 {
		t.Errorf("want 1 unknown field, got %v", got)
	}
}

type mockReadyErr struct{}

func (m *mockReadyErr) Readyz(ctx context.Context) error {
	return errors.New("down")
}

func TestReadyDependencies(t *testing.T) {
	tests := []struct {
		name     string
		postgres bool
		redis    bool
		want     int
		body     map[string]string
	}{
		{"redis down still ready", true, false, http.StatusOK, map[string]string{"postgres": "ok", "redis": "down", "weaviate": "ok"}},
		{"postgres down not ready", false, true, http.StatusServiceUnavailable, map[string]string{"postgres": "down", "redis": "ok", "weaviate": "ok"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestServer(t, &config.Config{})
			s.ReadyService = &mockReadyErr{}
			if tt.postgres {
				s.ReadyService = &mockReady{}
			}
			s.CacheReadyService = &mockReadyErr{}
			if tt.redis {
				s.CacheReadyService = &mockReady{}
			}
			s.VectorReadyService = &mockReady{}

			r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			w := httptest.NewRecorder()
			s.router.ServeHTTP(w, r)

			if got := w.Code; got != tt.want {
				t.Errorf("want status %d, got %d", tt.want, got)
			}
			var body map[string]string
			if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			for k, v := range tt.body {
				if body[k] != v {
					t.Errorf("want %s=%s, got %s", k, v, body[k])
				}
			}
		})
	}
}

func TestReadyShuttingDown(t *testing.T) {
	s := newTestServer(t, &config.Config{})
	s.healthy.Store(false)
	r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)

	if want, got := http.StatusServiceUnavailable, w.Code; got != want {
		t.Errorf("want status %d, got %d", want, got)
	}
}

func TestRunPortInUse(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	logger, err := logger.New("error", "debug", "analogdb-test")
	if err != nil {
		t.Fatal(err)
	}
	metrics, err := metrics.New(logger)
	if err != nil {
		t.Fatal(err)
	}
	s := New(port, logger, metrics, &config.Config{})
	if err := s.Run(); err == nil {
		t.Error("want error running on a bound port")
		_ = s.Close()
	}
}

func TestLegacyRequestsMetric(t *testing.T) {
	s := newTestServer(t, &config.Config{})

	tests := []struct {
		path      string
		userAgent string
	}{
		{"/encode", "analogdb-scraper/1.0"},
		{"/encode", "analogdb-web/1.0"},
		{"/encode", "curl/8.0"},
		{"/v1/encode", "analogdb-scraper/1.0"},
	}
	for _, tt := range tests {
		r := httptest.NewRequest(http.MethodPut, tt.path, nil)
		r.Header.Set("User-Agent", tt.userAgent)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, r)
	}

	for _, client := range []string{"scraper", "web", "other"} {
		if got := testutil.ToFloat64(s.stats.legacyRequests.WithLabelValues("/encode", client)); got != 1 {
			t.Errorf("want 1 legacy request for %s, got %v", client, got)
		}
	}
	if got := testutil.CollectAndCount(s.stats.legacyRequests); got != 3 {
		t.Errorf("want 3 legacy series, got %d", got)
	}
}

func TestDebugWebsocketRequiresAuth(t *testing.T) {
	s := newTestServer(t, &config.Config{})
	r := httptest.NewRequest(http.MethodGet, "/debug/statsviz/ws", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)

	if want, got := http.StatusUnauthorized, w.Code; got != want {
		t.Errorf("want status %d, got %d", want, got)
	}
}

type mockSlowSimilarityService struct {
	analogdb.SimilarityService
	delay time.Duration
}

func (m *mockSlowSimilarityService) EncodePost(ctx context.Context, id int) error {
	time.Sleep(m.delay)
	return nil
}

func TestEncodeExtendsWriteDeadline(t *testing.T) {
	cfg := &config.Config{}
	cfg.Tracing.Enabled = true
	s := newTestServer(t, cfg)
	s.SimilarityService = &mockSlowSimilarityService{delay: 300 * time.Millisecond}
	s.server.WriteTimeout = 100 * time.Millisecond

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = s.server.Serve(ln) }()
	defer func() { _ = s.Close() }()

	req, err := http.NewRequest(http.MethodPut, "http://"+ln.Addr().String()+"/v1/encode", strings.NewReader(`{"ids":[1]}`))
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth(testUsername, testPassword)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if want, got := http.StatusOK, resp.StatusCode; got != want {
		t.Errorf("want status %d, got %d", want, got)
	}
}
