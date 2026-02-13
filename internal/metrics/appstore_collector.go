package metrics

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

type DatabaseQuerier interface {
	GetInstallationsByPlugin(ctx context.Context) (map[string]int64, error)
	GetPoliciesByPlugin(ctx context.Context) (map[string]int64, error)
	GetFeesByPlugin(ctx context.Context) (map[string]int64, error)
}

type AppStoreCollector struct {
	db       DatabaseQuerier
	metrics  *AppStoreMetrics
	logger   *logrus.Logger
	interval time.Duration
	stopCh   chan struct{}
	doneCh   chan struct{}
}

func NewAppStoreCollector(db DatabaseQuerier, metrics *AppStoreMetrics, logger *logrus.Logger, interval time.Duration) *AppStoreCollector {
	return &AppStoreCollector{
		db:       db,
		metrics:  metrics,
		logger:   logger,
		interval: interval,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

func (c *AppStoreCollector) Start() {
	c.logger.Info("Starting App Store metrics collector")

	go func() {
		defer close(c.doneCh)

		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		c.collect()

		for {
			select {
			case <-ticker.C:
				c.collect()
			case <-c.stopCh:
				c.logger.Info("Stopping App Store metrics collector")
				return
			}
		}
	}()
}

func (c *AppStoreCollector) Stop() {
	close(c.stopCh)
	<-c.doneCh
}

func (c *AppStoreCollector) collect() {
	ctx := context.Background()

	installations, err := c.db.GetInstallationsByPlugin(ctx)
	if err != nil {
		c.logger.Errorf("Failed to collect installations: %v", err)
	} else {
		c.metrics.UpdateInstallations(installations)
		c.metrics.UpdateTimestamp()
	}

	policies, err := c.db.GetPoliciesByPlugin(ctx)
	if err != nil {
		c.logger.Errorf("Failed to collect policies: %v", err)
	} else {
		c.metrics.UpdatePolicies(policies)
		c.metrics.UpdateTimestamp()
	}

	fees, err := c.db.GetFeesByPlugin(ctx)
	if err != nil {
		c.logger.Errorf("Failed to collect fees: %v", err)
	} else {
		c.metrics.UpdateFees(fees)
		c.metrics.UpdateTimestamp()
	}
}
