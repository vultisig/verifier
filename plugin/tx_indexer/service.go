package tx_indexer

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/mobile-tss-lib/tss"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/rpc"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/storage"
	"github.com/vultisig/verifier/types"
	"github.com/vultisig/vultisig-go/common"
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

// Returns a tracked tx by its ID.
func (t *Service) GetTxByID(ctx context.Context, txID uuid.UUID) (storage.Tx, error) {
	r, err := t.repo.GetTxByID(ctx, txID)
	if err != nil {
		return storage.Tx{}, fmt.Errorf("t.repo.GetTxByID: %w", err)
	}
	return r, nil
}

func (t *Service) GetTxsInTimeRange(
	ctx context.Context,
	policyID uuid.UUID,
	from, to time.Time,
) ([]storage.Tx, error) {
	ch := t.repo.GetTxsInTimeRange(ctx, policyID, from, to)
	txs, err := storage.AllFromRowsStream(ch)
	if err != nil {
		return nil, fmt.Errorf("failed to get txs in time range: %w", err)
	}
	return txs, nil
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
	sigs map[string]tss.KeysignResponse,
) error {
	tx, err := t.repo.GetTxByID(ctx, txID)
	if err != nil {
		return fmt.Errorf("t.repo.GetTxByID: %w", err)
	}

	client, ok := t.chains[chainID]
	if !ok {
		return fmt.Errorf("client for chain not found: %s", chainID)
	}

	body, err := base64.StdEncoding.DecodeString(tx.ProposedTxHex)
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

func (t *Service) GetByPluginIDAndPublicKey(
	c context.Context,
	pluginID types.PluginID,
	publicKey string,
	skip, take uint32,
) ([]storage.Tx, uint32, error) {
	var (
		txs        []storage.Tx
		totalCount uint32
	)

	eg, ctx := errgroup.WithContext(c)
	eg.Go(func() error {
		ch := t.repo.GetByPluginIDAndPublicKey(ctx, pluginID, publicKey, skip, take)
		r, err := storage.AllFromRowsStream(ch)
		if err != nil {
			return fmt.Errorf("storage.AllFromRowsStream: %w", err)
		}
		txs = r
		return nil
	})
	eg.Go(func() error {
		r, err := t.repo.CountByPluginIDAndPublicKey(ctx, pluginID, publicKey)
		if err != nil {
			return fmt.Errorf("t.repo.CountByPluginIDAndPublicKey: %w", err)
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
