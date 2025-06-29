package service

import (
	"context"
	"fmt"
	"plugin"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/storage"
	"github.com/vultisig/verifier/internal/syncer"
	"github.com/vultisig/verifier/internal/tasks"
	"github.com/vultisig/verifier/internal/types"
)

type WorkerService struct {
	cfg          config.WorkerConfig
	verifierPort int64
	redis        *storage.RedisStorage
	logger       *logrus.Logger
	queueClient  *asynq.Client
	sdClient     *statsd.Client

	inspector   *asynq.Inspector
	plugin      plugin.Plugin
	db          storage.DatabaseStorage
	authService *AuthService
}

// NewWorker creates a new worker service
func NewWorker(cfg config.WorkerConfig,
	verifierPort int64,
	queueClient *asynq.Client,
	sdClient *statsd.Client, authService *AuthService,
	inspector *asynq.Inspector) (*WorkerService, error) {
	logger := logrus.WithField("service", "worker").Logger

	redis, err := storage.NewRedisStorage(cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("storage.NewRedisStorage failed: %w", err)
	}

	return &WorkerService{
		cfg:          cfg,
		redis:        redis,
		queueClient:  queueClient,
		sdClient:     sdClient,
		inspector:    inspector,
		logger:       logger,
		authService:  authService,
		verifierPort: verifierPort,
	}, nil
}

type KeyGenerationTaskResult struct {
	EDDSAPublicKey string
	ECDSAPublicKey string
}

func (s *WorkerService) upsertAndSyncTransaction(ctx context.Context, action syncer.Action, tx *types.TransactionHistory, jwtToken string) error {
	s.logger.Info("upsertAndSyncTransaction started with action: ", action)
	dbTx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer dbTx.Rollback(ctx)

	if action == syncer.CreateAction {
		txID, err := s.db.CreateTransactionHistoryTx(ctx, dbTx, *tx)
		if err != nil {
			s.logger.Errorf("Failed to create (or update) transaction history tx: %v", err)
			return fmt.Errorf("failed to create transaction history: %w", err)
		}
		tx.ID = txID
	} else {
		if err = s.db.UpdateTransactionStatusTx(ctx, dbTx, tx.ID, tx.Status, tx.Metadata); err != nil {
			return fmt.Errorf("failed to update transaction status: %w", err)
		}
	}

	if err = dbTx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (s *WorkerService) waitForTaskResult(taskID string, timeout time.Duration) ([]byte, error) {
	start := time.Now()
	pollInterval := time.Second

	for {
		if time.Since(start) > timeout {
			return nil, fmt.Errorf("timeout waiting for task result after %v", timeout)
		}

		task, err := s.inspector.GetTaskInfo(tasks.QUEUE_NAME, taskID)
		if err != nil {
			return nil, fmt.Errorf("failed to get task info: %w", err)
		}

		switch task.State {
		case asynq.TaskStateCompleted:
			s.logger.Info("Task completed successfully")
			return task.Result, nil
		case asynq.TaskStateArchived:
			return nil, fmt.Errorf("task archived: %s", task.LastErr)
		case asynq.TaskStateRetry:
			s.logger.Debug("Task scheduled for retry...")
		case asynq.TaskStatePending, asynq.TaskStateActive, asynq.TaskStateScheduled:
			s.logger.Debug("Task still in progress, waiting...")
		case asynq.TaskStateAggregating:
			s.logger.Debug("Task aggregating, waiting...")
		default:
			return nil, fmt.Errorf("unexpected task state: %s", task.State)
		}

		time.Sleep(pollInterval)
	}
}
