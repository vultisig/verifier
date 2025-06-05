package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/mobile-tss-lib/tss"
	"github.com/vultisig/verifier/common"
	"github.com/vultisig/verifier/internal/storage"
	"github.com/vultisig/verifier/internal/types"
)

type TxIndexerService struct {
	logger *logrus.Logger
	repo   storage.TxIndexerRepository
	tss    map[common.Chain]types.TxIndexerTss
}

func NewTxIndexerService(
	logger *logrus.Logger,
	repo storage.TxIndexerRepository,
	tss map[common.Chain]types.TxIndexerTss,
) *TxIndexerService {
	return &TxIndexerService{
		logger: logger.WithField("pkg", "service.tx_indexer").Logger,
		repo:   repo,
		tss:    tss,
	}
}

func (t *TxIndexerService) CreateTx(ctx context.Context, req types.CreateTxDto) (types.Tx, error) {
	r, err := t.repo.CreateTx(ctx, req)
	if err != nil {
		return types.Tx{}, fmt.Errorf("t.repo.CreateTx: %w", err)
	}
	return r, nil
}

func (t *TxIndexerService) SetStatus(ctx context.Context, txID uuid.UUID, status types.TxStatus) error {
	err := t.repo.SetStatus(ctx, txID, status)
	if err != nil {
		return fmt.Errorf("t.repo.SetStatus: %w", err)
	}
	return nil
}

func (t *TxIndexerService) SetOnChainStatus(ctx context.Context, txID uuid.UUID, status types.TxOnChainStatus) error {
	err := t.repo.SetOnChainStatus(ctx, txID, status)
	if err != nil {
		return fmt.Errorf("t.repo.SetOnChainStatus: %w", err)
	}
	return nil
}

func (t *TxIndexerService) SetSignedAndBroadcasted(
	ctx context.Context,
	chainID common.Chain,
	txID uuid.UUID,
	sigs []tss.KeysignResponse,
) error {
	tx, err := t.repo.GetTxByID(ctx, txID)
	if err != nil {
		return fmt.Errorf("t.repo.GetTxByID: %w", err)
	}

	client, ok := t.tss[chainID]
	if !ok {
		return fmt.Errorf("tss client for chain %s not found", chainID)
	}

	txHash, err := client.ComputeTxHash(tx.ProposedTxHex, sigs)
	if err != nil {
		if errors.Is(err, types.ErrChainNotImplemented) {
			t.logger.WithFields(logrus.Fields{
				"chain_id": chainID,
			}).Error("ComputeTxHash: chain not implemented")
			return nil
		}
		return fmt.Errorf("client.ComputeTxHash: %w", err)
	}

	err = t.repo.SetSignedAndBroadcasted(ctx, txID, txHash)
	if err != nil {
		return fmt.Errorf("t.repo.SetSignedAndBroadcasted: %w", err)
	}
	return nil
}
