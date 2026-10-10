package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/evanofslack/analogdb/config"
	v1 "github.com/evanofslack/analogdb/internal/gen/proto/analytics/v1"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

const (
	testVisitorIP = "203.0.113.7"
	testVisitorUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36"
)

type recordingUiEvents struct {
	mu     sync.Mutex
	events []*v1.UiEvent
}

func (r *recordingUiEvents) Write(ctx context.Context, event *v1.UiEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
	return nil
}

func (r *recordingUiEvents) Close() error { return nil }

func (r *recordingUiEvents) take() []*v1.UiEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	events := r.events
	r.events = nil
	return events
}

type fakeSalts struct {
	salts map[string][]byte
	err   error
}

func (f *fakeSalts) DailySalt(ctx context.Context, day string, salt []byte) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	if stored, ok := f.salts[day]; ok {
		return stored, nil
	}
	f.salts[day] = salt
	return salt, nil
}

func newEventsServer(t *testing.T, cfg *config.Config) (*Server, *recordingUiEvents) {
	t.Helper()
	s := newRolesServer(t, cfg)
	rec := &recordingUiEvents{}
	s.UiEventService = rec
	return s, rec
}

func pageView() map[string]any {
	return map[string]any{
		"id":       uuid.NewString(),
		"name":     "page_view",
		"v":        1,
		"ts":       time.Now().UnixMilli(),
		"path":     "/post/123",
		"route":    "/post/[id]",
		"referrer": "news.ycombinator.com",
		"utm":      map[string]string{"source": "hn", "medium": "", "campaign": ""},
		"vw":       1280,
		"post_id":  123,
		"props":    map[string]any{},
	}
}

func batchBody(t *testing.T, events ...map[string]any) []byte {
	t.Helper()
	if events == nil {
		events = []map[string]any{}
	}
	body, err := json.Marshal(map[string]any{"events": events})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func postEvents(s *Server, body []byte, creds *pair, headers map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, eventsRoute, bytes.NewReader(body))
	r.RemoteAddr = "10.0.0.2:1234"
	if creds != nil {
		r.SetBasicAuth(creds.username, creds.password)
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)
	return w
}

func visitorHeaders() map[string]string {
	return map[string]string{visitorIPHeader: testVisitorIP, visitorUAHeader: testVisitorUA}
}

func TestEventsRoles(t *testing.T) {
	s, _ := newEventsServer(t, &config.Config{})
	tests := []struct {
		name  string
		creds *pair
		want  int
	}{
		{name: "anonymous", want: http.StatusUnauthorized},
		{name: "scraper", creds: &scraperPair, want: http.StatusForbidden},
		{name: "web", creds: &webPair, want: http.StatusNoContent},
		{name: "admin", creds: &adminPair, want: http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if w := postEvents(s, batchBody(t), tt.creds, nil); w.Code != tt.want {
				t.Fatalf("want %d, got %d: %s", tt.want, w.Code, w.Body.String())
			}
		})
	}

	if w := roleRequest(s, http.MethodPost, "/events", &webPair, ""); w.Code != http.StatusNotFound && w.Code != http.StatusMethodNotAllowed {
		t.Errorf("want no events route outside /v1, got %d", w.Code)
	}
}

