package service

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/types"
)

const QUERY_GET_EXPIRED_SUBSCRIPTIONS = `SELECT
  p.public_key,
  b.plugin_policy_id AS policy_id,
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
	recurse := false

	for rows.Next() {
		var res struct {
			publicKey string
			policyId  string
			amount    uint64
		}
		if err := rows.Scan(&res.publicKey, &res.policyId, &res.amount); err != nil {
			fmt.Println(err)
			s.logger.WithError(err).Error("Failed to scan scheduled fee row")
			return fmt.Errorf("failed to scan scheduled fee row: %w", err)
		}
		recurse = true

		err = s.db.InsertFee(ctx, nil, &types.Fee{
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

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	// In case of maintenance, down time etc, if may be the case that several bill cycles have been missed.
	// Therefore we rerun the task with an updated next_billing_cycle value. If no values are returned by the
	// subsequent query then we end.
	if recurse {
		rows.Close()
		s.HandleScheduledFees(ctx, task)
	}

	return nil
}
