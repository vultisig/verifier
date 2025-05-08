package syncer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/vultiserver/contexthelper"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/vultisig/verifier/internal/storage"
	itypes "github.com/vultisig/verifier/internal/types"
)

const (
	defaultTimeout = 10 * time.Second
	policyEndpoint = "/plugin/policy"

	// Retry configuration
	maxRetries        = 3
	QUEUE_NAME        = "policy-syncer"
	TaskKeySyncPolicy = "sync_policy"
)

type Action int

const (
	CreateAction Action = iota
	UpdateAction
)

type Syncer struct {
	logger      *logrus.Logger
	client      *retryablehttp.Client
	storage     storage.DatabaseStorage
	asynqClient *asynq.Client

	cacheLocker              sync.Locker
	pluginServerAddressCache map[uuid.UUID]string
	wg                       *sync.WaitGroup
	stopCh                   chan struct{}
}

func NewPolicySyncer(storage storage.DatabaseStorage, client *asynq.Client) *Syncer {
	logger := logrus.WithField("component", "policy-syncer").Logger
	retryClient := retryablehttp.NewClient()
	retryClient.HTTPClient.Timeout = defaultTimeout
	retryClient.Logger = logger
	retryClient.RetryMax = maxRetries
	return &Syncer{
		logger:                   logger,
		client:                   retryClient,
		storage:                  storage,
		asynqClient:              client,
		pluginServerAddressCache: make(map[uuid.UUID]string),
		cacheLocker:              &sync.Mutex{},
		wg:                       &sync.WaitGroup{},
		stopCh:                   make(chan struct{}),
	}
}

func (s *Syncer) Start() error {
	s.wg.Add(1)
	go s.processFailedTasks()
	return nil
}

func (s *Syncer) processFailedTasks() {
	defer s.wg.Done()
	s.logger.Info("starting processing failed tasks")
	for {
		select {
		case <-s.stopCh:
			return
		case <-time.After(30 * time.Minute):
			syncTasks, err := s.storage.GetUnFinishedPluginPolicySyncs(context.Background())
			if err != nil {
				s.logger.Errorf("failed to get unfinished plugin policy syncs: %v", err)
				continue
			}
			for _, syncTask := range syncTasks {
				payload, err := json.Marshal(syncTask)
				if err != nil {
					s.logger.Errorf("failed to marshal sync task: %v", err)
					continue
				}
				ti, err := s.asynqClient.Enqueue(
					asynq.NewTask(TaskKeySyncPolicy, payload),
					asynq.Queue(QUEUE_NAME),
					asynq.MaxRetry(3),
				)
				if err != nil {
					s.logger.Errorf("failed to enqueue task: %v", err)
					continue
				}
				s.logger.WithField("task_id", ti.ID).Info("enqueued sync policy task")
			}
		}
	}

}

func (s *Syncer) Stop() error {
	close(s.stopCh)
	s.wg.Wait()
	return nil
}

func (s *Syncer) ProcessSyncTask(ctx context.Context, task *asynq.Task) error {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return err
	}
	var req itypes.PluginPolicySync
	if err := json.Unmarshal(task.Payload(), &req); err != nil {
		s.logger.Errorf("failed to unmarshal payload: %v", err)
		return fmt.Errorf("failed to unmarshal task payload: %s: %w", err, asynq.SkipRetry)
	}
	switch req.SyncType {
	case itypes.AddPolicy, itypes.UpdatePolicy:
		if err := s.createPolicySync(ctx, req); err != nil {
			s.logger.Errorf("failed to create policy sync: %v", err)
			return fmt.Errorf("failed to create policy sync: %w", err)
		}
	case itypes.RemovePolicy:
		if err := s.DeletePolicySync(ctx, req); err != nil {
			s.logger.Errorf("failed to delete policy sync: %v", err)
			return fmt.Errorf("failed to delete policy sync: %w", err)
		}
	}
	return nil
}

func (s *Syncer) getServerAddr(ctx context.Context, pluginID uuid.UUID) (string, error) {
	s.cacheLocker.Lock()
	defer s.cacheLocker.Unlock()

	if addr, ok := s.pluginServerAddressCache[pluginID]; ok {
		return addr, nil
	}

	addr, err := s.getServerAddrFromStorage(ctx, pluginID)
	if err != nil {
		return "", fmt.Errorf("failed to get server address from storage: %w", err)
	}
	s.pluginServerAddressCache[pluginID] = addr
	return addr, nil
}

func (s *Syncer) getServerAddrFromStorage(ctx context.Context, pluginID uuid.UUID) (string, error) {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return "", err
	}
	s.logger.Infof("pluginid: %s", pluginID.String())
	plugin, err := s.storage.FindPluginById(ctx, pluginID)
	if err != nil {
		return "", fmt.Errorf("failed to find plugin by id: %w", err)
	}
	return plugin.ServerEndpoint, nil
}

