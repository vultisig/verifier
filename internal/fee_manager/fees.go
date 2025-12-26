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
	logger    *logrus.Logger
	db        storage.DatabaseStorage
	vault     *vault.ManagementService
	safetyMgm vault.SafetyManager
}

func NewFeeManagementService(
	logger *logrus.Logger,
	db storage.DatabaseStorage,
	vault *vault.ManagementService,
	safetyMgm vault.SafetyManager) *FeeManagementService {
	return &FeeManagementService{
		logger:    logger,
		db:        db,
		vault:     vault,
		safetyMgm: safetyMgm,
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
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	if err := s.safetyMgm.EnforceKeygen(ctx, req.PluginID); err != nil {
		s.logger.WithError(err).Error("EnforceKeygen failed")
		return fmt.Errorf("EnforceKeygen failed: %v: %w", err, asynq.SkipRetry)
	}

	if err := s.RegisterInstallation(ctx, vtypes.PluginID(req.PluginID), req.PublicKey); err != nil {
		s.logger.WithError(err).Error("s.RegisterInstallation failed")
		return fmt.Errorf("s.RegisterInstallation failed: %v: %w", err, asynq.SkipRetry)
	}

	return nil
}

func (s *FeeManagementService) RegisterInstallation(ctx context.Context, pluginID vtypes.PluginID, publicKey string) error {
	return s.db.WithTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		pluginInfo, err := s.db.FindPluginById(ctx, tx, pluginID)
		if err != nil {
			return err
		}

		err = s.db.InsertPluginInstallation(ctx, tx, pluginID, publicKey)
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

		_, err = s.db.InsertFee(ctx, tx, &vtypes.Fee{
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
