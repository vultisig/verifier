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
	"github.com/vultisig/vultiserver/contexthelper"
	"github.com/vultisig/vultiserver/relay"

	"github.com/vultisig/verifier/internal/storage"
	"github.com/vultisig/verifier/types"
)

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
	verifierPort int64
	redis        *storage.RedisStorage
	logger       *logrus.Logger
	queueClient  *asynq.Client
	sdClient     *statsd.Client
	blockStorage *storage.BlockStorage
	inspector    *asynq.Inspector
	plugin       plugin.Plugin
	db           storage.DatabaseStorage
}

// NewManagementService creates a new instance of the ManagementService
func NewManagementService(cfg Config,
	verifierPort int64,
	queueClient *asynq.Client,
	sdClient *statsd.Client,
	blockStorage *storage.BlockStorage, inspector *asynq.Inspector) (*ManagementService, error) {
	logger := logrus.WithField("service", "worker").Logger

	// redis, err := storage.NewRedisStorage(cfg)
	// if err != nil {
	// 	return nil, fmt.Errorf("storage.NewRedisStorage failed: %w", err)
	// }
	//
	// db, err := postgres.NewPostgresBackend(false, cfg.Server.Database.DSN)
	// if err != nil {
	// 	return nil, fmt.Errorf("fail to connect to database: %w", err)
	// }

	return &ManagementService{
		cfg:          cfg,
		blockStorage: blockStorage,
		queueClient:  queueClient,
		sdClient:     sdClient,
		inspector:    inspector,
		logger:       logger,
		verifierPort: verifierPort,
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
	if req.LibType != types.DKLS {
		return fmt.Errorf("invalid lib type: %d: %w", req.LibType, asynq.SkipRetry)
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
	localStateAccessor, err := relay.NewLocalStateAccessorImp(s.cfg.Server.VaultsFilePath, "", "", s.blockStorage)
	if err != nil {
		return fmt.Errorf("relay.NewLocalStateAccessorImp failed: %s: %w", err, asynq.SkipRetry)
	}
	dklsService, err := NewDKLSTssService(s.cfg, s.blockStorage, localStateAccessor, s)
	if err != nil {
		return fmt.Errorf("NewDKLSTssService failed: %s: %w", err, asynq.SkipRetry)
	}
	keyECDSA, keyEDDSA, err := dklsService.ProceeDKLSKeygen(req)
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
