package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
)

func (s *PolicyService) HandleOneTimeFeeRecord(ctx context.Context, task *asynq.Task) error {
	fmt.Println("Handling one-time fee record task")
	var id uuid.UUID
	//TODO garry if there are errors here we need to handle them properly
	if err := id.UnmarshalBinary(task.Payload()); err != nil {
		fmt.Println("Failed to unmarshal task payload:", err)
		return fmt.Errorf("failed to unmarshal task payload: %w", err)
	}

	query := `SELECT ppb.id, ppb.amount from plugin_policies pp 
	LEFT JOIN plugin_policy_billing ppb ON ppb.plugin_policy_id = pp.id 
	LEFT JOIN fees f on f.plugin_policy_billing_id = ppb.id 
	WHERE ppb."type" = 'once' AND f.id IS NULL AND pp.id = $1`

	rows, err := s.db.Pool().Query(ctx, query, id)
	if err != nil {
		//TODO garry we need to handle this error properly
		fmt.Println("Failed to query one-time fee records:", err)
		return fmt.Errorf("failed to query one-time fee records: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		fmt.Println("Row")
		var billingID uuid.UUID
		var amount int
		rows.Scan(
			&billingID,
			&amount,
		)

		var feeId uuid.UUID
		var feeAmount int
		err = s.db.Pool().QueryRow(ctx, `INSERT INTO fees (plugin_policy_billing_id, amount) VALUES ($1, $2) RETURNING id, amount`, billingID, amount).Scan(&feeId, &feeAmount)
		if err != nil {
			//TODO garry we need to handle this error properly
			fmt.Println("Failed to insert one-time fee record X:", err)
		}
		s.logger.WithFields(logrus.Fields{
			"plugin_policy_id":         id,
			"plugin_policy_billing_id": billingID,
			"fee_id":                   feeId,
			"amount":                   feeAmount,
		}).Info("Inserted one-time fee record")
	}

	return nil
}

func (s *PolicyService) HandleScheduledFees(ctx context.Context, task *asynq.Task) error {
	//TODO garry implement this
	return nil
}
