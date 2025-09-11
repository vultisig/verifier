package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"plugin"
	"time"

	"github.com/google/uuid"
	"github.com/vultisig/verifier/plugin/tx_indexer"
	"github.com/vultisig/verifier/vault_config"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	keygenType "github.com/vultisig/commondata/go/vultisig/keygen/v1"
	vaultType "github.com/vultisig/commondata/go/vultisig/vault/v1"
	"github.com/vultisig/vultiserver/contexthelper"

	vtypes "github.com/vultisig/verifier/types"
	vcommon "github.com/vultisig/vultisig-go/common"
	vgtypes "github.com/vultisig/vultisig-go/types"
)

const EmailVaultBackupTypeName = "key:email"
const EmailQueueName = "vault:email_queue"

// KeyGenerationTaskResult is a struct that represents the result of a key generation task
type KeyGenerationTaskResult struct {
	EDDSAPublicKey string
	ECDSAPublicKey string
}

// ManagementService is a struct that represents the vault management service
// it provides the following capatilities
// - Keygen -- create vault / reshare vault
// - Keysign -- sign a message
type ManagementService struct {
	cfg              vault_config.Config
	logger           *logrus.Logger
	queueClient      *asynq.Client
	sdClient         *statsd.Client
	plugin           plugin.Plugin
	vaultStorage     Storage
	txIndexerService *tx_indexer.Service
}

// NewManagementService creates a new instance of the ManagementService
func NewManagementService(
	cfg vault_config.Config,
	queueClient *asynq.Client,
	sdClient *statsd.Client,
	storage Storage,
	txIndexerService *tx_indexer.Service,
) (*ManagementService, error) {
	logger := logrus.WithField("service", "vault-management").Logger

	return &ManagementService{
		cfg:              cfg,
		queueClient:      queueClient,
		sdClient:         sdClient,
		logger:           logger,
		vaultStorage:     storage,
		txIndexerService: txIndexerService,
	}, nil
}

func (s *ManagementService) incCounter(name string, tags []string) {
	if err := s.sdClient.Count(name, 1, tags, 1); err != nil {
		s.logger.WithError(err).Error("fail to count metric")
	}
}

func (s *ManagementService) measureTime(name string, start time.Time, tags []string) {
	if err := s.sdClient.Timing(name, time.Since(start), tags, 1); err != nil {
		s.logger.WithError(err).Error("fail to measure time metric")
	}
}

func (s *ManagementService) HandleKeyGenerationDKLS(ctx context.Context, t *asynq.Task) error {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return err
	}
	defer s.measureTime("worker.vault.create.latency", time.Now(), []string{})
	var req vgtypes.VaultCreateRequest
	if err := json.Unmarshal(t.Payload(), &req); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	s.logger.WithFields(logrus.Fields{
		"name":           req.Name,
		"session":        req.SessionID,
		"local_party_id": req.LocalPartyId,
		"email":          req.Email,
	}).Info("Joining keygen")
	s.incCounter("worker.vault.create.dkls", []string{})
	if err := req.IsValid(); err != nil {
		return fmt.Errorf("invalid vault create request: %s: %w", err, asynq.SkipRetry)
	}

	dklsService, err := NewDKLSTssService(s.cfg, s.vaultStorage, s.queueClient)
	if err != nil {
		return fmt.Errorf("NewDKLSTssService failed: %s: %w", err, asynq.SkipRetry)
	}
	keyECDSA, keyEDDSA, err := dklsService.ProcessDKLSKeygen(req)
	if err != nil {
		_ = s.sdClient.Count("worker.vault.create.dkls.error", 1, nil, 1)
		s.logger.WithError(err).Error("keygen.JoinKeyGeneration failed")
		return fmt.Errorf("keygen.JoinKeyGeneration failed: %v: %w", err, asynq.SkipRetry)
	}

	s.logger.WithFields(logrus.Fields{
		"keyECDSA": keyECDSA,
		"keyEDDSA": keyEDDSA,
	}).Info("localPartyID generation completed")

	result := KeyGenerationTaskResult{
		EDDSAPublicKey: keyEDDSA,
		ECDSAPublicKey: keyECDSA,
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		s.logger.WithError(err).Error("json.Marshal failed")
		return fmt.Errorf("json.Marshal failed: %v: %w", err, asynq.SkipRetry)
	}

	if _, err := t.ResultWriter().Write(resultBytes); err != nil {
		s.logger.WithError(err).Error("t.ResultWriter.Write failed")
		return fmt.Errorf("t.ResultWriter.Write failed: %v: %w", err, asynq.SkipRetry)
	}

	return nil
}

