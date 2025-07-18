package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/internal/storage"
	itypes "github.com/vultisig/verifier/internal/types"
)

type Fees interface {
	PublicKeyGetFeeInfo(ctx context.Context, publicKey string) (*itypes.FeeHistoryDto, error)
	MarkFeesCollected(ctx context.Context, collectedAt time.Time, ids []uuid.UUID, txHash string) ([]itypes.FeeDto, error)
}

var _ Fees = (*FeeService)(nil)

type FeeService struct {
	db     storage.DatabaseStorage
	logger *logrus.Logger
	client *asynq.Client
}

func NewFeeService(db storage.DatabaseStorage,
	client *asynq.Client, logger *logrus.Logger) (*FeeService, error) {
	if db == nil {
		return nil, fmt.Errorf("database storage cannot be nil")
	}
	return &FeeService{
		db:     db,
		logger: logger.WithField("service", "fee").Logger,
		client: client,
	}, nil
}

func (s *FeeService) PublicKeyGetFeeInfo(ctx context.Context, publicKey string) (*itypes.FeeHistoryDto, error) {

	fees, err := s.db.GetFeesByPublicKey(ctx, publicKey, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get fees: %w", err)
	}

	var totalFeesIncurred uint64
	var feesPendingCollection uint64

	ifees := make([]itypes.FeeDto, 0, len(fees))
	for _, fee := range fees {
		collected := true
		if fee.CollectedAt == nil {
			collected = false
		}
		collectedDt := ""
		if collected {
			collectedDt = fee.CollectedAt.Format(time.RFC3339)
		}
		ifee := itypes.FeeDto{
			ID:          fee.ID,
			PublicKey:   fee.PublicKey,
			PolicyId:    fee.PolicyID,
			PluginId:    fee.PluginID.String(),
			Amount:      fee.Amount,
			Collected:   collected,
			CollectedAt: collectedDt,
			ChargedAt:   fee.ChargedAt.Format(time.RFC3339),
		}
		totalFeesIncurred += fee.Amount
		if !collected {
			feesPendingCollection += fee.Amount
		}
		ifees = append(ifees, ifee)
	}

	return &itypes.FeeHistoryDto{
		Fees:                  ifees,
		TotalFeesIncurred:     totalFeesIncurred,
		FeesPendingCollection: feesPendingCollection,
	}, nil
}

func (s *FeeService) MarkFeesCollected(ctx context.Context, collectedAt time.Time, ids []uuid.UUID, txHash string) ([]itypes.FeeDto, error) {
	fees, err := s.db.MarkFeesCollected(ctx, collectedAt, ids, txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to mark fees as collected: %w", err)
	}

	feesDto := make([]itypes.FeeDto, 0, len(fees))
	for _, fee := range fees {
		feesDto = append(feesDto, itypes.FeeDto{
			ID:          fee.ID,
			PublicKey:   fee.PublicKey,
			PolicyId:    fee.PolicyID,
			PluginId:    fee.PluginID.String(),
			Amount:      fee.Amount,
			Collected:   true,
			CollectedAt: collectedAt.Format(time.RFC3339),
			ChargedAt:   fee.ChargedAt.Format(time.RFC3339),
		})
	}
	return feesDto, nil
}