func TestEventsValidation(t *testing.T) {
	s, rec := newEventsServer(t, &config.Config{})

	unknown := pageView()
	unknown["name"] = "mystery"
	badVersion := pageView()
	badVersion["v"] = 2
	badID := pageView()
	badID["id"] = "not-a-uuid"
	extraProp := pageView()
	extraProp["props"] = map[string]any{"extra": "x"}
	longString := map[string]any{
		"id": uuid.NewString(), "name": "search", "v": 1, "ts": time.Now().UnixMilli(),
		"props": map[string]any{"q": strings.Repeat("a", maxEventString+1)},
	}
	wrongType := map[string]any{
		"id": uuid.NewString(), "name": "search", "v": 1, "ts": time.Now().UnixMilli(),
		"props": map[string]any{"result_count": "ten"},
	}
	negativePost := pageView()
	negativePost["post_id"] = -1
	hugePost := pageView()
	hugePost["post_id"] = int64(1) << 40
	badRoute := pageView()
	badRoute["route"] = "post/[id]"
	tooManyProps := map[string]any{"id": uuid.NewString(), "name": "search", "v": 1, "props": map[string]any{}}
	for i := 0; i < maxEventProps+1; i++ {
		tooManyProps["props"].(map[string]any)[fmt.Sprintf("p%d", i)] = "x"
	}
	search := map[string]any{
		"id": uuid.NewString(), "name": "search", "v": 1, "ts": time.Now().UnixMilli(), "path": "/search",
		"props": map[string]any{
			"q": "portra 400", "q_norm": "portra 400", "mode": "text", "source": "typed",
			"filters": map[string]any{"nsfw": "false"}, "result_count": 12, "outcome": "ok", "page": 1,
		},
	}
	vital := map[string]any{
		"id": uuid.NewString(), "name": "web_vital", "v": 1, "ts": time.Now().UnixMilli(),
		"props": map[string]any{"metric": "CLS", "value": 0.05, "rating": "good"},
	}
	good := pageView()

	body := batchBody(t, good, unknown, badVersion, badID, extraProp, longString, wrongType, negativePost, hugePost, badRoute, tooManyProps, search, vital)
	body = bytes.Replace(body, []byte(`"events":[`), []byte(`"events":[{"name":"page_view","v":"one"},`), 1)
	if w := postEvents(s, body, &webPair, visitorHeaders()); w.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d: %s", w.Code, w.Body.String())
	}

	published := rec.take()
	if len(published) != 3 {
		t.Fatalf("want 3 published, got %d", len(published))
	}
	if published[0].EventId != good["id"] || published[1].EventName != "search" || published[2].EventName != "web_vital" {
		t.Errorf("want the valid events in order, got %+v", published)
	}
	if got := published[1].PropsJson; !strings.Contains(got, `"result_count":12`) || !strings.Contains(got, `"filters":{"nsfw":"false"}`) {
		t.Errorf("want props kept as JSON, got %s", got)
	}
	if got := published[0].PropsJson; got != "{}" {
		t.Errorf("want empty props as {}, got %s", got)
	}
	if got := published[2].PropsJson; !strings.Contains(got, `"value":0.05`) {
		t.Errorf("want number prop kept, got %s", got)
	}
	if published[0].PostId != 123 || published[0].ViewportWidth != 1280 || published[0].ReferrerHost != "news.ycombinator.com" || published[0].UtmSource != "hn" {
		t.Errorf("want envelope fields copied, got %+v", published[0])
	}

	stats := s.ui.stats.events
	if n := testutil.ToFloat64(stats.WithLabelValues("page_view", "accepted")); n != 1 {
		t.Errorf("want 1 accepted page_view, got %v", n)
	}
	if n := testutil.ToFloat64(stats.WithLabelValues("page_view", "dropped")); n != 6 {
		t.Errorf("want 6 dropped page_view, got %v", n)
	}
	if n := testutil.ToFloat64(stats.WithLabelValues("search", "dropped")); n != 3 {
		t.Errorf("want 3 dropped search, got %v", n)
	}
	if n := testutil.ToFloat64(stats.WithLabelValues(unknownEventName, "dropped")); n != 2 {
		t.Errorf("want 2 dropped unknown, got %v", n)
	}
	if n := testutil.ToFloat64(s.ui.stats.batches.WithLabelValues("ok")); n != 1 {
		t.Errorf("want 1 ok batch, got %v", n)
	}
}

func TestEventsBatchErrors(t *testing.T) {
	s, rec := newEventsServer(t, &config.Config{})

	tooMany := make([]map[string]any, maxEventsPerBatch+1)
	for i := range tooMany {
		tooMany[i] = pageView()
	}
	tests := []struct {
		name   string
		body   []byte
		want   int
		result string
	}{
		{name: "too many events", body: batchBody(t, tooMany...), want: http.StatusBadRequest, result: "bad_request"},
		{name: "invalid json", body: []byte(`{"events": [`), want: http.StatusBadRequest, result: "bad_request"},
		{name: "too large", body: []byte(`{"events": [], "pad": "` + strings.Repeat("a", 33<<10) + `"}`), want: http.StatusRequestEntityTooLarge, result: "too_large"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := testutil.ToFloat64(s.ui.stats.batches.WithLabelValues(tt.result))
			if w := postEvents(s, tt.body, &webPair, nil); w.Code != tt.want {
				t.Fatalf("want %d, got %d: %s", tt.want, w.Code, w.Body.String())
			}
			if n := testutil.ToFloat64(s.ui.stats.batches.WithLabelValues(tt.result)) - before; n != 1 {
				t.Errorf("want 1 %s batch, got %v", tt.result, n)
			}
		})
	}
	if got := rec.take(); len(got) != 0 {
		t.Errorf("want nothing published, got %d", len(got))
	}
}

