package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

const (
	readTimeout  = time.Second * 30
	writeTimeout = time.Second * 30
	idleTimeout  = time.Second * 120
)

type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

type Server struct {
	logger     *slog.Logger
	server     *http.Server
	port       int
	checkers   map[string]HealthChecker
	router     *mux.Router
	start      time.Time
	appName    string
	appVersion string
	appEnv     string
	appCommit  string
}

// New creates a new HTTP server instance
func New(logger *slog.Logger, port int, appName, appVersion, appEnv string) *Server {
	router := mux.NewRouter()

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	s := &Server{
		logger:     logger,
		server:     server,
		port:       port,
		checkers:   make(map[string]HealthChecker),
		router:     router,
		start:      time.Now(),
		appName:    appName,
		appVersion: appVersion,
		appEnv:     appEnv,
		appCommit:  getCommit(),
	}

	s.setupRoutes()
	s.server.Handler = s.router
	return s
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting server", "addr", s.server.Addr)

	errChan := make(chan error, 1)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("server failed to start: %w", err)
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return s.Shutdown(context.Background())
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server")

	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("Server shutdown", "error", err)
		return fmt.Errorf("server shutdown: %w", err)
	}

	s.logger.Info("Server stopped")
	return nil
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// Apply middleware to all routes
	s.router.Use(s.loggingMiddleware)
	s.router.Use(s.recoveryMiddleware)
	s.router.Use(s.corsMiddleware)

	s.router.HandleFunc("/health", s.healthHandler).Methods("GET")
	s.router.HandleFunc("/ready", s.readinessHandler).Methods("GET")
	s.router.HandleFunc("/info", s.infoHandler).Methods("GET")
}
