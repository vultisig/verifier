package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/storage"
	vtypes "github.com/vultisig/verifier/types"
	"github.com/vultisig/vultisig-go/common"
)

type Fees interface {
	PublicKeyGetFeeInfo(ctx context.Context, publicKey string) ([]*vtypes.Fee, error)
	MarkFeesCollected(ctx context.Context, feeIDs []uint64, network, txHash string, amount uint64) error
	IssueCredit(ctx context.Context, publicKey string, amount uint64, reason string) error
	GetUserFees(ctx context.Context, publicKey string) (*vtypes.UserFeeStatus, error)
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

func (s *FeeService) MarkFeesCollected(ctx context.Context, feeIDs []uint64, network, txHash string, amount uint64) error {
	//TODO: add handling for fees in different networks
	_, err := common.FromString(network)
	if err != nil {
		return fmt.Errorf("invalid network: %w", err)
	}

	err = s.db.WithTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return s.db.MarkFeesCollected(ctx, tx, feeIDs, txHash, amount)
	})
	if err != nil {
		return err
	}
	return nil
}

// IssueCredit for promo, bonuses, etc.
func (s *FeeService) IssueCredit(ctx context.Context, publicKey string, amount uint64, reason string) error {
	metadata := map[string]interface{}{
		"reason":    reason,
		"issued_at": time.Now().UTC(),
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	creditFee := &vtypes.Fee{
		PublicKey:      publicKey,
		TxType:         vtypes.TxTypeCredit,
		Amount:         amount,
		FeeType:        "free_credit",
		Metadata:       metadataJSON,
		UnderlyingType: reason,
	}

	_, err = s.db.InsertFee(ctx, nil, creditFee)
	if err != nil {
		return fmt.Errorf("failed to issue credit: %w", err)
	}

	return nil
}

func (s *FeeService) GetUserFees(ctx context.Context, publicKey string) (*vtypes.UserFeeStatus, error) {
	status, err := s.db.GetUserFees(ctx, publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get user fees: %w", err)
	}

	return status, nil
}
