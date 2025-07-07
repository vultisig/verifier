package service

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/internal/storage"
	itypes "github.com/vultisig/verifier/internal/types"
)

type Fees interface {
	PublicKeyGetFeeInfo(ctx context.Context, publicKey string) (itypes.FeeHistoryDto, error)
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

func (s *FeeService) PublicKeyGetFeeInfo(ctx context.Context, publicKey string) (itypes.FeeHistoryDto, error) {
	history := itypes.FeeHistoryDto{}

	fees, err := s.db.GetFeesByPublicKey(ctx, publicKey, true)
	if err != nil {
		return history, fmt.Errorf("failed to get fees: %w", err)
	}

	totalFeesIncurred := 0
	feesPendingCollection := 0

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

	history = itypes.FeeHistoryDto{
		Fees:                  ifees,
		TotalFeesIncurred:     totalFeesIncurred,
		FeesPendingCollection: feesPendingCollection,
	}

	return history, nil
}
