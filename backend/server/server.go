package server

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/config"
	_ "github.com/evanofslack/analogdb/docs"
	"github.com/evanofslack/analogdb/logger"
	"github.com/evanofslack/analogdb/metrics"
	"github.com/go-chi/chi/v5"
)

// @title AnalogDB API
// @version 1.0
// @description API for analogdb film database
// @host api.analogdb.com
// @BasePath /v1
// @securityDefinitions.basic BasicAuth

const (
	shutdownTimeout   = 5 * time.Second
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 120 * time.Second
)

type Server struct {
	server         *http.Server
	router         *chi.Mux
	healthy        atomic.Bool
	logger         *logger.Logger
	metrics        *metrics.Metrics
	config         *config.Config
	stats          *httpStats
	hostname       string
	trustedProxies []*net.IPNet
	startedAt      time.Time

	PostService       analogdb.PostService
	FilmService       analogdb.FilmService
	CameraService     analogdb.CameraService
	ReadyService      analogdb.ReadyService
	AuthorService     analogdb.AuthorService
	ScrapeService     analogdb.ScrapeService
	KeywordService    analogdb.KeywordService
	SimilarityService analogdb.SimilarityService
	EventService      analogdb.EventService
	AdminService      analogdb.AdminService
	AnalyticsService  analogdb.AnalyticsService
	VectorCounter     analogdb.VectorCounter

	CacheReadyService  analogdb.ReadyService
	VectorReadyService analogdb.ReadyService
}

func New(port string, logger *logger.Logger, metrics *metrics.Metrics, config *config.Config) *Server {
	s := &Server{
		server: &http.Server{
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		},
		router:    chi.NewRouter(),
		logger:    logger,
		metrics:   metrics,
		config:    config,
		hostname:  "localhost",
		startedAt: time.Now(),
	}

	if s.config.Auth.Username == "" && s.config.Auth.Password == "" {
		s.logger.Error("Config auth username and password not set!")
	}

	if s.config.Auth.RateLimitUsername == "" && s.config.Auth.RateLimitPassword == "" {
		s.logger.Error("Config ratelimit auth username and password not set!")
	}

	hostname, err := os.Hostname()
	if err != nil {
		s.logger.Warn("Fail get hostname", "error", err)
	} else {
		s.hostname = hostname
	}

	trusted, invalid := parseTrustedProxies(s.config.HTTP.TrustedProxies)
	if len(invalid) > 0 {
		s.logger.Error("Ignoring invalid trusted proxies", "invalid", invalid)
	}
	s.trustedProxies = trusted

	s.server.Handler = s.router
	s.server.Addr = ":" + port

	s.stats = newHttpStats()
	if err := s.stats.register(s.metrics.Registry); err != nil {
		s.logger.Error("Fail register http metrics", "error", err)
	}

	s.mountMiddleware()

	// Mount only at base root
	s.mountStaticHandlers()
	s.mountStatusHandlers()
	s.mountDebugHandlers()
	s.mountSwaggerHandlers()

	// Mount collection resources
	s.mountResourceHandlers()

	s.healthy.Store(true)
	return s
}

func (s *Server) Run() error {
	ln, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return err
	}
	s.logger.Info("Serving http server", "address", ln.Addr().String())
	go func() {
		if err := s.server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("Http server stopped", "error", err)
		}
	}()
	return nil
}

// Mount all resources at both base and v1 paths.
// Once deprecation date, can drop mount at base and just keep v1.
func (s *Server) mountResourceHandlers() {
	v1 := chi.NewRouter()
	s.mountPostHandlers(v1)
	s.mountFilmHandlers(v1)
	s.mountCameraHandlers(v1)
	s.mountAuthorHandlers(v1)
	s.mountSimilarityHandlers(v1)
	s.mountScrapeHandlers(v1)
	s.mountKeywordHandlers(v1)
	s.mountAdminHandlers(v1)
	s.router.Mount("/v1", v1)

	// Mount legacy routes with deprecation
	s.router.Group(func(r chi.Router) {
		r.Use(s.deprecationMiddleware)
		s.mountPostHandlers(r)
		s.mountFilmHandlers(r)
		s.mountCameraHandlers(r)
		s.mountAuthorHandlers(r)
		s.mountSimilarityHandlers(r)
		s.mountScrapeHandlers(r)
		s.mountKeywordHandlers(r)
	})
}

func (s *Server) Close() error {
	s.logger.Debug("Starting http server close")
	defer s.logger.Info("Closed http server")

	s.healthy.Store(false)
	if delay := s.config.HTTP.ShutdownDrainDelay; delay > 0 {
		s.logger.Info("Draining http server before shutdown", "delay", delay.String())
		time.Sleep(delay)
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return s.server.Shutdown(ctx)
}

func encodeResponse(w http.ResponseWriter, r *http.Request, status int, v any) error {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return err
	}
	return nil
}
