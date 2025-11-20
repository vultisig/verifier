package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// Server provides a standalone HTTP server for Prometheus metrics
type Server struct {
	server   *http.Server
	registry *Registry
	logger   *logrus.Logger
}

// Config holds configuration for the metrics server
type Config struct {
	Host    string `mapstructure:"host" json:"host"`
	Port    int    `mapstructure:"port" json:"port"`
	Enabled bool   `mapstructure:"enabled" json:"enabled"`
}

// DefaultConfig returns default metrics server configuration
func DefaultConfig() Config {
	return Config{
		Host:    "0.0.0.0",
		Port:    9090,
		Enabled: false,
	}
}

// NewServer creates a new metrics server
func NewServer(config Config, registry *Registry, logger *logrus.Logger) *Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", registry.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	return &Server{
		server:   server,
		registry: registry,
		logger:   logger.WithField("component", "metrics-server").Logger,
	}
}

// Start starts the metrics server
func (s *Server) Start() error {
	s.logger.Infof("Starting metrics server on %s", s.server.Addr)
	
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("metrics server failed to start: %w", err)
	}
	return nil
}

// Stop gracefully stops the metrics server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping metrics server")
	
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to gracefully shutdown metrics server: %w", err)
	}
	
	s.logger.Info("Metrics server stopped")
	return nil
}

// StartAsync starts the metrics server in a goroutine
func (s *Server) StartAsync() <-chan error {
	errChan := make(chan error, 1)
	
	go func() {
		if err := s.Start(); err != nil {
			errChan <- err
		}
		close(errChan)
	}()
	
	return errChan
}