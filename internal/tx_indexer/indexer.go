package tx_indexer

import (
	"context"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/tx_indexer/pkg/rpc"
	"github.com/vultisig/verifier/tx_indexer/pkg/storage"
	"github.com/vultisig/verifier/types"
	"golang.org/x/sync/errgroup"

	vstorage "github.com/vultisig/verifier/internal/storage"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/graceful"
	"github.com/vultisig/verifier/tx_indexer"
)

type FeeIndexer struct {
	logger *logrus.Logger
	db     vstorage.DatabaseStorage
	worker *tx_indexer.Worker
}

func NewFeeIndexer(logger *logrus.Logger, db vstorage.DatabaseStorage, worker *tx_indexer.Worker) *FeeIndexer {
	return &FeeIndexer{
		logger: logger.WithField("pkg", "tx_indexer.worker").Logger,
		worker: worker,
		db:     db,
	}
}

func (fi *FeeIndexer) start(aliveCtx context.Context) error {
	err := fi.updatePendingTxs()
	if err != nil {
		return fmt.Errorf("w.updatePendingTxs: %w", err)
	}

	for {
		select {
		case <-aliveCtx.Done():
			fi.logger.Infof("context done & no processing: stop worker")
			return nil
		case <-time.After(fi.worker.Interval()):
			er := fi.updatePendingTxs()
			if er != nil {
				fi.logger.Errorf("processing error, continue loop: %v", er)
			}
		}
	}
}

func (fi *FeeIndexer) Run() error {
	ctx, stop := context.WithCancel(context.Background())

	go func() {
		graceful.HandleSignals(stop)
		fi.logger.Info("got exit signal, will stop after current processing step finished...")
	}()

	err := fi.start(ctx)
	if err != nil {
		return fmt.Errorf("w.start: %w", err)
	}
	return nil
}

func (fi *FeeIndexer) updateTxStatus(ctx context.Context, tx storage.Tx) error {
	newStatus, err := fi.worker.UpdateTxStatus(ctx, tx)
	if err != nil {
		return fmt.Errorf("w.UpdateTxStatus: %w", err)
	}

	if newStatus == nil {
		return fmt.Errorf("new status is nil")
	}

	if tx.PluginID == types.PluginVultisigFees_feee && *newStatus == rpc.TxOnChainSuccess {
		err = fi.db.WithTransaction(ctx, func(ctx context.Context, dbTx pgx.Tx) error {
			var err error

			//Find plugin
			pluginInfo, err := fi.db.FindPluginById(ctx, nil, tx.PluginID)
			if err != nil {
				return err
			}

			txFee := new(big.Int)
			for _, pricing := range pluginInfo.Pricing {
				if pricing.Type == types.PricingTypePerTx {
					txFee = new(big.Int).SetUint64(pricing.Amount)
				}
			}
			if txFee.Cmp(big.NewInt(0)) < 1 {
				return nil
			}

			//Insert fee
			err = fi.db.InsertFee(ctx, dbTx, &types.Fee{
				PublicKey:      tx.FromPublicKey,
				TxType:         types.TxTypeDebit,
				Amount:         txFee,
				FeeType:        types.FeeTxExecFee,
				UnderlyingType: "tx_indexer_record",
				UnderlyingID:   tx.ID.String(),
			})
			if err != nil {
				return err
			}

			return nil
		})
	}
	return nil
}

func (fi *FeeIndexer) updatePendingTxs() error {
	ctx, cancel := context.WithTimeout(context.Background(), fi.worker.IterationTimeout())
	defer cancel()

	fi.logger.Info("worker tick")

	eg := &errgroup.Group{}
	eg.SetLimit(fi.worker.Concurrency())
	ch := fi.worker.TxIndexerRepo().GetPendingTxs(ctx)
	count := &atomic.Uint64{}
	for _row := range ch {
		row := _row
		eg.Go(func() error {
			if row.Err != nil {
				return fmt.Errorf("row.Err: %w", row.Err)
			}

			err := fi.updateTxStatus(ctx, row.Row)
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

	fi.logger.WithField("tx_count", count.Load()).Info("tx statuses updated")
	return nil
}
