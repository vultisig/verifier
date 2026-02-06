package tx_indexer

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/vultisig/verifier/plugin/metrics"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/graceful"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/rpc"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/storage"
	"github.com/vultisig/vultisig-go/common"
)

type Worker struct {
	logger           *logrus.Logger
	repo             storage.TxIndexerRepo
	interval         time.Duration
	iterationTimeout time.Duration
	markLostAfter    time.Duration // default fallback for UTXO chains
	concurrency      int
	clients          SupportedRpcs
	metrics          metrics.TxIndexerMetrics
}

// getMarkLostAfter returns chain-specific timeout for marking transactions as lost.
func (w *Worker) getMarkLostAfter(chain common.Chain) time.Duration {
	switch {
	case chain == common.Solana:
		return 2 * time.Minute
	case chain == common.XRP:
		return 5 * time.Minute
	case chain.IsEvm():
		return 30 * time.Minute
	default:
		return w.markLostAfter
	}
}

func NewWorker(
	logger *logrus.Logger,
	interval time.Duration,
	iterationTimeout time.Duration,
	markLostAfter time.Duration,
	concurrency int,
	repo storage.TxIndexerRepo,
	clients SupportedRpcs,
	txMetrics metrics.TxIndexerMetrics,
) *Worker {
	return &Worker{
		logger:           logger.WithField("pkg", "tx_indexer.worker").Logger,
		repo:             repo,
		interval:         interval,
		iterationTimeout: iterationTimeout,
		markLostAfter:    markLostAfter,
		concurrency:      concurrency,
		clients:          clients,
		metrics:          txMetrics,
	}
}

func (w *Worker) Interval() time.Duration {
	return w.interval
}

func (w *Worker) Concurrency() int {
	return w.concurrency
}

func (w *Worker) MarkLostAfter() time.Duration {
	return w.markLostAfter
}

func (w *Worker) IterationTimeout() time.Duration {
	return w.iterationTimeout
}

func (w *Worker) TxIndexerRepo() storage.TxIndexerRepo {
	return w.repo
}

func (w *Worker) Metrics() metrics.TxIndexerMetrics {
	return w.metrics
}

func (w *Worker) Clients() SupportedRpcs {
	return w.clients
}

func (w *Worker) start(aliveCtx context.Context) error {
	err := w.updatePendingTxs()
	if err != nil {
		return fmt.Errorf("w.updatePendingTxs: %w", err)
	}

	for {
		select {
		case <-aliveCtx.Done():
			w.logger.Infof("context done & no processing: stop worker")
			return nil
		case <-time.After(w.interval):
			er := w.updatePendingTxs()
			if er != nil {
				w.logger.Errorf("processing error, continue loop: %v", er)
			}
		}
	}
}

func (w *Worker) Run() error {
	ctx, stop := context.WithCancel(context.Background())

	go func() {
		graceful.HandleSignals(stop)
		w.logger.Info("got exit signal, will stop after current processing step finished...")
	}()

	err := w.start(ctx)
	if err != nil {
		return fmt.Errorf("w.start: %w", err)
	}
	return nil
}

func (w *Worker) UpdateTxStatus(ctx context.Context, tx storage.Tx) (*rpc.TxOnChainStatus, error) {
	if tx.BroadcastedAt == nil {
		return nil, errors.New("unexpected tx.BroadcastedAt == nil, tx_id=" + tx.ID.String())
	}
	if tx.TxHash == nil {
		return nil, errors.New("unexpected tx.TxHash == nil, tx_id=" + tx.ID.String())
	}
	if tx.StatusOnChain == nil {
		return nil, errors.New("unexpected tx.StatusOnChain == nil, tx_id=" + tx.ID.String())
	}

	fields := tx.Fields()
	chain := common.Chain(tx.ChainID)

	// Record processing attempt
	w.metrics.RecordProcessing(chain)

	if time.Now().After((*tx.BroadcastedAt).Add(w.getMarkLostAfter(chain))) {
		err := w.repo.SetLost(ctx, tx.ID, "timeout waiting for confirmation")
		if err != nil {
			w.metrics.RecordProcessingError(chain, "set_lost_timeout")
			return nil, fmt.Errorf("w.repo.SetLost: %w", err)
		}
		w.logger.WithFields(fields).Info("updated as lost (timeout since broadcast)")
		newStatus := rpc.TxOnChainFail
		w.metrics.RecordTransactionStatus(chain, string(newStatus))
		return &newStatus, nil
	}

	client, ok := w.clients[chain]
	if !ok {
		err := w.repo.SetLost(ctx, tx.ID, "chain not supported")
		if err != nil {
			w.metrics.RecordProcessingError(chain, "set_lost_unimplemented")
			return nil, fmt.Errorf("w.repo.SetLost: %w", err)
		}
		w.logger.WithFields(fields).Infof(
			"updated as lost (rpc unimplemented, chain=%s, tx_id=%s)",
			chain.String(),
			tx.ID.String(),
		)
		newStatus := rpc.TxOnChainFail
		w.metrics.RecordTransactionStatus(chain, string(newStatus))
		return &newStatus, nil
	}

	result, err := client.GetTxStatus(ctx, *tx.TxHash)
	if err != nil {
		w.metrics.RecordRPCError(chain)
		return nil, fmt.Errorf("client.GetTxStatus: %w", err)
	}
	if result.Status == *tx.StatusOnChain {
		w.logger.WithFields(fields).Info("status didn't changed since last call")
		return tx.StatusOnChain, nil
	}

	var errorMsg *string
	if result.Status == rpc.TxOnChainFail && result.ErrorMessage != "" {
		errorMsg = &result.ErrorMessage
	}

	err = w.repo.SetOnChainStatus(ctx, tx.ID, result.Status, errorMsg)
	if err != nil {
		w.metrics.RecordProcessingError(chain, "set_status")
		return nil, fmt.Errorf("w.repo.SetOnChainStatus: %w", err)
	}

	w.metrics.RecordTransactionStatus(chain, string(result.Status))

	w.logger.WithFields(fields).Infof("status updated, newStatus=%s", result.Status)
	return &result.Status, nil
}

func (w *Worker) updatePendingTxs() error {
	ctx, cancel := context.WithTimeout(context.Background(), w.iterationTimeout)
	defer cancel()

	start := time.Now()
	w.logger.Info("worker tick")

	// Update last processing timestamp
	w.metrics.SetLastProcessingTimestamp(float64(time.Now().Unix()))

	eg := &errgroup.Group{}
	eg.SetLimit(w.concurrency)
	ch := w.repo.GetPendingTxs(ctx)
	count := &atomic.Uint64{}

	for _row := range ch {
		row := _row
		eg.Go(func() error {
			if row.Err != nil {
				return fmt.Errorf("row.Err: %w", row.Err)
			}

			_, err := w.UpdateTxStatus(ctx, row.Row)
			if err != nil {
				return fmt.Errorf("w.updateTxStatus: %w", err)
			}
			count.Add(1)
			return nil
		})
	}

	err := eg.Wait()
	if err != nil {
		return fmt.Errorf("eg.Wait: %w", err)
	}

	// Record iteration duration for each supported chain
	duration := time.Since(start).Seconds()
	for chain := range w.clients {
		w.metrics.RecordIterationDuration(chain, duration)
	}

	w.logger.WithField("tx_count", count.Load()).Info("tx statuses updated")
	return nil
}
