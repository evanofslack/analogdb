package server

import (
	"context"
	"net/http"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/go-chi/chi/v5"
	"golang.org/x/sync/errgroup"
)

// Admin routes are internal: no swag annotations, so they stay out of the
// OpenAPI spec and the generated clients.

const (
	adminPath           = "/admin"
	defaultMissingLimit = 40
	maxMissingLimit     = 100
	defaultAuditLimit   = 50
	maxAuditLimit       = 200
)

type adminApp struct {
	Version  string `json:"version"`
	Env      string `json:"env"`
	Hostname string `json:"hostname"`
	Uptime   int64  `json:"uptime_seconds"`
}

type adminVectors struct {
	Objects *int `json:"objects"`
	Posts   int  `json:"posts"`
}

type adminOverviewResponse struct {
	Status map[string]string `json:"status"`
	App    adminApp          `json:"app"`
	*analogdb.AdminStats
	Vectors adminVectors `json:"vectors"`
}

type adminMissingResponse struct {
	Posts        []*analogdb.AdminPost `json:"posts"`
	NextBeforeID *int                  `json:"next_before_id"`
}

type adminAuditResponse struct {
	Entries    []*analogdb.AuditEntry `json:"entries"`
	NextBefore *int64                 `json:"next_before"`
}

func (s *Server) mountAdminHandlers(r chi.Router) {
	r.Route(adminPath, func(r chi.Router) {
		r.Use(noStore)
		r.Group(func(r chi.Router) {
			r.Use(s.require())
			r.Get("/overview", s.getAdminOverview)
			r.Get("/quality", s.getAdminQuality)
			r.Get("/posts/missing", s.getAdminMissingPosts)
			r.Get("/traffic", s.getAdminTraffic)
			r.Get("/analytics", s.getAdminAnalytics)
			r.Get("/audit", s.getAdminAudit)
			s.mountAdminReportHandlers(r)
		})
		r.Group(func(r chi.Router) {
			r.Use(s.require(roleScraper))
			s.mountExtractionHandlers(r)
			r.Get("/removed/permalinks", s.getRemovedPermalinks)
		})
	})
}

func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) getAdminOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var status map[string]string
	var stats *analogdb.AdminStats
	var vectors *int

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		checks := map[string]analogdb.ReadyService{
			"postgres":   s.ReadyService,
			"redis":      s.CacheReadyService,
			"weaviate":   s.VectorReadyService,
			"clickhouse": s.AnalyticsService,
		}
		status = s.checkDependencies(gctx, checks)
		return nil
	})
	g.Go(func() error {
		var err error
		stats, err = s.AdminService.Stats(gctx)
		return err
	})
	g.Go(func() error {
		if s.VectorCounter == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(gctx, readyCheckTimeout*5)
		defer cancel()
		count, err := s.VectorCounter.CountObjects(ctx)
		if err != nil {
			s.logger.WarnContext(ctx, "Fail count vector objects", "error", err)
			return nil
		}
		vectors = &count
		return nil
	})
	if err := g.Wait(); err != nil {
		s.writeError(w, r, err)
		return
	}

	response := adminOverviewResponse{
		Status: status,
		App: adminApp{
			Version:  s.config.App.Version,
			Env:      s.config.App.Env,
			Hostname: s.hostname,
			Uptime:   int64(time.Since(s.startedAt).Seconds()),
		},
		AdminStats: stats,
		Vectors:    adminVectors{Objects: vectors, Posts: stats.Counts.Posts},
	}
	if err := encodeResponse(w, r, http.StatusOK, response); err != nil {
		s.writeError(w, r, err)
	}
}

func (s *Server) getAdminQuality(w http.ResponseWriter, r *http.Request) {
	quality, err := s.AdminService.Quality(r.Context())
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := encodeResponse(w, r, http.StatusOK, quality); err != nil {
		s.writeError(w, r, err)
	}
}

func (s *Server) getAdminMissingPosts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	field := analogdb.MissingField(query.Get("field"))
	if !field.Valid() {
		s.writeError(w, r, badRequest("invalid field %q, want camera, film, description, keywords, colors, caption or vector", field))
		return
	}

	limit := defaultMissingLimit
	if str := query.Get("limit"); str != "" {
		val, err := stringToInt(str)
		if err != nil {
			s.writeError(w, r, err)
			return
		}
		limit = clampLimit(val, defaultMissingLimit, 1, maxMissingLimit)
	}

	filter := &analogdb.MissingPostsFilter{Field: field, Limit: limit}
	if str := query.Get("before_id"); str != "" {
		val, err := stringToInt(str)
		if err != nil {
			s.writeError(w, r, err)
			return
		}
		filter.BeforeID = &val
	}

	if field == analogdb.MissingVector {
		s.getAdminMissingVectors(w, r, filter)
		return
	}

	posts, err := s.AdminService.MissingPosts(r.Context(), filter)
	if err != nil {
		s.writeError(w, r, err)
		return
	}

	response := adminMissingResponse{Posts: posts}
	if len(posts) == limit {
		next := posts[len(posts)-1].ID
		response.NextBeforeID = &next
	}
	if err := encodeResponse(w, r, http.StatusOK, response); err != nil {
		s.writeError(w, r, err)
	}
}

