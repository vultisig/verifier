package metrics

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// Config holds metrics server configuration
type Config struct {
	Enabled bool   `mapstructure:"enabled" json:"enabled,omitempty"`
	Host    string `mapstructure:"host" json:"host,omitempty"`
	Port    int    `mapstructure:"port" json:"port,omitempty"`
	Token   string `mapstructure:"token" json:"token,omitempty"`
}

// DefaultConfig returns default metrics configuration
func DefaultConfig() Config {
	return Config{
		Enabled: true,
		Host:    "0.0.0.0",
		Port:    8088, // Use unprivileged port
	}
}

// Server represents the metrics HTTP server
type Server struct {
	server *http.Server
	logger *logrus.Logger
}

// bearerAuthMiddleware wraps an http.Handler with Bearer token authentication
func bearerAuthMiddleware(handler http.Handler, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		providedToken := strings.TrimPrefix(authHeader, "Bearer ")
		if providedToken != token {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	})
}

// NewServer creates a new metrics server with a custom registry
func NewServer(cfg Config, logger *logrus.Logger, registry *prometheus.Registry) *Server {
	mux := http.NewServeMux()

	var metricsHandler http.Handler
	if registry != nil {
		metricsHandler = promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	} else {
		metricsHandler = promhttp.Handler()
	}

	if cfg.Token != "" {
		metricsHandler = bearerAuthMiddleware(metricsHandler, cfg.Token)
		logger.Info("Metrics endpoint authentication enabled")
	}

	mux.Handle("/metrics", metricsHandler)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	if cfg.Host == "" {
		addr = fmt.Sprintf(":%d", cfg.Port)
	}
	if cfg.Port <= 0 {
		if cfg.Host == "" {
			addr = ":8088"
		} else {
			addr = fmt.Sprintf("%s:8088", cfg.Host)
		}
	}

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	return &Server{
		server: server,
		logger: logger,
	}
}

// Start starts the metrics server in a goroutine
func (s *Server) Start() {
	go func() {
		s.logger.Infof("Starting metrics server on %s", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Errorf("Metrics server failed to start: %v", err)
		}
	}()
}

// Stop gracefully shuts down the metrics server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Shutting down metrics server")
	return s.server.Shutdown(ctx)
}

// StartMetricsServer is a convenience function to start the metrics server with services
func StartMetricsServer(cfg Config, services []string, logger *logrus.Logger) *Server {
	if !cfg.Enabled {
		logger.Info("Metrics server disabled")
		return nil
	}

	registry := prometheus.NewRegistry()
	RegisterMetrics(services, registry, logger)

	server := NewServer(cfg, logger, registry)
	server.Start()
	return server
}
