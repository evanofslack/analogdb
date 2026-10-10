package clickhouse

import (
	"context"
	"os"
	"path/filepath"
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
	{ago: 3 * time.Hour, id: "00000000-0000-0000-0000-000000000003", name: "page_view", visitor: "v2", path: "/post/12", route: "/post/[id]", referrer: "www.analogdb.com", device: "mobile", browser: "Safari"},
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
		analytics, err := db.Analytics(ctx, analogdb.TrafficDay)
		if err != nil {
			t.Fatal(err)
		}
		if analytics.Available || analytics.Series == nil || analytics.Posts == nil {
			t.Errorf("want unavailable with empty lists, got %+v", analytics)
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
		if !analytics.Available || analytics.Bucket != "hour" {
			t.Fatalf("want available hour buckets, got %v %s", analytics.Available, analytics.Bucket)
		}

		totals := analytics.Totals
		if totals.PageViews != 4 {
			t.Errorf("want 4 page views without bots and duplicates, got %d", totals.PageViews)
		}
		if want := wantVisitorDays(now, day, 0); totals.VisitorDays != want {
			t.Errorf("want %d visitor days, got %d", want, totals.VisitorDays)
		}
		if totals.BotShare != 0.2 {
			t.Errorf("want bot share 0.2, got %f", totals.BotShare)
		}
		if analytics.Previous.PageViews != 1 || analytics.Previous.VisitorDays != 1 {
			t.Errorf("want one page view in the previous day, got %+v", analytics.Previous)
		}

		var views int64
		for _, b := range analytics.Series {
			views += b.PageViews
		}
		if views != 4 {
			t.Errorf("want 4 page views in series, got %d", views)
		}
		if analytics.Live != (analogdb.AnalyticsLive{PageViews: 1, Visitors: 1}) {
			t.Errorf("want one live view, got %+v", analytics.Live)
		}

		if len(analytics.Pages) != 3 || analytics.Pages[0].Name != "/post/[id]" || analytics.Pages[0].PageViews != 2 || analytics.Pages[0].Visitors != 2 {
			t.Errorf("unexpected pages %+v", analytics.Pages)
		}
		if len(analytics.Posts) != 1 || analytics.Posts[0].PostID != 12 || analytics.Posts[0].PageViews != 2 {
			t.Errorf("want post 12 by id and by path, got %+v", analytics.Posts)
		}
		if len(analytics.Referrers) != 1 || analytics.Referrers[0].Name != "news.ycombinator.com" {
			t.Errorf("want only the external referrer, got %+v", analytics.Referrers)
		}
		if len(analytics.Campaigns) != 1 || analytics.Campaigns[0] != (analogdb.AnalyticsCampaign{Source: "reddit", Campaign: "launch", PageViews: 1, Visitors: 1}) {
			t.Errorf("unexpected campaigns %+v", analytics.Campaigns)
		}
		if len(analytics.Devices) != 2 || analytics.Devices[0].Name != "desktop" || analytics.Devices[0].PageViews != 3 {
			t.Errorf("unexpected devices %+v", analytics.Devices)
		}
		if len(analytics.Browsers) != 2 || analytics.Browsers[0].Name != "Firefox" {
			t.Errorf("unexpected browsers %+v", analytics.Browsers)
		}
		if len(analytics.Countries) != 1 || analytics.Countries[0].Name != "NL" {
			t.Errorf("unexpected countries %+v", analytics.Countries)
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
		if analytics.Bucket != "day" || analytics.Totals.PageViews != 6 {
			t.Errorf("want 6 page views in day buckets, got %d in %s", analytics.Totals.PageViews, analytics.Bucket)
		}
		if want := wantVisitorDays(now, 7*day, 0); analytics.Totals.VisitorDays != want || want < 4 {
			t.Errorf("want %d visitor days, got %d", want, analytics.Totals.VisitorDays)
		}
		if analytics.Previous.PageViews != 1 {
			t.Errorf("want one page view in the previous week, got %d", analytics.Previous.PageViews)
		}
	})

	t.Run("bad range", func(t *testing.T) {
		if _, err := db.Analytics(ctx, "1y"); analogdb.ErrorCode(err) != analogdb.ERRBADREQUEST {
			t.Errorf("want bad request, got %v", err)
		}
	})
}
