package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/storage"
	itypes "github.com/vultisig/verifier/internal/types"
)

type Fees interface {
	PublicKeyGetFeeInfo(ctx context.Context, publicKey string, since *time.Time) (*itypes.FeeHistoryDto, error)
	MarkFeesCollected(ctx context.Context, collectedAt time.Time, ids []uuid.UUID, txHash string) ([]itypes.FeeDto, error)
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

func (s *FeeService) PublicKeyGetFeeInfo(ctx context.Context, publicKey string, since *time.Time) (*itypes.FeeHistoryDto, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *FeeService) MarkFeesCollected(ctx context.Context, collectedAt time.Time, ids []uuid.UUID, txHash string) ([]itypes.FeeDto, error) {
	return nil, fmt.Errorf("not implemented")
}