func TestEventsClientTime(t *testing.T) {
	s, rec := newEventsServer(t, &config.Config{})
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	s.ui.now = func() time.Time { return now }

	recent := pageView()
	recent["ts"] = now.Add(-time.Hour).UnixMilli()
	past := pageView()
	past["ts"] = now.Add(-25 * time.Hour).UnixMilli()
	future := pageView()
	future["ts"] = now.Add(25 * time.Hour).UnixMilli()
	long := pageView()
	long["path"] = "/" + strings.Repeat("é", maxEventPath)

	if w := postEvents(s, batchBody(t, recent, past, future, long), &webPair, nil); w.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", w.Code)
	}
	published := rec.take()
	if len(published) != 4 {
		t.Fatalf("want 4 published, got %d", len(published))
	}
	if published[0].ClientTs != recent["ts"] {
		t.Errorf("want recent client time kept, got %d", published[0].ClientTs)
	}
	for i, e := range published[1:3] {
		if e.ClientTs != now.UnixMilli() || e.ReceivedTs != now.UnixMilli() {
			t.Errorf("event %d: want client time replaced by received time, got %d", i+1, e.ClientTs)
		}
	}
	if got := published[3].Path; len(got) > maxEventPath || !strings.HasPrefix(got, "/é") || strings.ContainsRune(got, '�') {
		t.Errorf("want path truncated to %d bytes on a rune boundary, got %d bytes", maxEventPath, len(got))
	}
}

func TestEventsVisitor(t *testing.T) {
	s, rec := newEventsServer(t, &config.Config{})
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	s.ui.now = func() time.Time { return now }
	salt := s.dailySalt(context.Background(), now)
	fromHeaders := visitorHash(salt, testVisitorIP, testVisitorUA)
	fromRequest := visitorHash(salt, "10.0.0.2", "Go-http-client/1.1")

	tests := []struct {
		name    string
		creds   *pair
		headers map[string]string
		want    string
	}{
		{name: "web with visitor headers", creds: &webPair, headers: visitorHeaders(), want: fromHeaders},
		{name: "admin ignores visitor headers", creds: &adminPair, headers: visitorHeaders(), want: fromRequest},
		{name: "web with bad visitor ip", creds: &webPair, headers: map[string]string{visitorIPHeader: "not-an-ip", visitorUAHeader: testVisitorUA}, want: fromRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{"User-Agent": "Go-http-client/1.1"}
			for k, v := range tt.headers {
				headers[k] = v
			}
			if w := postEvents(s, batchBody(t, pageView()), tt.creds, headers); w.Code != http.StatusNoContent {
				t.Fatalf("want 204, got %d", w.Code)
			}
			published := rec.take()
			if len(published) != 1 {
				t.Fatalf("want 1 published, got %d", len(published))
			}
			if got := published[0].VisitorId; got != tt.want {
				t.Errorf("want visitor %s, got %s", tt.want, got)
			}
		})
	}

	if len(fromHeaders) != 32 {
		t.Errorf("want 32 char visitor id, got %d", len(fromHeaders))
	}

	if w := postEvents(s, batchBody(t, pageView()), &webPair, visitorHeaders()); w.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", w.Code)
	}
	published := rec.take()
	if published[0].VisitorId != fromHeaders {
		t.Errorf("want the same visitor on the same day")
	}
	if published[0].DeviceType != "desktop" || published[0].Browser != "Chrome" || published[0].Os != "Windows" || published[0].IsBot {
		t.Errorf("want parsed user agent, got %s %s %s bot=%v", published[0].DeviceType, published[0].Browser, published[0].Os, published[0].IsBot)
	}
	encoded, err := json.Marshal(published[0])
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte(testVisitorIP)) || bytes.Contains(encoded, []byte("Mozilla")) {
		t.Errorf("want no raw ip or user agent in the message, got %s", encoded)
	}
	if !bytes.Contains(encoded, []byte(`"visitor_id"`)) || !bytes.Contains(encoded, []byte(`"props_json"`)) {
		t.Errorf("want snake case json keys, got %s", encoded)
	}

	now = now.Add(24 * time.Hour)
	if w := postEvents(s, batchBody(t, pageView()), &webPair, visitorHeaders()); w.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", w.Code)
	}
	if got := rec.take()[0].VisitorId; got == fromHeaders {
		t.Error("want a different visitor id with the next day's salt")
	}
}