func (s *Syncer) createPolicySync(ctx context.Context, policySyncEntity itypes.PluginPolicySync) error {
	s.logger.WithFields(logrus.Fields{
		"policy_id": policySyncEntity.PolicyID,
		"sync_id":   policySyncEntity.ID,
	}).Info("Starting policy creation sync")
	policy, err := s.storage.GetPluginPolicy(ctx, policySyncEntity.PolicyID)
	if err != nil {
		return fmt.Errorf("failed to get policy for sync: %w", err)
	}

	policyBytes, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("fail to marshal policy: %v,err: %w", err, asynq.SkipRetry)
	}
	defer func() {
		if err := s.updatePolicySyncStatus(ctx, policySyncEntity); err != nil {
			s.logger.Errorf("failed to update policy sync status: %v", err)
		}
	}()
	serverEndpoint, err := s.getServerAddr(ctx, policySyncEntity.PluginID)
	if err != nil {
		policySyncEntity.Status = itypes.Failed
		policySyncEntity.FailReason = fmt.Sprintf("failed to get server address: %w", err)
		return fmt.Errorf("failed to get server address: %w", err)
	}

	url := serverEndpoint + policyEndpoint
	resp, err := s.client.Post(url, "application/json", bytes.NewBuffer(policyBytes))
	if err != nil {
		policySyncEntity.Status = itypes.Failed
		policySyncEntity.FailReason = fmt.Sprintf("failed to sync policy with plugin server(%s): %s", url, err.Error())
		return fmt.Errorf("fail to sync policy with verifier server: %w", err)
	}
	defer s.closer(resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		policySyncEntity.Status = itypes.Failed
		policySyncEntity.FailReason = fmt.Sprintf("failed to sync policy with plugin server(%s): %s", url, err.Error())
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		s.logger.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"body":        string(body),
			"policy_id":   policy.ID,
		}).Error("Failed to sync create policy")
		policySyncEntity.Status = itypes.Failed
		policySyncEntity.FailReason = fmt.Sprintf("failed to sync policy with plugin server(%s): status: %d, body: %s", url, resp.StatusCode, string(body))

		return fmt.Errorf("fail to sync policy with verifier server: status: %d", resp.StatusCode)
	}
	policySyncEntity.Status = itypes.Synced

	s.logger.WithFields(logrus.Fields{
		"policy_id": policy.ID,
	}).Info("sync successfully")

	return nil

}
func (s *Syncer) updatePolicySyncStatus(ctx context.Context, policySyncEntity itypes.PluginPolicySync) (returnErr error) {
	// update plugin policy sync status
	tx, err := s.storage.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Commit(ctx); err != nil && returnErr == nil {
			returnErr = fmt.Errorf("failed to commit transaction: %w", err)
		}
	}()
	if err := s.storage.UpdatePluginPolicySync(ctx, tx, policySyncEntity); err != nil {
		return fmt.Errorf("failed to update policy sync status: %w", err)
	}
	return nil
}

func (s *Syncer) closer(c io.Closer) {
	if err := c.Close(); err != nil {
		s.logger.Errorf("failed to close io.Closer: %s", err)
	}
}

type DeleteRequestBody struct {
	Signature string `json:"signature"`
}

func (s *Syncer) DeletePolicySync(ctx context.Context, syncEntity itypes.PluginPolicySync) error {
	s.logger.WithFields(logrus.Fields{
		"sync_id":   syncEntity.ID,
		"policy_id": syncEntity.PolicyID,
	}).Info("Starting policy delete sync")

	reqBody := DeleteRequestBody{
		Signature: syncEntity.Signature,
	}
	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("fail to marshal request body: %v,err: %w", err, asynq.SkipRetry)
	}
	defer func() {
		if err := s.updatePolicySyncStatus(ctx, syncEntity); err != nil {
			s.logger.Errorf("failed to update policy sync status: %v", err)
		}
	}()
	serverEndpoint, err := s.getServerAddr(ctx, syncEntity.PluginID)
	if err != nil {
		syncEntity.Status = itypes.Failed
		syncEntity.FailReason = fmt.Sprintf("failed to get server address: %w", err)
		return fmt.Errorf("failed to get server address: %w", err)
	}

	url := serverEndpoint + policyEndpoint + "/" + syncEntity.PolicyID.String()
	req, err := retryablehttp.NewRequest(http.MethodDelete, url, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		syncEntity.Status = itypes.Failed
		syncEntity.FailReason = fmt.Sprintf("failed to create request: %w", err)
		return fmt.Errorf("fail to create request, err: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		syncEntity.Status = itypes.Failed
		syncEntity.FailReason = fmt.Sprintf("failed to delete policy with plugin server(%s): %s", url, err.Error())
		return fmt.Errorf("fail to delete policy on verifier server, err: %w", err)
	}
	defer s.closer(resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		syncEntity.Status = itypes.Failed
		syncEntity.FailReason = fmt.Sprintf("failed to read response body: %w", err)
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		s.logger.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"body":        string(body),
			"policy_id":   syncEntity.PolicyID,
		}).Error("Failed to sync delete policy")
		syncEntity.Status = itypes.Failed
		syncEntity.FailReason = fmt.Sprintf("failed to sync delete policy with plugin server(%s): status: %d, body: %s", url, resp.StatusCode, string(body))
		return fmt.Errorf("fail to delete policy on verifier server, status: %d", resp.StatusCode)
	}
	syncEntity.Status = itypes.Synced
	syncEntity.FailReason = ""

	s.logger.WithFields(logrus.Fields{
		"policy_id": syncEntity.PolicyID,
	}).Info("Successfully sync deleted policy")

	return nil

}
