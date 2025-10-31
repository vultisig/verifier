package main

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/storage/postgres"
	fee_tx_indexer "github.com/vultisig/verifier/internal/tx_indexer"
	"github.com/vultisig/verifier/tx_indexer"
	"github.com/vultisig/verifier/tx_indexer/pkg/storage"
)

func main() {
	ctx := context.Background()

	logger := logrus.New()

	cfg, err := config.ReadTxIndexerConfig()
	if err != nil {
		panic(fmt.Errorf("config.ReadTxIndexerConfig: %w", err))
	}

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

	worker := tx_indexer.NewWorker(
		logger,
		cfg.Interval,
		cfg.IterationTimeout,
		cfg.MarkLostAfter,
		cfg.Concurrency,
		txIndexerStore,
		rpcs,
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
