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
	Direct  int64     `json:"direct"`
	Status4 int64     `json:"status_4xx"`
	Status5 int64     `json:"status_5xx"`
}

// TrafficTotals counts API requests, direct is neither web nor scraper
type TrafficTotals struct {
	Requests  int64   `json:"requests"`
	Web       int64   `json:"web"`
	Scraper   int64   `json:"scraper"`
	Direct    int64   `json:"direct"`
	Bots      int64   `json:"bots"`
	Status4   int64   `json:"status_4xx"`
	Status5   int64   `json:"status_5xx"`
	P95Ms     float64 `json:"p95_ms"`
	PageViews int64   `json:"page_views"`
}

type TrafficSummary struct {
	Current  TrafficTotals `json:"current"`
	Previous TrafficTotals `json:"previous"`
}

// TrafficCaller groups direct requests by caller, kind is bot, tool, browser or empty
type TrafficCaller struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Requests int64  `json:"requests"`
	IPs      int64  `json:"ips"`
}

type TrafficRoute struct {
	Route    string  `json:"route"`
	Requests int64   `json:"requests"`
	TotalMs  int64   `json:"total_ms"`
	P50Ms    float64 `json:"p50_ms"`
	P95Ms    float64 `json:"p95_ms"`
	Status4  int64   `json:"status_4xx"`
	Status5  int64   `json:"status_5xx"`
}

type TrafficStatus struct {
	Status   int32  `json:"status"`
	Route    string `json:"route"`
	Requests int64  `json:"requests"`
}

type TrafficError struct {
	Time      time.Time `json:"time"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Status    int32     `json:"status"`
	RequestID string    `json:"request_id"`
}

type TrafficErrors struct {
	ByStatus []TrafficStatus `json:"by_status"`
	Recent   []TrafficError  `json:"recent"`
}

// Traffic is API load from request logs
type Traffic struct {
	Range   TrafficRange    `json:"range"`
	Bucket  string          `json:"bucket"`
	Summary TrafficSummary  `json:"summary"`
	Series  []TrafficBucket `json:"series"`
	Callers []TrafficCaller `json:"callers"`
	Routes  []TrafficRoute  `json:"routes"`
	Errors  TrafficErrors   `json:"errors"`
}

// ViewCounts holds page views and visitor-days, visitor ids rotate daily
type ViewCounts struct {
	PageViews int64 `json:"page_views"`
	Visitors  int64 `json:"visitors"`
}

type AnalyticsTotals struct {
	ViewCounts
	BotViews int64 `json:"bot_views"`
}

type AnalyticsSummary struct {
	Current  AnalyticsTotals `json:"current"`
	Previous AnalyticsTotals `json:"previous"`
	Live     ViewCounts      `json:"live"`
}

type AnalyticsBucket struct {
	Time time.Time `json:"time"`
	ViewCounts
}

type AnalyticsCount struct {
	Name string `json:"name"`
	ViewCounts
}

type AnalyticsTop struct {
	Pages     []AnalyticsCount `json:"pages"`
	Referrers []AnalyticsCount `json:"referrers"`
	Sources   []AnalyticsCount `json:"sources"`
	Campaigns []AnalyticsCount `json:"campaigns"`
	Devices   []AnalyticsCount `json:"devices"`
	Browsers  []AnalyticsCount `json:"browsers"`
}

type AnalyticsPost struct {
	PostID int64  `json:"post_id"`
	Title  string `json:"title"`
	LowURL string `json:"low_url"`
	ViewCounts
}

// AnalyticsVital holds p75 web vitals for a route, nil when a metric has no samples
type AnalyticsVital struct {
	Route   string   `json:"route"`
	LCP     *float64 `json:"lcp"`
	INP     *float64 `json:"inp"`
	CLS     *float64 `json:"cls"`
	Samples int64    `json:"samples"`
}

// Analytics is product traffic from UI events
type Analytics struct {
	Range   TrafficRange      `json:"range"`
	Bucket  string            `json:"bucket"`
	Summary AnalyticsSummary  `json:"summary"`
	Series  []AnalyticsBucket `json:"series"`
	Top     AnalyticsTop      `json:"top"`
	Posts   []AnalyticsPost   `json:"posts"`
	Vitals  []AnalyticsVital  `json:"vitals"`
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
