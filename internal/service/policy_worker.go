package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
)

func (s *PolicyService) HandleOneTimeFeeRecord(ctx context.Context, task *asynq.Task) error {
	var id uuid.UUID
	//TODO garry if there are errors here we need to handle them properly
	if err := id.UnmarshalBinary(task.Payload()); err != nil {
		return fmt.Errorf("failed to unmarshal task payload: %w", err)
	}

	query := `SELECT ppb.id, ppb.amount from plugin_policies pp 
	LEFT JOIN plugin_policy_billing ppb ON ppb.plugin_policy_id = pp.id 
	LEFT JOIN fees f on f.plugin_policy_billing_id = ppb.id 
	WHERE ppb."type" = 'once' AND f.id IS NULL AND pp.id = $1`

	rows, err := s.db.Pool().Query(ctx, query, id)
	if err != nil {
		//TODO garry we need to handle this error properly
		return fmt.Errorf("failed to query one-time fee records: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
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
			s.logger.WithError(err).WithFields(logrus.Fields{
				"plugin_policy_id":         id,
				"plugin_policy_billing_id": billingID,
				"amount":                   amount,
			}).Error("Failed to insert one-time fee record")
			return fmt.Errorf("failed to insert one-time fee record: %w", err)
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
	fmt.Println("Handling scheduled fees")
	query := `SELECT pb.billing_id, pb.amount, pb.next_billing_date FROM billing_periods pb WHERE pb.next_billing_date <= CURRENT_DATE AND pb.active = true`
	rows, err := s.db.Pool().Query(ctx, query)
	if err != nil {
		fmt.Println(err)
		s.logger.WithError(err).Error("Failed to query scheduled fees")
		return fmt.Errorf("failed to query scheduled fees: %w", err)
	}
	defer rows.Close()
	recurse := false

	for rows.Next() {
		var billingID uuid.UUID
		var amount int
		var nextBillingDate time.Time
		if err := rows.Scan(&billingID, &amount, &nextBillingDate); err != nil {
			fmt.Println(err)
			s.logger.WithError(err).Error("Failed to scan scheduled fee row")
			return fmt.Errorf("failed to scan scheduled fee row: %w", err)
		}
		recurse = true

		insertedId := uuid.Nil
		err = s.db.Pool().QueryRow(ctx, `INSERT INTO fees (plugin_policy_billing_id, amount, charged_at) VALUES ($1, $2, $3) RETURNING id`, billingID, amount, nextBillingDate).Scan(&insertedId)
		if err != nil {
			fmt.Println(err)
			s.logger.WithError(err).WithFields(logrus.Fields{
				"plugin_policy_billing_id": billingID,
				"amount":                   amount,
				"charged_at":               nextBillingDate,
			}).Error("Failed to insert scheduled fee record")
			return fmt.Errorf("failed to insert scheduled fee record: %w", err)
		}

		fmt.Println("success")
		s.logger.WithFields(logrus.Fields{
			"plugin_policy_billing_id": billingID,
			"amount":                   amount,
			"charged_at":               nextBillingDate,
			"fee_id":                   insertedId,
		}).Info("Inserted scheduled fee record")

	}

	// In case of maintenance, down time etc, if may be the case that several bill cycles have been missed. Therefore we rerun the task with an updated next_billing_cycle value. If no values are returned by the subsequent query then we end.
	if recurse {
		fmt.Println("Recursing")
		rows.Close()
		s.HandleScheduledFees(ctx, task)
	} else {
		fmt.Println("Not recursing")
	}

	return nil
}
