package main

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/graceful"
	"github.com/vultisig/verifier/internal/storage/postgres"
	"github.com/vultisig/verifier/internal/tx_indexer"
	"golang.org/x/sync/errgroup"
)

func main() {
	ctx, stop := context.WithCancel(context.Background())

	logger := logrus.New()

	cfg, err := config.ReadTxIndexerConfig()
	if err != nil {
		panic(fmt.Errorf("config.ReadTxIndexerConfig: %w", err))
	}

	rpcs, err := tx_indexer.Rpcs(ctx, cfg.Rpc)
	if err != nil {
		panic(fmt.Errorf("tx_indexer.Rpcs: %w", err))
	}

	db, err := postgres.NewPostgresBackend(cfg.Database.DSN, nil)
	if err != nil {
		panic(fmt.Errorf("postgres.NewPostgresBackend: %w", err))
	}

	worker := tx_indexer.NewWorker(
		logger,
		cfg.Interval,
		cfg.IterationTimeout,
		cfg.MarkLostAfter,
		cfg.Concurrency,
		db,
		rpcs,
	)

	var eg errgroup.Group
	eg.Go(func() error {
		return worker.Start(ctx)
	})
	eg.Go(func() error {
		graceful.HandleSignals(stop)
		logger.Info("got exit signal, will stop after current processing step finished...")
		return nil
	})
	err = eg.Wait()
	if err != nil {
		panic(fmt.Errorf("failed to start worker: %w", err))
	}
}
