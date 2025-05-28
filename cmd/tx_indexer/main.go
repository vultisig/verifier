package main

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/internal/graceful"
	"github.com/vultisig/verifier/internal/tx_indexer"
	"golang.org/x/sync/errgroup"
)

func main() {
	ctx, stop := context.WithCancel(context.Background())

	logger := logrus.New()

	worker := tx_indexer.NewWorker()

	var eg errgroup.Group
	eg.Go(func() error {
		return worker.Start(ctx)
	})
	eg.Go(func() error {
		graceful.HandleSignals(stop)
		logger.Info("got exit signal, will stop after current processing step finished...")
		return nil
	})
	err := eg.Wait()
	if err != nil {
		panic(fmt.Errorf("failed to start worker: %w", err))
	}
}