// getAdminMissingVectors pages the cached missing vector ids, newest first
func (s *Server) getAdminMissingVectors(w http.ResponseWriter, r *http.Request, filter *analogdb.MissingPostsFilter) {
	missing, _, err := s.missingVectors(r.Context())
	if err != nil {
		s.writeError(w, r, err)
		return
	}

	ids := make([]int, 0, filter.Limit)
	for _, id := range missing {
		if filter.BeforeID != nil && id >= *filter.BeforeID {
			continue
		}
		ids = append(ids, id)
		if len(ids) == filter.Limit {
			break
		}
	}

	posts := []*analogdb.AdminPost{}
	if len(ids) > 0 {
		posts, err = s.AdminService.PostsByIDs(r.Context(), ids)
		if err != nil {
			s.writeError(w, r, err)
			return
		}
	}

	response := adminMissingResponse{Posts: posts}
	if len(ids) == filter.Limit {
		next := ids[len(ids)-1]
		response.NextBeforeID = &next
	}
	if err := encodeResponse(w, r, http.StatusOK, response); err != nil {
		s.writeError(w, r, err)
	}
}

func (s *Server) analyticsUnavailable(w http.ResponseWriter, r *http.Request) bool {
	if s.AnalyticsService != nil {
		return false
	}
	s.writeError(w, r, &analogdb.Error{Code: analogdb.ERRUNAVAILABLE, Message: "clickhouse is not enabled"})
	return true
}

func (s *Server) getAdminTraffic(w http.ResponseWriter, r *http.Request) {
	rng := analogdb.TrafficWeek
	if str := r.URL.Query().Get("range"); str != "" {
		rng = analogdb.TrafficRange(str)
	}
	if _, ok := rng.Duration(); !ok {
		s.writeError(w, r, badRequest("invalid range %q, want 24h, 7d or 30d", rng))
		return
	}
	if s.analyticsUnavailable(w, r) {
		return
	}

	traffic, err := s.AnalyticsService.Traffic(r.Context(), rng)
	if err != nil {
		s.writeError(w, r, analyticsError(err))
		return
	}
	if err := encodeResponse(w, r, http.StatusOK, traffic); err != nil {
		s.writeError(w, r, err)
	}
}

func (s *Server) getAdminAnalytics(w http.ResponseWriter, r *http.Request) {
	rng := analogdb.TrafficWeek
	if str := r.URL.Query().Get("range"); str != "" {
		rng = analogdb.TrafficRange(str)
	}
	if _, ok := rng.Duration(); !ok {
		s.writeError(w, r, badRequest("invalid range %q, want 24h, 7d or 30d", rng))
		return
	}
	if s.analyticsUnavailable(w, r) {
		return
	}

	analytics, err := s.AnalyticsService.Analytics(r.Context(), rng)
	if err != nil {
		s.writeError(w, r, analyticsError(err))
		return
	}
	s.addAnalyticsPosts(r.Context(), analytics.Posts)
	if err := encodeResponse(w, r, http.StatusOK, analytics); err != nil {
		s.writeError(w, r, err)
	}
}

// addAnalyticsPosts fills in titles and thumbnails, missing posts keep only their id
func (s *Server) addAnalyticsPosts(ctx context.Context, posts []analogdb.AnalyticsPost) {
	if len(posts) == 0 || s.AdminService == nil {
		return
	}
	ids := make([]int, 0, len(posts))
	for _, p := range posts {
		ids = append(ids, int(p.PostID))
	}
	found, err := s.AdminService.PostsByIDs(ctx, ids)
	if err != nil {
		s.logger.WarnContext(ctx, "Fail look up analytics posts", "error", err)
		return
	}
	byID := make(map[int]*analogdb.AdminPost, len(found))
	for _, p := range found {
		byID[p.ID] = p
	}
	for i := range posts {
		if p, ok := byID[int(posts[i].PostID)]; ok {
			posts[i].Title = p.Title
			posts[i].LowURL = p.LowURL
		}
	}
}

func (s *Server) getAdminAudit(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	limit := defaultAuditLimit
	if str := query.Get("limit"); str != "" {
		val, err := stringToInt(str)
		if err != nil {
			s.writeError(w, r, err)
			return
		}
		limit = clampLimit(val, defaultAuditLimit, 1, maxAuditLimit)
	}

	filter := &analogdb.AuditFilter{Limit: limit}
	if str := query.Get("before"); str != "" {
		val, err := stringToInt64(str)
		if err != nil {
			s.writeError(w, r, err)
			return
		}
		filter.Before = &val
	}
	if s.analyticsUnavailable(w, r) {
		return
	}

	entries, err := s.AnalyticsService.Audit(r.Context(), filter)
	if err != nil {
		s.writeError(w, r, analyticsError(err))
		return
	}

	response := adminAuditResponse{Entries: entries}
	if len(entries) == limit {
		next := entries[len(entries)-1].StartMs
		response.NextBefore = &next
	}
	if err := encodeResponse(w, r, http.StatusOK, response); err != nil {
		s.writeError(w, r, err)
	}
}

// analyticsError reports ClickHouse failures as unavailable so the admin page
// can tell them apart from bugs
func analyticsError(err error) error {
	if code := analogdb.ErrorCode(err); code != analogdb.ERRINTERNAL {
		return err
	}
	return &analogdb.Error{Code: analogdb.ERRUNAVAILABLE, Message: "clickhouse query failed"}
}
