package postgres

import (
	"context"
	"fmt"

	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/types"
)

// Extrapolate the public key from the policy ID and returns the output from `GetFeeDebitsByPublicKey“
func (p *PostgresBackend) GetFeeDebitsByPolicyId(ctx context.Context, policyID uuid.UUID, since *time.Time) ([]types.FeeDebit, error) {
	query := `SELECT public_key FROM plugin_policies WHERE id = $1`
	rows, err := p.pool.Query(ctx, query, policyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var publicKey string
		err := rows.Scan(&publicKey)
		if err != nil {
			return nil, err
		}
		return p.GetFeeDebitsByPublicKey(ctx, publicKey, since)
	} else {
		return nil, fmt.Errorf("no public key found for policy %s", policyID)
	}
}

func (p *PostgresBackend) GetFeeDebitsByPublicKey(ctx context.Context, publicKey string, since *time.Time) ([]types.FeeDebit, error) {
	fees := []types.FeeDebit{}
	var rows pgx.Rows
	var err error
	if since != nil {
		query := `SELECT id, public_key, type, amount, plugin_policy_billing_id, charged_at, created_at, ref FROM fee_debits WHERE public_key = $1 AND created_at >= $2`
		rows, err = p.pool.Query(ctx, query, publicKey, since)
	} else {
		query := `SELECT id, public_key, type, amount, plugin_policy_billing_id, charged_at, created_at, ref FROM fee_debits WHERE public_key = $1`
		rows, err = p.pool.Query(ctx, query, publicKey)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var fee types.FeeDebit
		err := rows.Scan(
			&fee.ID,
			&fee.PublicKey,
			&fee.Type,
			&fee.Amount,
			&fee.PluginPolicyBillingID,
			&fee.ChargedAt,
			&fee.CreatedAt,
			&fee.Ref,
		)
		if err != nil {
			return nil, err
		}
		fees = append(fees, fee)
	}
	return fees, nil
}

func (p *PostgresBackend) GetFeeCreditsByIds(ctx context.Context, ids []uuid.UUID) ([]types.FeeCredit, error) {
	fees := []types.FeeCredit{}
	query := `SELECT id, public_key, type, amount, created_at, transaction_hash, ref FROM fee_credits WHERE id = ANY($1)`
	rows, err := p.pool.Query(ctx, query, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var fee types.FeeCredit
		err := rows.Scan(
			&fee.ID,
			&fee.PublicKey,
			&fee.Type,
			&fee.Amount,
			&fee.CreatedAt,
			&fee.Ref,
		)
		if err != nil {
			return nil, err
		}
		fees = append(fees, fee)
	}
	return fees, nil
}

func (p *PostgresBackend) GetFeeDebitsByIds(ctx context.Context, ids []uuid.UUID) ([]types.FeeDebit, error) {
	fees := []types.FeeDebit{}
	query := `SELECT id, public_key, type, amount, created_at, plugin_policy_billing_id, charged_at, ref FROM fee_debits WHERE id = ANY($1)`
	rows, err := p.pool.Query(ctx, query, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var fee types.FeeDebit
		err := rows.Scan(
			&fee.ID,
			&fee.PublicKey,
			&fee.Type,
			&fee.Amount,
			&fee.CreatedAt,
			&fee.PluginPolicyBillingID,
			&fee.ChargedAt,
			&fee.Ref,
		)
		if err != nil {
			return nil, err
		}
		fees = append(fees, fee)
	}
	return fees, nil
}

func (p *PostgresBackend) GetFeesOwed(ctx context.Context, publicKey string, ids ...uuid.UUID) (int64, error) {
	var row pgx.Row
	if len(ids) == 0 {
		// Use the original view for all fees
		query := `SELECT public_key, total_owed FROM fee_balance WHERE public_key = $1`
		row = p.pool.QueryRow(ctx, query, publicKey)
	} else {
		query := `SELECT public_key, COALESCE(SUM(total_owed), 0) FROM fee_balance_for_ids($1) WHERE public_key = $2 group by public_key`
		row = p.pool.QueryRow(ctx, query, ids, publicKey)
	}
	var scannedPublicKey string
	var totalOwed int64
	err := row.Scan(&scannedPublicKey, &totalOwed)
	if err != nil {
		return 0, err
	}
	return totalOwed, nil
}

// InsertFee inserts a fee record for a billing policy within a transaction
func (p *PostgresBackend) InsertFeeCreditTx(ctx context.Context, dbTx pgx.Tx, fee types.FeeCredit) (*types.FeeCredit, error) {
	if fee.ID == uuid.Nil {
		fee.ID = uuid.New()
	}
	if fee.Amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than 0")
	}

	err := dbTx.QueryRow(ctx,
		`INSERT INTO fee_credits (id, public_key, subtype, amount, ref) VALUES ($1, $2, $3, $4, $5) RETURNING id, public_key, type, subtype, amount, ref, created_at`,
		fee.ID, fee.PublicKey, fee.Subtype, fee.Amount, fee.Ref,
	).Scan(&fee.ID, &fee.PublicKey, &fee.Type, &fee.Subtype, &fee.Amount, &fee.Ref, &fee.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to insert fee record for public_key %s: %w", fee.PublicKey, err)
	}

	return &fee, nil
}

