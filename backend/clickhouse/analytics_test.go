package clickhouse

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/evanofslack/analogdb"
)

type uiRow struct {
	ago      time.Duration
	id       string
	name     string
	visitor  string
	path     string
	route    string
	referrer string
	source   string
	campaign string
	device   string
	browser  string
	country  string
	bot      bool
	postID   uint32
	props    string
}

var uiRows = []uiRow{
	{ago: time.Hour, id: "00000000-0000-0000-0000-000000000001", name: "page_view", visitor: "v1", path: "/post/12", route: "/post/[id]", referrer: "news.ycombinator.com", source: "reddit", campaign: "launch", device: "desktop", browser: "Firefox", postID: 12},
	{ago: time.Hour, id: "00000000-0000-0000-0000-000000000001", name: "page_view", visitor: "v1", path: "/post/12", route: "/post/[id]", referrer: "news.ycombinator.com", source: "reddit", campaign: "launch", device: "desktop", browser: "Firefox", postID: 12},
	{ago: 2 * time.Hour, id: "00000000-0000-0000-0000-000000000002", name: "page_view", visitor: "v1", path: "/", route: "/", referrer: "analogdb.com", device: "desktop", browser: "Firefox", country: "NL"},
	{ago: 3 * time.Hour, id: "00000000-0000-0000-0000-000000000003", name: "page_view", visitor: "v2", path: "/post/12", route: "/post/[id]", referrer: "www.analogdb.com", device: "mobile", browser: "Safari", postID: 12},
	{ago: time.Hour, id: "00000000-0000-0000-0000-000000000004", name: "page_view", visitor: "v3", path: "/", route: "/", device: "desktop", browser: "Chrome", bot: true},
	{ago: 10 * time.Minute, id: "00000000-0000-0000-0000-000000000005", name: "page_view", visitor: "v1", path: "/films", route: "/films", device: "desktop", browser: "Firefox"},
	{ago: 25 * time.Hour, id: "00000000-0000-0000-0000-000000000006", name: "page_view", visitor: "v1", path: "/", route: "/", device: "desktop", browser: "Firefox"},
	{ago: 50 * time.Hour, id: "00000000-0000-0000-0000-000000000007", name: "page_view", visitor: "v4", path: "/", route: "/", device: "desktop", browser: "Firefox"},
	{ago: 10 * 24 * time.Hour, id: "00000000-0000-0000-0000-000000000008", name: "page_view", visitor: "v5", path: "/", route: "/", device: "desktop", browser: "Firefox"},
	{ago: time.Hour, id: "00000000-0000-0000-0000-000000000009", name: "web_vital", visitor: "v1", path: "/post/12", route: "/post/[id]", props: `{"metric":"LCP","value":2000,"rating":"good"}`},
	{ago: time.Hour, id: "00000000-0000-0000-0000-000000000010", name: "web_vital", visitor: "v2", path: "/post/12", route: "/post/[id]", props: `{"metric":"LCP","value":3000,"rating":"needs-improvement"}`},
	{ago: time.Hour, id: "00000000-0000-0000-0000-000000000011", name: "web_vital", visitor: "v2", path: "/post/12", route: "/post/[id]", props: `{"metric":"CLS","value":0.05,"rating":"good"}`},
	{ago: time.Hour, id: "00000000-0000-0000-0000-000000000011", name: "web_vital", visitor: "v2", path: "/post/12", route: "/post/[id]", props: `{"metric":"CLS","value":0.05,"rating":"good"}`},
	{ago: time.Hour, id: "00000000-0000-0000-0000-000000000012", name: "web_vital", visitor: "v3", path: "/post/12", route: "/post/[id]", bot: true, props: `{"metric":"LCP","value":9000,"rating":"poor"}`},
}

