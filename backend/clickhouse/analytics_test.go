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
	search   string
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
	{ago: time.Hour, id: "00000000-0000-0000-0000-000000000013", name: "search", visitor: "v1", path: "/search", route: "/search", search: "s1", props: `{"q_norm":"portra 400","mode":"text","outcome":"ok","result_count":40,"page":0}`},
	{ago: time.Hour, id: "00000000-0000-0000-0000-000000000013", name: "search", visitor: "v1", path: "/search", route: "/search", search: "s1", props: `{"q_norm":"portra 400","mode":"text","outcome":"ok","result_count":40,"page":0}`},
	{ago: time.Hour, id: "00000000-0000-0000-0000-000000000014", name: "search", visitor: "v1", path: "/search", route: "/search", search: "s1", props: `{"q_norm":"portra 400","mode":"text","outcome":"ok","result_count":12,"page":1}`},
	{ago: 2 * time.Hour, id: "00000000-0000-0000-0000-000000000015", name: "search", visitor: "v2", path: "/search", route: "/search", search: "s2", props: `{"q_norm":"portra 400","mode":"text","outcome":"ok","result_count":40,"page":0}`},
	{ago: 2 * time.Hour, id: "00000000-0000-0000-0000-000000000016", name: "search", visitor: "v2", path: "/search", route: "/search", search: "s3", props: `{"q_norm":"mamiya 6","mode":"text","outcome":"empty","result_count":0,"page":0}`},
	{ago: 2 * time.Hour, id: "00000000-0000-0000-0000-000000000017", name: "search", visitor: "v3", path: "/search", route: "/search", search: "s4", bot: true, props: `{"q_norm":"hp5","mode":"text","outcome":"empty","result_count":0,"page":0}`},
	{ago: 2 * time.Hour, id: "00000000-0000-0000-0000-000000000018", name: "search", visitor: "v2", path: "/search", route: "/search", search: "s5", props: `{"q_norm":"","mode":"image","outcome":"ok","result_count":40,"page":0}`},
	{ago: 30 * time.Hour, id: "00000000-0000-0000-0000-000000000019", name: "search", visitor: "v1", path: "/search", route: "/search", search: "s6", props: `{"q_norm":"tri-x","mode":"text","outcome":"empty","result_count":0,"page":0}`},
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

	batch, err := db.conn.PrepareBatch(ctx, `INSERT INTO ui_events (event_id, event_name, visitor_id, client_ts, received_ts, path, route, referrer_host, utm_source, utm_campaign, device_type, browser, country, is_bot, post_id, search_id, props)`)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range uiRows {
		ts := now.Add(-row.ago)
		if err := batch.Append(row.id, row.name, row.visitor, ts, ts, row.path, row.route, row.referrer, row.source, row.campaign, row.device, row.browser, row.country, row.bot, row.postID, row.search, row.props); err != nil {
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

		api := analytics.API
		wantCurrent := analogdb.TrafficTotals{Requests: 12, Web: 2, Scraper: 1, Direct: 9, Bots: 3, Status4: 3, Status5: 1, PageViews: 4}
		current := api.Summary.Current
		if current.P95Ms <= 0 {
			t.Errorf("want a current api p95, got %v", current.P95Ms)
		}
		current.P95Ms = 0
		if current != wantCurrent {
			t.Errorf("want api current %+v, got %+v", wantCurrent, current)
		}
		wantPrevious := analogdb.TrafficTotals{Requests: 2, Web: 1, Direct: 1, Status5: 1, P95Ms: api.Summary.Previous.P95Ms, PageViews: 1}
		if api.Summary.Previous != wantPrevious {
			t.Errorf("want api previous %+v, got %+v", wantPrevious, api.Summary.Previous)
		}
		wantCallers := []analogdb.TrafficCaller{
			{Name: "SemrushBot", Kind: "bot", Requests: 3, IPs: 3},
			{Name: "curl", Kind: "tool", Requests: 3, IPs: 2},
			{Name: safari, Kind: "browser", Requests: 1, IPs: 1},
			{Name: "", Kind: "empty", Requests: 1, IPs: 1},
			{Name: "python-requests", Kind: "tool", Requests: 1, IPs: 1},
		}
		if !slices.Equal(api.Callers, wantCallers) {
			t.Errorf("want api callers %+v, got %+v", wantCallers, api.Callers)
		}
		if len(api.Errors) != 1 || api.Errors[0].RequestID != "r4" || api.Errors[0].Status != 500 {
			t.Errorf("unexpected api errors %+v", api.Errors)
		}

		searches := analytics.Searches
		if want := (analogdb.AnalyticsSearchTotals{Searches: 3, Empty: 1}); searches.Current != want {
			t.Errorf("want current searches %+v, got %+v", want, searches.Current)
		}
		if want := (analogdb.AnalyticsSearchTotals{Searches: 1, Empty: 1}); searches.Previous != want {
			t.Errorf("want previous searches %+v, got %+v", want, searches.Previous)
		}
		wantTop := []analogdb.AnalyticsSearch{{Query: "portra 400", Searches: 2}, {Query: "mamiya 6", Searches: 1}}
		if !slices.Equal(searches.Top, wantTop) {
			t.Errorf("want top searches %+v, got %+v", wantTop, searches.Top)
		}
		wantEmpty := []analogdb.AnalyticsSearch{{Query: "mamiya 6", Searches: 1}}
		if !slices.Equal(searches.Empty, wantEmpty) {
			t.Errorf("want empty searches %+v, got %+v", wantEmpty, searches.Empty)
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
		if analytics.API.Summary.Current.Requests != 15 || analytics.API.Summary.Previous.Requests != 0 {
			t.Errorf("want 15 api requests and none before, got %+v", analytics.API.Summary)
		}
		if want := (analogdb.AnalyticsSearchTotals{Searches: 4, Empty: 2}); analytics.Searches.Current != want {
			t.Errorf("want week searches %+v, got %+v", want, analytics.Searches.Current)
		}
	})

	t.Run("month", func(t *testing.T) {
		analytics, err := db.Analytics(ctx, analogdb.TrafficMonth)
		if err != nil {
			t.Fatal(err)
		}
		if analytics.Bucket != "day" {
			t.Errorf("want day buckets, got %s", analytics.Bucket)
		}
		mustFilled(t, analyticsTimes(analytics.Series), 24*time.Hour, 30)
		if analytics.API.Summary.Current.Requests != 15 || analytics.API.Summary.Previous.Requests != 1 {
			t.Errorf("want 15 api requests and one before, got %+v", analytics.API.Summary)
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
		var top, api, searches map[string]json.RawMessage
		for key, dst := range map[string]*map[string]json.RawMessage{"top": &top, "api": &api, "searches": &searches} {
			if err := json.Unmarshal(body[key], dst); err != nil {
				t.Fatal(err)
			}
		}
		if got := sortedKeys(body); !slices.Equal(got, []string{"api", "bucket", "posts", "range", "searches", "series", "summary", "top", "vitals"}) {
			t.Errorf("unexpected top level keys %v", got)
		}
		if got := sortedKeys(api); !slices.Equal(got, []string{"callers", "errors", "summary"}) {
			t.Errorf("unexpected api keys %v", got)
		}
		if got := sortedKeys(searches); !slices.Equal(got, []string{"current", "empty", "previous", "top"}) {
			t.Errorf("unexpected searches keys %v", got)
		}
		for _, key := range []string{"callers", "errors"} {
			if string(api[key]) == "null" {
				t.Errorf("want an empty list for api %s, got null", key)
			}
		}
		for _, key := range []string{"top", "empty"} {
			if string(searches[key]) == "null" {
				t.Errorf("want an empty list for searches %s, got null", key)
			}
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