func (p *PostgresBackend) InsertFeeDebitTx(ctx context.Context, dbTx pgx.Tx, fee types.FeeDebit) (*types.FeeDebit, error) {
	if fee.ID == uuid.Nil {
		fee.ID = uuid.New()
	}
	if fee.Amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than 0")
	}

	err := dbTx.QueryRow(ctx,
		`INSERT INTO fee_debits (id, public_key, subtype, amount, plugin_policy_billing_id, charged_at, ref) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, public_key, type, subtype, amount, plugin_policy_billing_id, charged_at, ref, created_at`,
		fee.ID, fee.PublicKey, fee.Subtype, fee.Amount, fee.PluginPolicyBillingID, fee.ChargedAt, fee.Ref,
	).Scan(&fee.ID, &fee.PublicKey, &fee.Type, &fee.Subtype, &fee.Amount, &fee.PluginPolicyBillingID, &fee.ChargedAt, &fee.Ref, &fee.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to insert fee record for billing policy %s: %w", fee.PluginPolicyBillingID, err)
	}

	return &fee, nil
}

// Return a slice of all fees that are not in a batch
func (p *PostgresBackend) GetUnclaimedFeeMembers(ctx context.Context, publicKey string) ([]types.Fee, error) {
	query := `SELECT id, public_key, type, amount FROM fees WHERE public_key = $1 AND id NOT IN (SELECT fee_id FROM fee_batch_members)`
	rows, err := p.pool.Query(ctx, query, publicKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	fees := []types.Fee{}
	for rows.Next() {
		var fee types.Fee
		err := rows.Scan(&fee.ID, &fee.PublicKey, &fee.Type, &fee.Amount)
		if err != nil {
			return nil, err
		}
		fees = append(fees, fee)
	}
	return fees, nil
}

func (p *PostgresBackend) CreateFeeBatchWithMembers(ctx context.Context, dbTx pgx.Tx, publicKey string, batchId uuid.UUID, members ...uuid.UUID) error {

	if len(members) == 0 {
		return fmt.Errorf("no members provided")
	}

	_, err := dbTx.Exec(ctx, `INSERT INTO fee_batch (id, public_key) VALUES ($1, $2)`, batchId, publicKey)
	if err != nil {
		dbTx.Rollback(ctx)
		return err
	}

	for _, member := range members {
		_, err := dbTx.Exec(ctx, `INSERT INTO fee_batch_members (fee_batch_id, fee_id) VALUES ($1, $2)`, batchId, member)
		if err != nil {
			dbTx.Rollback(ctx)
			return err
		}
	}

	return nil
}

func (p *PostgresBackend) GetCreditTxByBatchId(ctx context.Context, batchId uuid.UUID) (*types.FeeCredit, error) {
	batchRef := fmt.Sprintf("batch:%s", batchId.String())
	query := `SELECT id, public_key, type, amount, created_at, ref FROM fee_credits WHERE ref LIKE '%' || $1 || '%'`
	row := p.pool.QueryRow(ctx, query, batchRef)
	var fee types.FeeCredit
	err := row.Scan(&fee.ID, &fee.PublicKey, &fee.Type, &fee.Amount, &fee.CreatedAt, &fee.Ref)
	if err != nil {
		return nil, err
	}
	return &fee, nil
}

func (p *PostgresBackend) GetFeeBatch(ctx context.Context, batchId uuid.UUID) (*types.FeeBatch, error) {
	query := `SELECT id, created_at, tx_hash, status FROM fee_batch WHERE id = $1`
	row := p.pool.QueryRow(ctx, query, batchId)
	var batch types.FeeBatch
	err := row.Scan(&batch.ID, &batch.CreatedAt, &batch.TxHash, &batch.Status)
	if err != nil {
		return nil, err
	}
	return &batch, nil
}

func (p *PostgresBackend) UpdateFeeBatch(ctx context.Context, dbTx pgx.Tx, batchId uuid.UUID, txHash string, status types.FeeBatchStatus) error {
	_, err := dbTx.Exec(ctx, `UPDATE fee_batch SET tx_hash = $1, status = $2 WHERE id = $3`, txHash, status, batchId)
	if err != nil {
		return err
	}
	return nil
}

func (p *PostgresBackend) GetFeeBatchAmount(ctx context.Context, batchId uuid.UUID) (uint64, error) {
	query := `SELECT fb.public_key, fbm.fee_id from fee_batch fb join fee_batch_members fbm on fb.id = fbm.fee_batch_id WHERE fb.id = $1`
	rows, err := p.pool.Query(ctx, query, batchId)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	ids := []uuid.UUID{}
	var publicKey string
	for rows.Next() {
		var feeId uuid.UUID
		var pk string
		err := rows.Scan(&pk, &feeId)
		if err != nil {
			return 0, err
		}
		if publicKey != "" && pk != publicKey {
			return 0, fmt.Errorf("fee batch public key mismatch")
		}
		publicKey = pk
		ids = append(ids, feeId)
	}

	amount, err := p.GetFeesOwed(ctx, publicKey, ids...)
	if err != nil {
		return 0, err
	}
	return uint64(amount), nil
}

func (p *PostgresBackend) GetFeeBatchesByStateAndPublicKey(ctx context.Context, publicKey string, status types.FeeBatchStatus) ([]itypes.FeeBatchRequest, error) {
	query := `SELECT id, public_key, status FROM fee_batch WHERE public_key = $1 AND status = $2`
	rows, err := p.pool.Query(ctx, query, publicKey, status)
	if err != nil {
		return nil, err
	}
	batches := []itypes.FeeBatchRequest{}
	defer rows.Close()
	for rows.Next() {
		var batchStatus types.FeeBatchStatus
		var batch itypes.FeeBatchRequest
		err := rows.Scan(&batch.BatchID, &batch.PublicKey, &batchStatus)
		if err != nil {
			return nil, err
		}
		amount, err := p.GetFeeBatchAmount(ctx, batch.BatchID)
		if err != nil {
			return nil, err
		}
		batch.Amount = amount
		batches = append(batches, batch)
	}
	return batches, nil
}
