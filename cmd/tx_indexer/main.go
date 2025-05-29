package main

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/common"
	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/graceful"
	"github.com/vultisig/verifier/internal/rpc"
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

	rpcBtc, err := rpc.NewBitcoinClient(cfg.Rpc.Bitcoin)
	if err != nil {
		panic(fmt.Errorf("rpc.NewBitcoinClient: %w", err))
	}

	rpcEth, err := rpc.NewEvmClient(ctx, cfg.Rpc.Ethereum)
	if err != nil {
		panic(fmt.Errorf("rpc.NewEvmClient: %w", err))
	}

	worker := tx_indexer.NewWorker(
		logger,
		cfg.Interval,
		cfg.IterationTimeout,
		cfg.MarkLostAfter,
		cfg.Concurrency,
		map[common.Chain]rpc.TxIndexer{
			common.Bitcoin:  rpcBtc,
			common.Ethereum: rpcEth,
		},
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
