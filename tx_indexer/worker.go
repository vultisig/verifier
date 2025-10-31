package tx_indexer

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/tx_indexer/pkg/graceful"
	"github.com/vultisig/verifier/tx_indexer/pkg/rpc"
	"github.com/vultisig/verifier/tx_indexer/pkg/storage"
	"github.com/vultisig/vultisig-go/common"
	"golang.org/x/sync/errgroup"
)

type Worker struct {
	logger           *logrus.Logger
	repo             storage.TxIndexerRepo
	interval         time.Duration
	iterationTimeout time.Duration
	markLostAfter    time.Duration
	concurrency      int
	clients          SupportedRpcs
}

func NewWorker(
	logger *logrus.Logger,
	interval time.Duration,
	iterationTimeout time.Duration,
	markLostAfter time.Duration,
	concurrency int,
	repo storage.TxIndexerRepo,
	clients SupportedRpcs,
) *Worker {
	return &Worker{
		logger:           logger.WithField("pkg", "tx_indexer.worker").Logger,
		repo:             repo,
		interval:         interval,
		iterationTimeout: iterationTimeout,
		markLostAfter:    markLostAfter,
		concurrency:      concurrency,
		clients:          clients,
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

	if time.Now().After((*tx.BroadcastedAt).Add(w.markLostAfter)) {
		err := w.repo.SetLost(ctx, tx.ID)
		if err != nil {
			return nil, fmt.Errorf("w.repo.SetLost: %w", err)
		}
		w.logger.WithFields(fields).Info("updated as lost (timeout since broadcast)")
		return nil, nil
	}

	client, ok := w.clients[common.Chain(tx.ChainID)]
	if !ok {
		err := w.repo.SetLost(ctx, tx.ID)
		if err != nil {
			return nil, fmt.Errorf("w.repo.SetLost: %w", err)
		}
		w.logger.WithFields(fields).Infof(
			"updated as lost (rpc unimplemented, chain=%s, tx_id=%s)",
			common.Chain(tx.ChainID).String(),
			tx.ID.String(),
		)
		return nil, nil
	}

	newStatus, err := client.GetTxStatus(ctx, *tx.TxHash)
	if err != nil {
		return nil, fmt.Errorf("client.GetTxStatus: %w", err)
	}
	if newStatus == *tx.StatusOnChain {
		w.logger.WithFields(fields).Info("status didn't changed since last call")
		return nil, nil
	}

	err = w.repo.SetOnChainStatus(ctx, tx.ID, newStatus)
	if err != nil {
		return nil, fmt.Errorf("w.repo.SetOnChainStatus: %w", err)
	}
	w.logger.WithFields(fields).Infof("status updated, newStatus=%s", newStatus)

	return &newStatus, nil
}

func (w *Worker) updatePendingTxs() error {
	ctx, cancel := context.WithTimeout(context.Background(), w.iterationTimeout)
	defer cancel()

	w.logger.Info("worker tick")

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

	w.logger.WithField("tx_count", count.Load()).Info("tx statuses updated")
	return nil
}