// mustCreateUIEvents creates ui_events with the consumer's DDL and loads uiRows
func mustCreateUIEvents(t *testing.T, db *DB, now time.Time) {
	t.Helper()
	ctx := context.Background()

	schema, err := os.ReadFile(filepath.Join("testdata", "ui_events.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.conn.Exec(ctx, string(schema)); err != nil {
		t.Fatal(err)
	}

	batch, err := db.conn.PrepareBatch(ctx, `INSERT INTO ui_events (event_id, event_name, visitor_id, client_ts, received_ts, path, route, referrer_host, utm_source, utm_campaign, device_type, browser, country, is_bot, post_id, props)`)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range uiRows {
		ts := now.Add(-row.ago)
		if err := batch.Append(row.id, row.name, row.visitor, ts, ts, row.path, row.route, row.referrer, row.source, row.campaign, row.device, row.browser, row.country, row.bot, row.postID, row.props); err != nil {
			t.Fatal(err)
		}
	}
	if err := batch.Send(); err != nil {
		t.Fatal(err)
	}
}

// wantVisitorDays counts distinct UTC day and visitor pairs of human page views
func wantVisitorDays(now time.Time, from, to time.Duration) int64 {
	seen := map[string]bool{}
	for _, row := range uiRows {
		if row.name != "page_view" || row.bot || row.ago > from || row.ago <= to {
			continue
		}
		seen[now.Add(-row.ago).UTC().Format("2006-01-02")+row.visitor] = true
	}
	return int64(len(seen))
}

func TestAnalytics(t *testing.T) {
	db := mustOpen(t)
	ctx := context.Background()

	t.Run("missing table", func(t *testing.T) {
		if _, err := db.Analytics(ctx, analogdb.TrafficDay); err == nil {
			t.Error("want an error before the table exists")
		}
	})

	now := time.Now()
	mustCreateUIEvents(t, db, now)
	day := 24 * time.Hour

	t.Run("day", func(t *testing.T) {
		analytics, err := db.Analytics(ctx, analogdb.TrafficDay)
		if err != nil {
			t.Fatal(err)
		}
		if analytics.Bucket != "hour" {
			t.Fatalf("want hour buckets, got %s", analytics.Bucket)
		}

		summary := analytics.Summary
		if summary.Current.PageViews != 4 {
			t.Errorf("want 4 page views without bots and duplicates, got %d", summary.Current.PageViews)
		}
		if want := wantVisitorDays(now, day, 0); summary.Current.Visitors != want {
			t.Errorf("want %d visitor days, got %d", want, summary.Current.Visitors)
		}
		if summary.Current.BotViews != 1 {
			t.Errorf("want 1 bot view, got %d", summary.Current.BotViews)
		}
		if summary.Previous.PageViews != 1 || summary.Previous.Visitors != 1 || summary.Previous.BotViews != 0 {
			t.Errorf("want one page view in the previous day, got %+v", summary.Previous)
		}
		if summary.Live != (analogdb.ViewCounts{PageViews: 1, Visitors: 1}) {
			t.Errorf("want one live view, got %+v", summary.Live)
		}

		mustFilled(t, analyticsTimes(analytics.Series), time.Hour, 24)
		var views int64
		for _, b := range analytics.Series {
			views += b.PageViews
		}
		if views != 4 {
			t.Errorf("want 4 page views in series, got %d", views)
		}

		top := analytics.Top
		one := analogdb.ViewCounts{PageViews: 1, Visitors: 1}
		if len(top.Pages) != 3 || top.Pages[0].Name != "/post/[id]" || top.Pages[0].ViewCounts != (analogdb.ViewCounts{PageViews: 2, Visitors: 2}) {
			t.Errorf("unexpected pages %+v", top.Pages)
		}
		if len(analytics.Posts) != 1 || analytics.Posts[0].PostID != 12 || analytics.Posts[0].PageViews != 2 {
			t.Errorf("want post 12 with 2 views, got %+v", analytics.Posts)
		}
		if len(top.Referrers) != 1 || top.Referrers[0].Name != "news.ycombinator.com" {
			t.Errorf("want only the external referrer, got %+v", top.Referrers)
		}
		if len(top.Sources) != 1 || top.Sources[0] != (analogdb.AnalyticsCount{Name: "reddit", ViewCounts: one}) {
			t.Errorf("unexpected sources %+v", top.Sources)
		}
		if len(top.Campaigns) != 1 || top.Campaigns[0] != (analogdb.AnalyticsCount{Name: "launch", ViewCounts: one}) {
			t.Errorf("unexpected campaigns %+v", top.Campaigns)
		}
		if len(top.Devices) != 2 || top.Devices[0].Name != "desktop" || top.Devices[0].PageViews != 3 {
			t.Errorf("unexpected devices %+v", top.Devices)
		}
		if len(top.Browsers) != 2 || top.Browsers[0].Name != "Firefox" {
			t.Errorf("unexpected browsers %+v", top.Browsers)
		}

		if len(analytics.Vitals) != 1 {
			t.Fatalf("want vitals for one route, got %+v", analytics.Vitals)
		}
		vital := analytics.Vitals[0]
		if vital.Route != "/post/[id]" || vital.Samples != 3 || vital.INP != nil {
			t.Errorf("unexpected vital %+v", vital)
		}
		if vital.LCP == nil || *vital.LCP < 2000 || *vital.LCP > 3000 {
			t.Errorf("want human LCP p75 within samples, got %v", vital.LCP)
		}
		if vital.CLS == nil || *vital.CLS != 0.05 {
			t.Errorf("want CLS 0.05, got %v", vital.CLS)
		}
	})

	t.Run("week", func(t *testing.T) {
		analytics, err := db.Analytics(ctx, analogdb.TrafficWeek)
		if err != nil {
			t.Fatal(err)
		}
		current := analytics.Summary.Current
		if analytics.Bucket != "hour" || current.PageViews != 6 {
			t.Errorf("want 6 page views in hour buckets, got %d in %s", current.PageViews, analytics.Bucket)
		}
		mustFilled(t, analyticsTimes(analytics.Series), time.Hour, 7*24)
		if want := wantVisitorDays(now, 7*day, 0); current.Visitors != want || want < 4 {
			t.Errorf("want %d visitor days, got %d", want, current.Visitors)
		}
		if analytics.Summary.Previous.PageViews != 1 {
			t.Errorf("want one page view in the previous week, got %d", analytics.Summary.Previous.PageViews)
		}
	})

	t.Run("shape", func(t *testing.T) {
		analytics, err := db.Analytics(ctx, analogdb.TrafficDay)
		if err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(analytics)
		if err != nil {
			t.Fatal(err)
		}
		var body map[string]json.RawMessage
		if err := json.Unmarshal(data, &body); err != nil {
			t.Fatal(err)
		}
		var top map[string]json.RawMessage
		if err := json.Unmarshal(body["top"], &top); err != nil {
			t.Fatal(err)
		}
		if got := sortedKeys(body); !slices.Equal(got, []string{"bucket", "posts", "range", "series", "summary", "top", "vitals"}) {
			t.Errorf("unexpected top level keys %v", got)
		}
		if got := sortedKeys(top); !slices.Equal(got, []string{"browsers", "campaigns", "devices", "pages", "referrers", "sources"}) {
			t.Errorf("unexpected top keys %v", got)
		}
		for key, raw := range top {
			if string(raw) == "null" {
				t.Errorf("want an empty list for %s, got null", key)
			}
		}
	})

	t.Run("bad range", func(t *testing.T) {
		if _, err := db.Analytics(ctx, "1y"); analogdb.ErrorCode(err) != analogdb.ERRBADREQUEST {
			t.Errorf("want bad request, got %v", err)
		}
	})
}

func analyticsTimes(series []analogdb.AnalyticsBucket) []time.Time {
	times := make([]time.Time, len(series))
	for i, b := range series {
		times[i] = b.Time
	}
	return times
}

func sortedKeys(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
