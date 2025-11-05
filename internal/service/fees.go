package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/storage"
	vtypes "github.com/vultisig/verifier/types"
	"github.com/vultisig/vultisig-go/common"
)

type Fees interface {
	PublicKeyGetFeeInfo(ctx context.Context, publicKey string) ([]*vtypes.Fee, error)
	MarkFeesCollected(ctx context.Context, id uint64, txHash, network string, amount uint64) error
}

var _ Fees = (*FeeService)(nil)

type FeeService struct {
	db        storage.DatabaseStorage
	logger    *logrus.Logger
	client    *asynq.Client
	feeConfig config.FeesConfig
}

func NewFeeService(db storage.DatabaseStorage,
	client *asynq.Client, logger *logrus.Logger, feeConfig config.FeesConfig) (*FeeService, error) {
	if db == nil {
		return nil, fmt.Errorf("database storage cannot be nil")
	}
	return &FeeService{
		db:        db,
		logger:    logger.WithField("service", "fee").Logger,
		client:    client,
		feeConfig: feeConfig,
	}, nil
}

func (s *FeeService) PublicKeyGetFeeInfo(ctx context.Context, publicKey string) ([]*vtypes.Fee, error) {
	return s.db.GetFeesByPublicKey(ctx, publicKey)
}

func (s *FeeService) MarkFeesCollected(ctx context.Context, id uint64, txHash, network string, amount uint64) error {
	chain, err := common.FromString(network)
	if err != nil {
		return err
	}

	metadata := vtypes.CreditMetadata{
		DebitFeeID: id,
		TxHash:     txHash,
		Network:    chain.String(),
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed marshaling metadata: %w", err)
	}

	feeInfo, err := s.db.GetFeeById(ctx, id)
	if err != nil {
		return fmt.Errorf("failed fetching fee: %w", err)
	}

	creditFee := &vtypes.Fee{
		PolicyID:       feeInfo.PolicyID,
		PublicKey:      feeInfo.PublicKey,
		TxType:         vtypes.TxTypeCredit,
		Amount:         amount,
		CreatedAt:      time.Now(),
		FeeType:        "fee_collection",
		Metadata:       metadataJSON,
		UnderlyingType: "tx",
		UnderlyingID:   fmt.Sprint(id),
	}
	err = s.db.InsertFee(ctx, nil, creditFee)
	if err != nil {
		return fmt.Errorf("failed inserting fee: %w", err)
	}
	return nil
}
