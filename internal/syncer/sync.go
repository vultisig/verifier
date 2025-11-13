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

	"github.com/sirupsen/logrus"
	"github.com/vultisig/vultiserver/contexthelper"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/vultisig/verifier/internal/storage"
	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/types"
	ptypes "github.com/vultisig/verifier/types"
)

const (
	defaultTimeout = 10 * time.Second
	policyEndpoint = "/plugin/policy"

	// Retry configuration
	maxRetries = 3
)

type Action int

const (
	CreateAction Action = iota
	UpdateAction
)

type Syncer struct {
	logger  *logrus.Logger
	client  *retryablehttp.Client
	storage storage.DatabaseStorage

	cacheLocker              sync.Locker
	pluginServerAddressCache map[ptypes.PluginID]string
}

func NewPolicySyncer(storage storage.DatabaseStorage) *Syncer {
	logger := logrus.WithField("component", "policy-syncer").Logger
	retryClient := retryablehttp.NewClient()
	retryClient.HTTPClient.Timeout = defaultTimeout
	retryClient.Logger = logger
	retryClient.RetryMax = maxRetries
	return &Syncer{
		logger:                   logger,
		client:                   retryClient,
		storage:                  storage,
		pluginServerAddressCache: make(map[ptypes.PluginID]string),
		cacheLocker:              &sync.Mutex{},
	}
}

func (s *Syncer) getServerAddr(ctx context.Context, pluginID ptypes.PluginID) (string, error) {
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

func (s *Syncer) getServerAddrFromStorage(ctx context.Context, pluginID ptypes.PluginID) (string, error) {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return "", err
	}
	s.logger.Infof("pluginid: %s", pluginID.String())
	plugin, err := s.storage.FindPluginById(ctx, nil, pluginID)
	if err != nil {
		return "", fmt.Errorf("failed to find plugin by id: %w", err)
	}
	return plugin.ServerEndpoint, nil
}

// Creates a policy in a synchronous manner.
func (s *Syncer) CreatePolicySync(ctx context.Context, pluginPolicy types.PluginPolicy) error {
	policyBytes, err := json.Marshal(pluginPolicy)
	if err != nil {
		return fmt.Errorf("fail to marshal policy: %v", err)
	}
	serverEndpoint, err := s.getServerAddr(ctx, pluginPolicy.PluginID)
	if err != nil {
		return fmt.Errorf("failed to get server address: %w", err)
	}

	url := serverEndpoint + policyEndpoint
	resp, err := s.client.Post(url, "application/json", bytes.NewBuffer(policyBytes))
	if err != nil {
		return fmt.Errorf("failed to sync policy with plugin server(%s): %s", url, err.Error())
	}
	defer s.closer(resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to sync policy with plugin server(%s): status: %d, body: %s", url, resp.StatusCode, string(body))
	}
	return nil
}

// Deprecated: use CreatePolicySync instead.
func (s *Syncer) CreatePolicyAsync(ctx context.Context, policySyncEntity itypes.PluginPolicySync) error {
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
		return fmt.Errorf("fail to marshal policy: %v", err)
	}
	serverEndpoint, err := s.getServerAddr(ctx, policySyncEntity.PluginID)
	if err != nil {
		policySyncEntity.Status = itypes.Failed
		policySyncEntity.FailReason = fmt.Sprintf("failed to get server address: %s", err)
		// Update status in database before returning error
		if updateErr := s.updatePolicySyncStatus(ctx, policySyncEntity); updateErr != nil {
			s.logger.Errorf("failed to update policy sync status: %v", updateErr)
		}
		return fmt.Errorf("failed to get server address: %w", err)
	}

	url := serverEndpoint + policyEndpoint
	resp, err := s.client.Post(url, "application/json", bytes.NewBuffer(policyBytes))
	if err != nil {
		policySyncEntity.Status = itypes.Failed
		policySyncEntity.FailReason = fmt.Sprintf("failed to sync policy with plugin server(%s): %s", url, err.Error())
		// Update status in database before returning error
		if updateErr := s.updatePolicySyncStatus(ctx, policySyncEntity); updateErr != nil {
			s.logger.Errorf("failed to update policy sync status: %v", updateErr)
		}
		return fmt.Errorf("fail to sync policy with verifier server: %w", err)
	}
	defer s.closer(resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		policySyncEntity.Status = itypes.Failed
		policySyncEntity.FailReason = fmt.Sprintf("failed to sync policy with plugin server(%s): %s", url, err.Error())
		// Update status in database before returning error
		if updateErr := s.updatePolicySyncStatus(ctx, policySyncEntity); updateErr != nil {
			s.logger.Errorf("failed to update policy sync status: %v", updateErr)
		}
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

		// Update status in database before returning error
		if updateErr := s.updatePolicySyncStatus(ctx, policySyncEntity); updateErr != nil {
			s.logger.Errorf("failed to update policy sync status: %v", updateErr)
		}

		return fmt.Errorf("fail to sync policy with verifier server: status: %d", resp.StatusCode)
	}

	// Success case
	policySyncEntity.Status = itypes.Synced
	policySyncEntity.FailReason = ""

	// Update status in database
	if err := s.updatePolicySyncStatus(ctx, policySyncEntity); err != nil {
		s.logger.Errorf("failed to update policy sync status: %v", err)
	}

	s.logger.WithFields(logrus.Fields{
		"policy_id": policy.ID,
	}).Info("sync successfully")

	return nil
}
func (s *Syncer) updatePolicySyncStatus(ctx context.Context, policySyncEntity itypes.PluginPolicySync) error {
	// update plugin policy sync status
	tx, err := s.storage.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		err := tx.Rollback(ctx)
		if err != nil {
			s.logger.WithError(err).Error("failed to rollback transaction")
		}
	}()

	err = s.storage.UpdatePluginPolicySync(ctx, tx, policySyncEntity)
	if err != nil {
		return fmt.Errorf("failed to update policy sync status: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
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

// Deletes a policy in a synchronous manner.
func (s *Syncer) DeletePolicySync(ctx context.Context, pluginPolicy types.PluginPolicy) error {
	reqBody := DeleteRequestBody{
		Signature: pluginPolicy.Signature,
	}
	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %v", err)
	}

	serverEndpoint, err := s.getServerAddr(ctx, pluginPolicy.PluginID)
	if err != nil {
		return fmt.Errorf("failed to get server address: %w", err)
	}

	url := serverEndpoint + policyEndpoint + "/" + pluginPolicy.ID.String()
	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete policy with plugin server(%s): %s", url, err.Error())
	}
	defer s.closer(resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete policy with plugin server(%s): status: %d, body: %s", url, resp.StatusCode, string(body))
	}
	return nil
}

