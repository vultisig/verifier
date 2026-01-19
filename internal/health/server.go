package health

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// Server provides a simple HTTP server for health checks.
// It exposes a /healthz endpoint that returns 200 OK when the service is running.
type Server struct {
	port   int
	server *http.Server
}

// New creates a new health server on the specified port.
func New(port int) *Server {
	return &Server{
		port: port,
	}
}

// Start starts the health server and blocks until the context is cancelled.
// It gracefully shuts down when the context is done.
func (s *Server) Start(ctx context.Context, logger *logrus.Logger) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.server.Shutdown(shutdownCtx); err != nil {
			logger.Errorf("health server shutdown error: %v", err)
		}
	}()

	logger.Infof("health probe server listening on :%d", s.port)

	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("health server failed: %w", err)
	}

	return nil
}
