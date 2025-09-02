package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/types"
)

// This function inserts fee records for all billing records that have fees that need to be charged
func (s *PolicyService) HandleScheduledFees(ctx context.Context, task *asynq.Task) error {

	// Loop will always run at least once, and then only run if there are more records to process
	recurse := true
	for recurse {
		recurse = false
		query := `SELECT bp.billing_id, bp.amount, bp.next_billing_date, pp.public_key
	FROM billing_periods bp join plugin_policies pp on bp.plugin_policy_id = pp.id 
	WHERE bp.next_billing_date <= CURRENT_DATE 
	AND bp.active = true`
		rows, err := s.db.Pool().Query(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to query scheduled fees: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			recurse = true
			var billingID uuid.UUID
			var amount int
			var nextBillingDate time.Time
			var publicKey string
			if err := rows.Scan(&billingID, &amount, &nextBillingDate, &publicKey); err != nil {
				return fmt.Errorf("failed to scan scheduled fee row: %w", err)
			}
			recurse = true

			insertedId := uuid.Nil
			tx, err := s.db.Pool().Begin(ctx)
			if err != nil {
				return fmt.Errorf("failed to begin transaction: %w", err)
			}

			var txErr error
			defer func() {
				if txErr != nil {
					tx.Rollback(ctx)
				}
			}()

			insertedRecord, err := s.db.InsertFeeDebitTx(ctx, tx, types.FeeDebit{
				Fee: types.Fee{
					Amount:    uint64(amount),
					PublicKey: publicKey,
				},
				PluginPolicyBillingID: &billingID,
				ChargedAt:             nextBillingDate,
				Subtype:               types.FeeDebitSubtypeTypeFee,
			})
			if err != nil || insertedRecord == nil || insertedRecord.ID == uuid.Nil {
				txErr = err
				return fmt.Errorf("failed to insert scheduled fee record: %w", err)
			}

			s.logger.WithFields(logrus.Fields{
				"plugin_policy_billing_id": billingID,
				"amount":                   amount,
				"charged_at":               nextBillingDate,
				"fee_id":                   insertedId,
				"public_key":               publicKey,
			}).Info("Inserted scheduled fee record")

			if err := tx.Commit(ctx); err != nil {
				txErr = err
				return fmt.Errorf("failed to commit transaction: %w", err)
			}
		}

	}
	return nil
}
