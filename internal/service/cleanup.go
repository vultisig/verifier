package service

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/internal/storage"
)

type CleanupService struct {
	db            storage.DatabaseStorage
	logger        *logrus.Logger
	cleanupTicker *time.Ticker
	done          chan struct{}
}

func NewCleanupService(db storage.DatabaseStorage, logger *logrus.Logger) *CleanupService {
	return &CleanupService{
		db:     db,
		logger: logger,
		done:   make(chan struct{}),
	}
}

func (s *CleanupService) Start(ctx context.Context, interval time.Duration) {
	s.cleanupTicker = time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-s.cleanupTicker.C:
				if err := s.db.CleanupExpiredNonces(ctx); err != nil {
					s.logger.Errorf("Failed to cleanup expired nonces: %v", err)
					// Add exponential backoff on error
					time.Sleep(time.Second * 5)
				}
			case <-ctx.Done():
				s.cleanupTicker.Stop()
				close(s.done)
				return
			}
		}
	}()
}

func (s *CleanupService) Stop() {
	if s.cleanupTicker != nil {
		s.cleanupTicker.Stop()
		<-s.done
	} else {
		// If Start was never called, don't wait
		select {
		case <-s.done:
		default:
			close(s.done)
		}
	}
}