func (s *ManagementService) HandleKeySignDKLS(ctx context.Context, t *asynq.Task) error {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return err
	}
	var p vtypes.KeysignRequest
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		s.logger.WithError(err).Error("json.Unmarshal failed")
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	defer s.measureTime("worker.vault.sign.latency", time.Now(), []string{})
	s.incCounter("worker.vault.sign", []string{})
	s.logger.WithFields(logrus.Fields{
		"PublicKey": p.PublicKey,
		"session":   p.SessionID,
		"Messages":  len(p.Messages),
		"PluginID":  p.PluginID,
		"PolicyID":  p.PolicyID,
	}).Info("joining keysign")

	dklsService, err := NewDKLSTssService(s.cfg, s.vaultStorage, s.queueClient)
	if err != nil {
		return fmt.Errorf("NewDKLSTssService failed: %s: %w", err, asynq.SkipRetry)
	}

	signatures, err := dklsService.ProcessDKLSKeysign(p)
	if err != nil {
		s.logger.WithError(err).Error("join keysign failed")
		return fmt.Errorf("join keysign failed: %v: %w", err, asynq.SkipRetry)
	}

	s.logger.WithFields(logrus.Fields{
		"Signatures": signatures,
	}).Info("localPartyID sign completed")

	resultBytes, err := json.Marshal(signatures)
	if err != nil {
		s.logger.WithError(err).Error("json.Marshal failed")
		return fmt.Errorf("json.Marshal failed: %v: %w", err, asynq.SkipRetry)
	}

	if _, err := t.ResultWriter().Write(resultBytes); err != nil {
		s.logger.WithError(err).Error("t.ResultWriter.Write failed")
		return fmt.Errorf("t.ResultWriter.Write failed: %v: %w", err, asynq.SkipRetry)
	}

	for _, msg := range p.Messages {
		if msg.TxIndexerID == "" {
			continue // not from plugin
		}

		txID, er := uuid.Parse(msg.TxIndexerID)
		if er != nil {
			s.logger.WithError(er).Error("uuid.Parse(reqPlugin.TxIndexerID)")
			return fmt.Errorf("uuid.Parse(reqPlugin.TxIndexerID): %v: %w", er, asynq.SkipRetry)
		}

		er = s.txIndexerService.SetSignedAndBroadcasted(
			ctx,
			msg.Chain,
			txID,
			signatures,
		)
		if er != nil {
			s.logger.WithError(er).Error("s.txIndexerService.SetSignedAndBroadcasted")
			return fmt.Errorf("s.txIndexerService.SetSignedAndBroadcasted: %v: %w", er, asynq.SkipRetry)
		}
	}

	return nil
}

func (s *ManagementService) HandleReshareDKLS(ctx context.Context, t *asynq.Task) error {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return err
	}
	var req vtypes.ReshareRequest
	if err := json.Unmarshal(t.Payload(), &req); err != nil {
		s.logger.WithError(err).Error("json.Unmarshal failed")
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	defer s.measureTime("worker.vault.reshare.latency", time.Now(), []string{})
	s.incCounter("worker.vault.reshare.dkls", []string{})
	s.logger.WithFields(logrus.Fields{
		"name":           req.Name,
		"session":        req.SessionID,
		"local_party_id": req.LocalPartyId,
		"email":          req.Email,
	}).Info("reshare request")
	if err := req.IsValid(); err != nil {
		return fmt.Errorf("invalid reshare request: %s: %w", err, asynq.SkipRetry)
	}

	var vault *vaultType.Vault
	// trying to get existing vault
	vaultFileName := vcommon.GetVaultBackupFilename(req.PublicKey, req.PluginID)
	vaultContent, err := s.vaultStorage.GetVault(vaultFileName)
	if err != nil || vaultContent == nil {
		vault = &vaultType.Vault{
			Name:           req.Name,
			PublicKeyEcdsa: "",
			PublicKeyEddsa: "",
			HexChainCode:   req.HexChainCode,
			LocalPartyId:   vcommon.GenerateLocalPartyId(s.cfg.LocalPartyPrefix),
			Signers:        req.OldParties,
			LibType:        keygenType.LibType_LIB_TYPE_DKLS,
		}
	} else {
		return fmt.Errorf("plugin (%s) has been installed to vault (%s), err:%w", req.PluginID, req.PublicKey, asynq.SkipRetry)
	}

	service, err := NewDKLSTssService(s.cfg, s.vaultStorage, s.queueClient)
	if err != nil {
		s.logger.WithError(err).Error("NewDKLSTssService failed")
		return fmt.Errorf("NewDKLSTssService failed: %v: %w", err, asynq.SkipRetry)
	}

	if err := service.ProcessReshare(vault, req.SessionID, req.HexEncryptionKey, req.Email, req.PluginID); err != nil {
		s.logger.WithError(err).Error("reshare failed")
		return fmt.Errorf("reshare failed: %v: %w", err, asynq.SkipRetry)
	}

	return nil
}
