package metrics

import (
	"context"
	"fmt"
	"net/http"
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

// NewServer creates a new metrics server with a custom registry
func NewServer(host string, port int, logger *logrus.Logger, registry *prometheus.Registry) *Server {
	mux := http.NewServeMux()

	// Register the Prometheus metrics handler with custom registry
	if registry != nil {
		mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	} else {
		mux.Handle("/metrics", promhttp.Handler())
	}

	// Add a health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	addr := fmt.Sprintf("%s:%d", host, port)
	if host == "" {
		addr = fmt.Sprintf(":%d", port)
	}
	if port <= 0 {
		if host == "" {
			addr = ":8088"
		} else {
			addr = fmt.Sprintf("%s:8088", host)
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

	// Create registry and register metrics for specified services
	registry := prometheus.NewRegistry()
	RegisterMetrics(services, registry, logger)

	server := NewServer(cfg.Host, cfg.Port, logger, registry)
	server.Start()
	return server
}

// StartMetricsServerWithRegistry starts a metrics server with a pre-configured registry
func StartMetricsServerWithRegistry(cfg Config, registry *prometheus.Registry, logger *logrus.Logger) *Server {
	if !cfg.Enabled {
		logger.Info("Metrics server disabled")
		return nil
	}

	server := NewServer(cfg.Host, cfg.Port, logger, registry)
	server.Start()
	return server
}
