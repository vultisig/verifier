package main

import (
	"context"
	"fmt"

	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/logging"
	internalMetrics "github.com/vultisig/verifier/internal/metrics"
	"github.com/vultisig/verifier/internal/storage/postgres"
	fee_tx_indexer "github.com/vultisig/verifier/internal/tx_indexer"
	"github.com/vultisig/verifier/plugin/metrics"
	"github.com/vultisig/verifier/plugin/tx_indexer"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/storage"
)

func main() {
	ctx := context.Background()

	cfg, err := config.ReadTxIndexerConfig()
	if err != nil {
		panic(fmt.Errorf("config.ReadTxIndexerConfig: %w", err))
	}

	logger := logging.NewLogger(cfg.LogFormat)

	rpcs, err := tx_indexer.Rpcs(ctx, cfg.Rpc)
	if err != nil {
		panic(fmt.Errorf("tx_indexer.Rpcs: %w", err))
	}

	txIndexerStore, err := storage.NewPostgresTxIndexStore(ctx, cfg.Database.DSN)
	if err != nil {
		panic(fmt.Errorf("storage.NewPostgresTxIndexStore: %w", err))
	}

	backendDB, err := postgres.NewPostgresBackend(cfg.Database.DSN, nil, "")
	if err != nil {
		panic(fmt.Sprintf("failed to initialize database: %v", err))
	}

	// Initialize metrics based on configuration
	var txMetrics metrics.TxIndexerMetrics

	if cfg.Metrics.Enabled {
		logger.Info("Metrics enabled, setting up Prometheus metrics")

		// Start metrics HTTP server with TX indexer metrics
		metricsConfig := internalMetrics.Config{
			Enabled: true,
			Host:    cfg.Metrics.Host,
			Port:    cfg.Metrics.Port,
		}
		_ = internalMetrics.StartMetricsServer(metricsConfig, []string{internalMetrics.ServiceTxIndexer}, logger)

		// Create TX indexer metrics implementation
		txMetrics = internalMetrics.NewTxIndexerMetrics()
	} else {
		logger.Info("Metrics disabled, using no-op implementation")
		txMetrics = metrics.NewNilTxIndexerMetrics()
	}

	worker := tx_indexer.NewWorker(
		logger,
		cfg.Interval,
		cfg.IterationTimeout,
		cfg.MarkLostAfter,
		cfg.Concurrency,
		txIndexerStore,
		rpcs,
		txMetrics,
	)

	feeIndexer := fee_tx_indexer.NewFeeIndexer(
		logger,
		backendDB,
		worker,
	)

	err = feeIndexer.Run()
	if err != nil {
		panic(fmt.Errorf("failed to start feeIndexer: %w", err))
	}
}