// Deprecated: use DeletePolicySync instead.
func (s *Syncer) DeletePolicyAsync(ctx context.Context, syncEntity itypes.PluginPolicySync) error {
	s.logger.WithFields(logrus.Fields{
		"sync_id":   syncEntity.ID,
		"policy_id": syncEntity.PolicyID,
	}).Info("Starting policy delete sync")

	reqBody := DeleteRequestBody{
		Signature: syncEntity.Signature,
	}
	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		syncEntity.Status = itypes.Failed
		syncEntity.FailReason = fmt.Sprintf("failed to marshal request body: %v", err)
		// Update status in database before returning error
		if updateErr := s.updatePolicySyncStatus(ctx, syncEntity); updateErr != nil {
			s.logger.Errorf("failed to update policy sync status: %v", updateErr)
		}
		return fmt.Errorf("fail to marshal request body: %v", err)
	}

	serverEndpoint, err := s.getServerAddr(ctx, syncEntity.PluginID)
	if err != nil {
		syncEntity.Status = itypes.Failed
		syncEntity.FailReason = fmt.Sprintf("failed to get server address: %s", err)
		// Update status in database before returning error
		if updateErr := s.updatePolicySyncStatus(ctx, syncEntity); updateErr != nil {
			s.logger.Errorf("failed to update policy sync status: %v", updateErr)
		}
		return fmt.Errorf("failed to get server address: %w", err)
	}

	url := serverEndpoint + policyEndpoint + "/" + syncEntity.PolicyID.String()
	req, err := retryablehttp.NewRequest(http.MethodDelete, url, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		syncEntity.Status = itypes.Failed
		syncEntity.FailReason = fmt.Sprintf("failed to create request: %s", err)
		// Update status in database before returning error
		if updateErr := s.updatePolicySyncStatus(ctx, syncEntity); updateErr != nil {
			s.logger.Errorf("failed to update policy sync status: %v", updateErr)
		}
		return fmt.Errorf("fail to create request, err: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		syncEntity.Status = itypes.Failed
		syncEntity.FailReason = fmt.Sprintf("failed to delete policy with plugin server(%s): %s", url, err.Error())
		// Update status in database before returning error
		if updateErr := s.updatePolicySyncStatus(ctx, syncEntity); updateErr != nil {
			s.logger.Errorf("failed to update policy sync status: %v", updateErr)
		}
		return fmt.Errorf("fail to delete policy on verifier server, err: %w", err)
	}
	defer s.closer(resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		syncEntity.Status = itypes.Failed
		syncEntity.FailReason = fmt.Sprintf("failed to read response body: %s", err)
		// Update status in database before returning error
		if updateErr := s.updatePolicySyncStatus(ctx, syncEntity); updateErr != nil {
			s.logger.Errorf("failed to update policy sync status: %v", updateErr)
		}
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

		// Update status in database before returning error
		if updateErr := s.updatePolicySyncStatus(ctx, syncEntity); updateErr != nil {
			s.logger.Errorf("failed to update policy sync status: %v", updateErr)
		}

		return fmt.Errorf("fail to delete policy on verifier server, status: %d", resp.StatusCode)
	}

	// Success case
	syncEntity.Status = itypes.Synced
	syncEntity.FailReason = ""

	// Update status in database
	if err := s.updatePolicySyncStatus(ctx, syncEntity); err != nil {
		s.logger.Errorf("failed to update policy sync status: %v", err)
	}

	s.logger.WithFields(logrus.Fields{
		"policy_id": syncEntity.PolicyID,
	}).Info("Successfully sync deleted policy")

	return nil
}