func TestEventsBots(t *testing.T) {
	s := newTestServer(t, &config.Config{})
	tests := []struct {
		ua     string
		device string
		bot    bool
	}{
		{ua: testVisitorUA, device: "desktop"},
		{ua: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1", device: "mobile"},
		{ua: "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)", device: "bot", bot: true},
		{ua: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/120.0.0.0 Safari/537.36", device: "bot", bot: true},
		{ua: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome-Lighthouse", device: "bot", bot: true},
		{ua: "", device: "other"},
	}
	for _, tt := range tests {
		device, _, _, bot := s.parseUserAgent(tt.ua)
		if device != tt.device || bot != tt.bot {
			t.Errorf("%q: want %s bot=%v, got %s bot=%v", tt.ua, tt.device, tt.bot, device, bot)
		}
	}
}

func TestDailySalt(t *testing.T) {
	s := newTestServer(t, &config.Config{})
	day := time.Date(2026, 10, 10, 1, 0, 0, 0, time.UTC)

	local := s.dailySalt(context.Background(), day)
	if len(local) != saltBytes {
		t.Fatalf("want %d byte salt, got %d", saltBytes, len(local))
	}
	if again := s.dailySalt(context.Background(), day.Add(20*time.Hour)); !bytes.Equal(again, local) {
		t.Error("want the same salt within a UTC day")
	}
	if next := s.dailySalt(context.Background(), day.Add(23*time.Hour)); bytes.Equal(next, local) {
		t.Error("want a new salt on the next UTC day")
	}

	shared := []byte("shared salt from another instance")
	s.SaltService = &fakeSalts{salts: map[string][]byte{"2026-10-12": shared}}
	if got := s.dailySalt(context.Background(), day.Add(48*time.Hour)); !bytes.Equal(got, shared) {
		t.Errorf("want the stored salt, got %q", got)
	}

	s.SaltService = &fakeSalts{err: errors.New("redis down")}
	if got := s.dailySalt(context.Background(), day.Add(72*time.Hour)); len(got) != saltBytes {
		t.Errorf("want a local salt when the store fails, got %d bytes", len(got))
	}
}

func TestEventsRateLimit(t *testing.T) {
	cfg := &config.Config{}
	cfg.App.RateLimitEnabled = true
	cfg.App.RateLimitWebPerMinute = 2
	cfg.App.RateLimitEventsPerMinute = 3
	s, _ := newEventsServer(t, cfg)

	for i := 0; i < 3; i++ {
		if w := postEvents(s, batchBody(t), &webPair, nil); w.Code != http.StatusNoContent {
			t.Fatalf("events request %d: want 204, got %d", i, w.Code)
		}
	}
	if w := postEvents(s, batchBody(t), &webPair, nil); w.Code != http.StatusTooManyRequests {
		t.Errorf("want 429 at the events cap, got %d", w.Code)
	}
	for i := 0; i < 2; i++ {
		if code := roleRequest(s, http.MethodGet, "/ping", &webPair, "").Code; code != http.StatusOK {
			t.Fatalf("web request %d: want the web bucket untouched by events, got %d", i, code)
		}
	}
	if n := testutil.ToFloat64(s.stats.rateLimited.WithLabelValues("web_events")); n != 1 {
		t.Errorf("want 1 web_events rejection counted, got %v", n)
	}
	if n := testutil.ToFloat64(s.stats.rateLimited.WithLabelValues("web")); n != 0 {
		t.Errorf("want no web rejections, got %v", n)
	}
	if w := postEvents(s, batchBody(t), nil, nil); w.Code != http.StatusUnauthorized {
		t.Errorf("want anonymous 401, got %d", w.Code)
	}
}

func TestEventsSkipRequestLog(t *testing.T) {
	s, _ := newEventsServer(t, &config.Config{})
	rec := &recordingEvents{}
	s.EventService = rec

	if w := postEvents(s, batchBody(t, pageView()), &webPair, nil); w.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", w.Code)
	}
	if len(rec.events) != 0 {
		t.Errorf("want no request event for %s, got %d", eventsRoute, len(rec.events))
	}
	roleRequest(s, http.MethodGet, "/ping", &webPair, "")
	if len(rec.events) != 1 {
		t.Errorf("want other routes still logged, got %d", len(rec.events))
	}
}
