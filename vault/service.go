package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"plugin"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	keygenType "github.com/vultisig/commondata/go/vultisig/keygen/v1"
	vaultType "github.com/vultisig/commondata/go/vultisig/vault/v1"
	"github.com/vultisig/vultiserver/common"
	"github.com/vultisig/vultiserver/contexthelper"

	"github.com/vultisig/verifier/types"
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
	cfg          Config
	logger       *logrus.Logger
	queueClient  *asynq.Client
	sdClient     *statsd.Client
	plugin       plugin.Plugin
	vaultStorage Storage
}

// NewManagementService creates a new instance of the ManagementService
func NewManagementService(cfg Config,
	queueClient *asynq.Client,
	sdClient *statsd.Client,
	storage Storage) (*ManagementService, error) {
	logger := logrus.WithField("service", "vault-management").Logger

	return &ManagementService{
		cfg:          cfg,
		queueClient:  queueClient,
		sdClient:     sdClient,
		logger:       logger,
		vaultStorage: storage,
	}, nil
}

func (s *ManagementService) incCounter(name string, tags []string) {
	if err := s.sdClient.Count(name, 1, tags, 1); err != nil {
		s.logger.Errorf("fail to count metric, err: %v", err)
	}
}

func (s *ManagementService) measureTime(name string, start time.Time, tags []string) {
	if err := s.sdClient.Timing(name, time.Since(start), tags, 1); err != nil {
		s.logger.Errorf("fail to measure time metric, err: %v", err)
	}
}

func (s *ManagementService) HandleKeyGenerationDKLS(ctx context.Context, t *asynq.Task) error {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return err
	}
	defer s.measureTime("worker.vault.create.latency", time.Now(), []string{})
	var req types.VaultCreateRequest
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
		s.logger.Errorf("keygen.JoinKeyGeneration failed: %v", err)
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
		s.logger.Errorf("json.Marshal failed: %v", err)
		return fmt.Errorf("json.Marshal failed: %v: %w", err, asynq.SkipRetry)
	}

	if _, err := t.ResultWriter().Write(resultBytes); err != nil {
		s.logger.Errorf("t.ResultWriter.Write failed: %v", err)
		return fmt.Errorf("t.ResultWriter.Write failed: %v: %w", err, asynq.SkipRetry)
	}

	return nil
}

func (s *ManagementService) HandleKeySignDKLS(ctx context.Context, t *asynq.Task) error {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return err
	}
	var p types.KeysignRequest
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		s.logger.Errorf("json.Unmarshal failed: %v", err)
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	defer s.measureTime("worker.vault.sign.latency", time.Now(), []string{})
	s.incCounter("worker.vault.sign", []string{})
	s.logger.WithFields(logrus.Fields{
		"PublicKey":  p.PublicKey,
		"session":    p.SessionID,
		"Messages":   p.Messages,
		"DerivePath": p.DerivePath,
		"IsECDSA":    p.IsECDSA,
	}).Info("joining keysign")

	dklsService, err := NewDKLSTssService(s.cfg, s.vaultStorage, s.queueClient)
	if err != nil {
		return fmt.Errorf("NewDKLSTssService failed: %s: %w", err, asynq.SkipRetry)
	}

	signatures, err := dklsService.ProcessDKLSKeysign(p)
	if err != nil {
		s.logger.Errorf("join keysign failed: %v", err)
		return fmt.Errorf("join keysign failed: %v: %w", err, asynq.SkipRetry)
	}

	s.logger.WithFields(logrus.Fields{
		"Signatures": signatures,
	}).Info("localPartyID sign completed")

	resultBytes, err := json.Marshal(signatures)
	if err != nil {
		s.logger.Errorf("json.Marshal failed: %v", err)
		return fmt.Errorf("json.Marshal failed: %v: %w", err, asynq.SkipRetry)
	}

	if _, err := t.ResultWriter().Write(resultBytes); err != nil {
		s.logger.Errorf("t.ResultWriter.Write failed: %v", err)
		return fmt.Errorf("t.ResultWriter.Write failed: %v: %w", err, asynq.SkipRetry)
	}

	return nil
}

func (s *ManagementService) HandleReshareDKLS(ctx context.Context, t *asynq.Task) error {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return err
	}
	var req types.ReshareRequest
	if err := json.Unmarshal(t.Payload(), &req); err != nil {
		s.logger.Errorf("json.Unmarshal failed: %v", err)
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
	vaultFileName := req.PublicKey + ".bak"
	vaultContent, err := s.vaultStorage.GetVault(vaultFileName)
	if err != nil || vaultContent == nil {
		vault = &vaultType.Vault{
			Name:           req.Name,
			PublicKeyEcdsa: "",
			PublicKeyEddsa: "",
			HexChainCode:   req.HexChainCode,
			LocalPartyId:   req.LocalPartyId,
			Signers:        req.OldParties,
			ResharePrefix:  req.OldResharePrefix,
			LibType:        keygenType.LibType_LIB_TYPE_DKLS,
		}
	} else {
		// decrypt the vault
		vault, err = common.DecryptVaultFromBackup(s.cfg.EncryptionSecret, vaultContent)
		if err != nil {
			s.logger.Errorf("fail to decrypt vault from the backup, err: %v", err)
			return fmt.Errorf("fail to decrypt vault from the backup, err: %v: %w", err, asynq.SkipRetry)
		}
	}

	service, err := NewDKLSTssService(s.cfg, s.vaultStorage, s.queueClient)
	if err != nil {
		s.logger.Errorf("NewDKLSTssService failed: %v", err)
		return fmt.Errorf("NewDKLSTssService failed: %v: %w", err, asynq.SkipRetry)
	}

	if err := service.ProcessReshare(vault, req.SessionID, req.HexEncryptionKey, req.Email); err != nil {
		s.logger.Errorf("reshare failed: %v", err)
		return fmt.Errorf("reshare failed: %v: %w", err, asynq.SkipRetry)
	}

	return nil
}
