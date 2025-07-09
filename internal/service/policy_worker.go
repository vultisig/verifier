package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
)

func (s *PolicyService) HandleScheduledFees(ctx context.Context, task *asynq.Task) error {
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
			s.logger.WithError(err).WithFields(logrus.Fields{
				"plugin_policy_billing_id": billingID,
				"amount":                   amount,
				"charged_at":               nextBillingDate,
			}).Error("Failed to insert scheduled fee record")
			return fmt.Errorf("failed to insert scheduled fee record: %w", err)
		}

		s.logger.WithFields(logrus.Fields{
			"plugin_policy_billing_id": billingID,
			"amount":                   amount,
			"charged_at":               nextBillingDate,
			"fee_id":                   insertedId,
		}).Info("Inserted scheduled fee record")

	}

	// In case of maintenance, down time etc, if may be the case that several bill cycles have been missed. Therefore we rerun the task with an updated next_billing_cycle value. If no values are returned by the subsequent query then we end.
	if recurse {
		rows.Close()
		s.HandleScheduledFees(ctx, task)
	}

	return nil
}
