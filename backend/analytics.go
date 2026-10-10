package analogdb

import (
	"context"
	"time"
)

const (
	WebUserAgentPrefix     = "analogdb-web/"
	ScraperUserAgentPrefix = "analogdb-scraper/"
)

type TrafficRange string

const (
	TrafficDay   TrafficRange = "24h"
	TrafficWeek  TrafficRange = "7d"
	TrafficMonth TrafficRange = "30d"
)

func (r TrafficRange) Duration() (time.Duration, bool) {
	switch r {
	case TrafficDay:
		return 24 * time.Hour, true
	case TrafficWeek:
		return 7 * 24 * time.Hour, true
	case TrafficMonth:
		return 30 * 24 * time.Hour, true
	}
	return 0, false
}

type TrafficBucket struct {
	Time    time.Time `json:"time"`
	Web     int64     `json:"web"`
	Scraper int64     `json:"scraper"`
	Other   int64     `json:"other"`
	Status4 int64     `json:"status_4xx"`
	Status5 int64     `json:"status_5xx"`
}

type TrafficTotals struct {
	Requests  int64 `json:"requests"`
	UniqueIPs int64 `json:"unique_ips"`
	Status2   int64 `json:"status_2xx"`
	Status3   int64 `json:"status_3xx"`
	Status4   int64 `json:"status_4xx"`
	Status5   int64 `json:"status_5xx"`
}

type TrafficRoute struct {
	Route    string  `json:"route"`
	Requests int64   `json:"requests"`
	P50Ms    float64 `json:"p50_ms"`
	P95Ms    float64 `json:"p95_ms"`
	Errors   int64   `json:"errors"`
}

type TrafficPost struct {
	PostID   int64 `json:"post_id"`
	Requests int64 `json:"requests"`
}

type TrafficLegacy struct {
	Client   string `json:"client"`
	Requests int64  `json:"requests"`
	Legacy   int64  `json:"legacy"`
}

type TrafficCount struct {
	Name     string `json:"name"`
	Client   string `json:"client,omitempty"`
	Requests int64  `json:"requests"`
}

type TrafficError struct {
	Time      time.Time `json:"time"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Status    int32     `json:"status"`
	RequestID string    `json:"request_id"`
}

type Traffic struct {
	Range      TrafficRange    `json:"range"`
	Bucket     string          `json:"bucket"`
	Series     []TrafficBucket `json:"series"`
	Totals     TrafficTotals   `json:"totals"`
	Routes     []TrafficRoute  `json:"routes"`
	Posts      []TrafficPost   `json:"posts"`
	Legacy     []TrafficLegacy `json:"legacy"`
	Params     []TrafficCount  `json:"params"`
	UserAgents []TrafficCount  `json:"user_agents"`
	IPs        []TrafficCount  `json:"ips"`
	Errors     []TrafficError  `json:"errors"`
}

type AnalyticsTotals struct {
	PageViews       int64   `json:"page_views"`
	VisitorDays     int64   `json:"visitor_days"`
	ViewsPerVisitor float64 `json:"views_per_visitor"`
	BotShare        float64 `json:"bot_share"`
}

type AnalyticsBucket struct {
	Time      time.Time `json:"time"`
	PageViews int64     `json:"page_views"`
	Visitors  int64     `json:"visitors"`
}

type AnalyticsLive struct {
	PageViews int64 `json:"page_views"`
	Visitors  int64 `json:"visitors"`
}

type AnalyticsPost struct {
	PostID    int64  `json:"post_id"`
	Title     string `json:"title"`
	LowURL    string `json:"low_url"`
	PageViews int64  `json:"page_views"`
	Visitors  int64  `json:"visitors"`
}

type AnalyticsCount struct {
	Name      string `json:"name"`
	PageViews int64  `json:"page_views"`
	Visitors  int64  `json:"visitors"`
}

type AnalyticsCampaign struct {
	Source    string `json:"source"`
	Campaign  string `json:"campaign"`
	PageViews int64  `json:"page_views"`
	Visitors  int64  `json:"visitors"`
}

// AnalyticsVital holds p75 web vitals for a route, nil when a metric has no samples
type AnalyticsVital struct {
	Route   string   `json:"route"`
	LCP     *float64 `json:"lcp"`
	INP     *float64 `json:"inp"`
	CLS     *float64 `json:"cls"`
	Samples int64    `json:"samples"`
}

// Analytics is product traffic from UI events, Available is false until the table exists
type Analytics struct {
	Range     TrafficRange        `json:"range"`
	Bucket    string              `json:"bucket"`
	Available bool                `json:"available"`
	Totals    AnalyticsTotals     `json:"totals"`
	Previous  AnalyticsTotals     `json:"previous"`
	Series    []AnalyticsBucket   `json:"series"`
	Live      AnalyticsLive       `json:"live"`
	Pages     []AnalyticsCount    `json:"pages"`
	Posts     []AnalyticsPost     `json:"posts"`
	Referrers []AnalyticsCount    `json:"referrers"`
	Campaigns []AnalyticsCampaign `json:"campaigns"`
	Devices   []AnalyticsCount    `json:"devices"`
	Browsers  []AnalyticsCount    `json:"browsers"`
	Countries []AnalyticsCount    `json:"countries"`
	Vitals    []AnalyticsVital    `json:"vitals"`
}

type AuditFilter struct {
	Limit int
	// Before is a start time in unix milliseconds
	Before *int64
}

type AuditEntry struct {
	Time      time.Time `json:"time"`
	StartMs   int64     `json:"start_ms"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Status    int32     `json:"status"`
	RemoteIP  string    `json:"remote_ip"`
	UserAgent string    `json:"user_agent"`
	RequestID string    `json:"request_id"`
	Client    string    `json:"client"`
}

type AnalyticsService interface {
	Traffic(ctx context.Context, r TrafficRange) (*Traffic, error)
	Analytics(ctx context.Context, r TrafficRange) (*Analytics, error)
	Audit(ctx context.Context, filter *AuditFilter) ([]*AuditEntry, error)
	Readyz(ctx context.Context) error
}
