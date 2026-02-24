package metrics

import (
	"context"
	"crypto/subtle"
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
	token  string
}

// NewServer creates a new metrics server with a custom registry
func NewServer(host string, port int, token string, logger *logrus.Logger, registry *prometheus.Registry) *Server {
	s := &Server{
		logger: logger,
		token:  token,
	}

	mux := http.NewServeMux()

	var metricsHandler http.Handler
	if registry != nil {
		metricsHandler = promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	} else {
		metricsHandler = promhttp.Handler()
	}

	if token != "" {
		logger.Info("Metrics endpoint protected with Bearer token authentication")
		mux.Handle("/metrics", s.authMiddleware(metricsHandler))
	} else {
		logger.Warn("Metrics endpoint is NOT protected - consider setting metrics.token in config")
		mux.Handle("/metrics", metricsHandler)
	}

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

	s.server = server
	return s
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(authHeader, prefix) {
			http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
			return
		}

		token := authHeader[len(prefix):]
		if token == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}

		if !s.validateToken(token) {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) validateToken(token string) bool {
	tokenBytes := []byte(token)
	return subtle.ConstantTimeCompare([]byte(s.token), tokenBytes) == 1
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

	server := NewServer(cfg.Host, cfg.Port, cfg.Token, logger, registry)
	server.Start()
	return server
}

// StartMetricsServerWithRegistry starts a metrics server with a pre-configured registry
func StartMetricsServerWithRegistry(cfg Config, registry *prometheus.Registry, logger *logrus.Logger) *Server {
	if !cfg.Enabled {
		logger.Info("Metrics server disabled")
		return nil
	}

	server := NewServer(cfg.Host, cfg.Port, cfg.Token, logger, registry)
	server.Start()
	return server
}
