package fee_manager

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/internal/storage"
	vtypes "github.com/vultisig/verifier/types"
	"github.com/vultisig/verifier/vault"
)

type FeeManagementService struct {
	logger *logrus.Logger
	db     storage.DatabaseStorage
	vault  *vault.ManagementService
}

func NewFeeManagementService(
	logger *logrus.Logger,
	db storage.DatabaseStorage,
	vault *vault.ManagementService) *FeeManagementService {
	return &FeeManagementService{
		logger: logger,
		db:     db,
		vault:  vault,
	}
}

func (s *FeeManagementService) HandleReshareDKLS(ctx context.Context, t *asynq.Task) error {
	err := s.vault.HandleReshareDKLS(ctx, t)
	if err != nil {
		return err
	}

	var req vtypes.ReshareRequest
	if err := json.Unmarshal(t.Payload(), &req); err != nil {
		s.logger.WithError(err).Error("json.Unmarshal failed")
		return fmt.Errorf("s.RegisterInstallationFee failed: %w", asynq.SkipRetry)
	}

	if err := s.RegisterInstallationFee(ctx, vtypes.PluginID(req.PluginID), req.PublicKey); err != nil {
		s.logger.WithError(err).Error("s.RegisterInstallationFee failed")
		return fmt.Errorf("s.RegisterInstallationFee failed: %w", asynq.SkipRetry)
	}

	return nil
}

func (s *FeeManagementService) RegisterInstallationFee(ctx context.Context, pluginID vtypes.PluginID, publicKey string) error {
	return s.db.WithTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var err error

		//Find plugin
		pluginInfo, err := s.db.FindPluginById(ctx, tx, pluginID)
		if err != nil {
			return err
		}

		var installationFee uint64
		for _, pricing := range pluginInfo.Pricing {
			if pricing.Type == vtypes.PricingTypeOnce {
				installationFee = pricing.Amount
				break
			}
		}
		if installationFee == 0 {
			s.logger.WithFields(logrus.Fields{
				"plugin": pluginID,
			}).Info("installation is free")
			return nil
		}

		//Insert fee
		err = s.db.InsertFee(ctx, tx, &vtypes.Fee{
			PublicKey:      publicKey,
			TxType:         vtypes.TxTypeDebit,
			Amount:         installationFee,
			FeeType:        vtypes.FeeTypeInstallationFee,
			UnderlyingType: "plugin",
			UnderlyingID:   pluginID.String(),
		})
		if err != nil {
			return err
		}

		return nil
	})
}
