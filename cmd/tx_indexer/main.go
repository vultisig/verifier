package main

import (
	"context"
	"fmt"

	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/logging"
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

	backendDB, err := postgres.NewPostgresBackend(cfg.Database.DSN, nil)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize database: %v", err))
	}

	// Use no-op metrics implementation, disable metrics for now
	worker := tx_indexer.NewWorker(
		logger,
		cfg.Interval,
		cfg.IterationTimeout,
		cfg.MarkLostAfter,
		cfg.Concurrency,
		txIndexerStore,
		rpcs,
		metrics.NewNilTxIndexerMetrics(), // no-op metrics implementation
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
