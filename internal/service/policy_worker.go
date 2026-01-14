package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/types"
)

const QUERY_GET_EXPIRED_SUBSCRIPTIONS = `SELECT
  p.public_key,
  b.plugin_policy_id AS policy_id,
  p.plugin_id,
  b.amount
FROM plugin_policies p
JOIN plugin_policy_billing b ON b.plugin_policy_id = p.id
LEFT JOIN (
  SELECT policy_id, public_key, MAX(created_at) AS last_fee_at
  FROM fees
  WHERE fee_type = 'subscription_fee'
  GROUP BY policy_id, public_key
) f ON f.policy_id = p.id AND f.public_key = p.public_key
WHERE b.type = 'recurring'
  AND (
    f.last_fee_at IS NULL
    OR (f.last_fee_at + b.frequency::interval) <= CURRENT_DATE
  )`

func (s *PolicyService) HandleScheduledFees(ctx context.Context, task *asynq.Task) error {
	query := QUERY_GET_EXPIRED_SUBSCRIPTIONS
	rows, err := s.db.Pool().Query(ctx, query)
	if err != nil {
		fmt.Println(err)
		s.logger.WithError(err).Error("Failed to query scheduled fees")
		return fmt.Errorf("failed to query scheduled fees: %w", err)
	}
	defer rows.Close()

	var feesToInsert []struct {
		publicKey string
		policyId  string
		pluginId  string
		amount    uint64
	}
	for rows.Next() {
		var res struct {
			publicKey string
			policyId  string
			pluginId  string
			amount    uint64
		}
		err := rows.Scan(&res.publicKey, &res.policyId, &res.pluginId, &res.amount)
		if err != nil {
			fmt.Println(err)
			s.logger.WithError(err).Error("Failed to scan scheduled fee row")
			return fmt.Errorf("failed to scan scheduled fee row: %w", err)
		}
		feesToInsert = append(feesToInsert, res)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	for _, res := range feesToInsert {
		_, err = s.db.InsertFee(ctx, nil, &types.Fee{
			PluginID:       res.pluginId,
			PublicKey:      res.publicKey,
			TxType:         types.TxTypeDebit,
			Amount:         res.amount,
			FeeType:        types.FeeSubscriptionFee,
			UnderlyingType: "policy",
			UnderlyingID:   res.policyId,
		})
		if err != nil {
			s.logger.WithError(err).WithFields(logrus.Fields{
				"plugin_policy_id": res.policyId,
				"amount":           res.amount,
			}).Error("Failed to insert scheduled fee record")
			return fmt.Errorf("failed to insert scheduled fee record: %w", err)
		}

		s.logger.WithFields(logrus.Fields{
			"plugin_policy_id": res.policyId,
			"amount":           res.amount,
		}).Info("Inserted scheduled fee record")
	}

	if len(feesToInsert) > 0 {
		return s.HandleScheduledFees(ctx, task)
	}

	return nil
}

// HandlePolicyDeactivate handles deferred policy deactivation tasks.
// The task payload contains the policy ID as a raw string.
func (s *PolicyService) HandlePolicyDeactivate(ctx context.Context, task *asynq.Task) error {
	policyIDStr := string(task.Payload())
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		s.logger.WithError(err).WithField("payload", policyIDStr).Error("Invalid policy ID in deactivation task")
		return fmt.Errorf("invalid policy ID: %w", err)
	}

	policy, err := s.db.GetPluginPolicy(ctx, policyID)
	if err != nil {
		s.logger.WithError(err).WithField("policy_id", policyID).Error("Failed to get policy for deactivation")
		return fmt.Errorf("failed to get policy: %w", err)
	}

	// Already deactivated (maybe manually or by another process)
	if !policy.Active {
		s.logger.WithField("policy_id", policyID).Debug("Policy already inactive, skipping deactivation")
		return nil
	}

	policy.Deactivate(types.DeactivationReasonExpiry)
	_, err = s.UpdatePolicy(ctx, *policy)
	if err != nil {
		s.logger.WithError(err).WithField("policy_id", policyID).Error("Failed to deactivate policy")
		return fmt.Errorf("failed to deactivate policy: %w", err)
	}

	s.logger.WithField("policy_id", policyID).Info("Policy deactivated via scheduled task")
	return nil
}
