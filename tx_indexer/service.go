package tx_indexer

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/mobile-tss-lib/tss"
	"github.com/vultisig/verifier/common"
	"github.com/vultisig/verifier/tx_indexer/pkg/rpc"
	"github.com/vultisig/verifier/tx_indexer/pkg/storage"
	"github.com/vultisig/verifier/types"
	"golang.org/x/sync/errgroup"
)

type Service struct {
	logger *logrus.Logger
	repo   storage.TxIndexerRepo
	chains SupportedChains
}

func NewService(
	logger *logrus.Logger,
	repo storage.TxIndexerRepo,
	chains SupportedChains,
) *Service {
	return &Service{
		logger: logger.WithField("pkg", "service.tx_indexer").Logger,
		repo:   repo,
		chains: chains,
	}
}

func (t *Service) CreateTx(ctx context.Context, req storage.CreateTxDto) (storage.Tx, error) {
	r, err := t.repo.CreateTx(ctx, req)
	if err != nil {
		return storage.Tx{}, fmt.Errorf("t.repo.CreateTx: %w", err)
	}
	return r, nil
}

func (t *Service) GetTxsInTimeRange(
	ctx context.Context,
	chainID common.Chain,
	pluginID types.PluginID,
	policyID uuid.UUID,
	tokenID, recipientPublicKey string,
	from, to time.Time,
) <-chan storage.RowsStream[storage.Tx] {
	return t.repo.GetTxsInTimeRange(
		ctx,
		chainID,
		pluginID,
		policyID,
		tokenID,
		recipientPublicKey,
		from,
		to,
	)
}

func (t *Service) GetTxInTimeRange(
	ctx context.Context,
	chainID common.Chain,
	pluginID types.PluginID,
	policyID uuid.UUID,
	tokenID, recipientPublicKey string,
	from, to time.Time,
) (storage.Tx, error) {
	tx, err := t.repo.GetTxInTimeRange(
		ctx,
		chainID,
		pluginID,
		policyID,
		tokenID,
		recipientPublicKey,
		from,
		to,
	)
	if err != nil {
		return storage.Tx{}, fmt.Errorf("t.repo.GetTxInTimeRange: %w", err)
	}
	return tx, nil
}

func (t *Service) SetStatus(ctx context.Context, txID uuid.UUID, status storage.TxStatus) error {
	err := t.repo.SetStatus(ctx, txID, status)
	if err != nil {
		return fmt.Errorf("t.repo.SetStatus: %w", err)
	}
	return nil
}

func (t *Service) SetOnChainStatus(ctx context.Context, txID uuid.UUID, status rpc.TxOnChainStatus) error {
	err := t.repo.SetOnChainStatus(ctx, txID, status)
	if err != nil {
		return fmt.Errorf("t.repo.SetOnChainStatus: %w", err)
	}
	return nil
}

func (t *Service) SetSignedAndBroadcasted(
	ctx context.Context,
	chainID common.Chain,
	txID uuid.UUID,
	sigs []tss.KeysignResponse,
) error {
	tx, err := t.repo.GetTxByID(ctx, txID)
	if err != nil {
		return fmt.Errorf("t.repo.GetTxByID: %w", err)
	}

	client, ok := t.chains[chainID]
	if !ok {
		return fmt.Errorf("client for chain not found: %s", chainID)
	}

	body, err := hexutil.Decode(tx.ProposedTxHex)
	if err != nil {
		return fmt.Errorf("failed to decode proposed tx: %w", err)
	}

	txHash, err := client.ComputeTxHash(body, sigs)
	if err != nil {
		return fmt.Errorf("client.ComputeTxHash: %w", err)
	}

	err = t.repo.SetSignedAndBroadcasted(ctx, txID, txHash)
	if err != nil {
		return fmt.Errorf("t.repo.SetSignedAndBroadcasted: %w", err)
	}
	return nil
}

func (t *Service) GetByPolicyID(
	c context.Context,
	policyID uuid.UUID,
	skip, take uint32,
) ([]storage.Tx, uint32, error) {
	var (
		txs        []storage.Tx
		totalCount uint32
	)

	eg, ctx := errgroup.WithContext(c)
	eg.Go(func() error {
		ch := t.repo.GetByPolicyID(ctx, policyID, skip, take)
		r, err := storage.AllFromRowsStream(ch)
		if err != nil {
			return fmt.Errorf("storage.AllFromRowsStream: %w", err)
		}
		txs = r
		return nil
	})
	eg.Go(func() error {
		r, err := t.repo.CountByPolicyID(ctx, policyID)
		if err != nil {
			return fmt.Errorf("t.repo.CountByPolicyID: %w", err)
		}
		totalCount = r
		return nil
	})
	err := eg.Wait()
	if err != nil {
		return nil, 0, fmt.Errorf("eg.Wait: %w", err)
	}

	return txs, totalCount, nil
}
