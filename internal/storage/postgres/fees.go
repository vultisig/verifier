package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/rpc"
	"github.com/vultisig/verifier/types"
)

const (
	queryInsertPluginInstallation = `INSERT INTO fees (
        policy_id, plugin_id, public_key, transaction_type, amount,
        fee_type, metadata, underlying_type, underlying_id
    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    ON CONFLICT (underlying_id, public_key)
    WHERE fee_type = 'installation_fee' AND underlying_type = 'plugin'
    DO NOTHING
    RETURNING id
    `
	queryInsertTrial = `INSERT INTO fees (
            policy_id, plugin_id, public_key, transaction_type, amount,
            fee_type, metadata, underlying_type, underlying_id
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        ON CONFLICT (public_key)
        WHERE fee_type = 'trial'
        DO NOTHING
        RETURNING id
        `
	queryInsertFee = `INSERT INTO fees (
            policy_id, plugin_id, public_key, transaction_type, amount,
            fee_type, metadata, underlying_type, underlying_id
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id
        `
	queryTrialStartDate = `SELECT created_at
        FROM fees
        WHERE public_key      = $1
          AND fee_type        = 'trial'
        ORDER BY created_at DESC
        LIMIT 1;`

	trialDuration = 7 * 24 * time.Hour
)

func (p *PostgresBackend) InsertFee(ctx context.Context, dbTx pgx.Tx, fee *types.Fee) (uint64, error) {
	var query string
	switch fee.FeeType {
	case types.FeeTypeInstallationFee:
		query = queryInsertPluginInstallation
	case types.FeeTypeTrial:
		query = queryInsertTrial
	default:
		query = queryInsertFee
	}

	var feeID uint64
	var err error

	if dbTx != nil {
		err = dbTx.QueryRow(ctx, query,
			fee.PolicyID, fee.PluginID, fee.PublicKey, fee.TxType, fee.Amount,
			fee.FeeType, fee.Metadata, fee.UnderlyingType, fee.UnderlyingID,
		).Scan(&feeID)
	} else {
		err = p.pool.QueryRow(ctx, query,
			fee.PolicyID, fee.PluginID, fee.PublicKey, fee.TxType, fee.Amount,
			fee.FeeType, fee.Metadata, fee.UnderlyingType, fee.UnderlyingID,
		).Scan(&feeID)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to insert fee: %w", err)
	}

	return feeID, nil
}

func (p *PostgresBackend) GetFeesByPublicKey(ctx context.Context, publicKey string) ([]*types.Fee, error) {
	query := `
    WITH last_cutoff AS (
        SELECT COALESCE(MAX(fb.batch_cutoff), 0) as cutoff_id
        FROM fee_batches fb
        WHERE fb.batch_cutoff IS NOT NULL
          AND EXISTS (
              SELECT 1 
              FROM fee_batch_members fbm
              JOIN fees f ON fbm.fee_id = f.id
              WHERE fbm.batch_id = fb.id
                AND f.public_key = $1
          )
    )
    SELECT
        f.id,
        f.policy_id,
        f.plugin_id,
        f.public_key,
        f.transaction_type,
        f.amount,
        f.created_at,
        f.fee_type,
        f.metadata,
        f.underlying_type,
        f.underlying_id
    FROM fees f, last_cutoff lc
    WHERE f.public_key = $1
      AND f.id > lc.cutoff_id
    ORDER BY f.created_at ASC
    `

	rows, err := p.pool.Query(ctx, query, publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to query debit fees for public key: %w", err)
	}
	defer rows.Close()

	var fees []*types.Fee
	for rows.Next() {
		fee := &types.Fee{}
		var pluginID *string
		err := rows.Scan(
			&fee.ID,
			&fee.PolicyID,
			&pluginID,
			&fee.PublicKey,
			&fee.TxType,
			&fee.Amount,
			&fee.CreatedAt,
			&fee.FeeType,
			&fee.Metadata,
			&fee.UnderlyingType,
			&fee.UnderlyingID,
		)
		if pluginID != nil {
			fee.PluginID = *pluginID
		}
		if err != nil {
			return nil, fmt.Errorf("failed to scan fee row: %w", err)
		}
		fees = append(fees, fee)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating fee rows: %w", err)
	}

	return fees, nil
}

func (p *PostgresBackend) GetFeeById(ctx context.Context, id uint64) (*types.Fee, error) {
	query := `
        SELECT
            id,
            policy_id,
            plugin_id,
            public_key,
            transaction_type,
            amount,
            created_at,
            fee_type,
            metadata,
            underlying_type,
            underlying_id
        FROM fees
        WHERE id = $1
    `

	fee := &types.Fee{}
	var pluginID *string
	err := p.pool.QueryRow(ctx, query, id).Scan(
		&fee.ID,
		&fee.PolicyID,
		&pluginID,
		&fee.PublicKey,
		&fee.TxType,
		&fee.Amount,
		&fee.CreatedAt,
		&fee.FeeType,
		&fee.Metadata,
		&fee.UnderlyingType,
		&fee.UnderlyingID,
	)
	if pluginID != nil {
		fee.PluginID = *pluginID
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get fee by id: %w", err)
	}

	return fee, nil
}

func (p *PostgresBackend) MarkFeesCollected(ctx context.Context, dbTx pgx.Tx, feeIDs []uint64, txHash string, totalAmount uint64) error {
	var publicKey string
	var feeCount int
	var distinctKeys int
	err := dbTx.QueryRow(ctx, `
        SELECT 
            MIN(public_key) as public_key,
            COUNT(*) as fee_count,
            COUNT(DISTINCT public_key) as distinct_keys
        FROM fees
        WHERE id = ANY($1)
    `, feeIDs).Scan(&publicKey, &feeCount, &distinctKeys)
	if err != nil {
		return fmt.Errorf("failed to validate fees: %w", err)
	}

	if feeCount != len(feeIDs) {
		return fmt.Errorf("fee count mismatch: expected %d, found %d", len(feeIDs), feeCount)
	}

	if distinctKeys != 1 {
		return fmt.Errorf("fees belong to multiple public keys: found %d distinct keys", distinctKeys)
	}

	var batchID int64
	err = dbTx.QueryRow(ctx, `
        INSERT INTO fee_batches (total_value, status, batch_cutoff, collection_tx_id)
        VALUES ($1, 'SIGNED', 0, $2)
        RETURNING id
    `, totalAmount, txHash).Scan(&batchID)
	if err != nil {
		return fmt.Errorf("failed to create batch: %w", err)
	}

	_, err = dbTx.Exec(ctx, `
        INSERT INTO fee_batch_members (batch_id, fee_id)
        SELECT $1, unnest($2::bigint[])
    `, batchID, feeIDs)
	if err != nil {
		return fmt.Errorf("failed to insert batch members: %w", err)
	}

	metadata := map[string]interface{}{
		"tx_hash":  txHash,
		"batch_id": batchID,
		"fee_ids":  feeIDs,
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	creditFee := &types.Fee{
		PublicKey:      publicKey,
		TxType:         types.TxTypeCredit,
		Amount:         totalAmount,
		FeeType:        types.FeeTypeBatch,
		Metadata:       metadataJSON,
		UnderlyingType: "batch",
		UnderlyingID:   fmt.Sprint(batchID),
	}

	creditID, err := p.InsertFee(ctx, dbTx, creditFee)
	if err != nil {
		return fmt.Errorf("failed to insert credit: %w", err)
	}

	_, err = dbTx.Exec(ctx, `
        UPDATE fee_batches
        SET batch_cutoff = $1
        WHERE id = $2
    `, creditID, batchID)
	if err != nil {
		return fmt.Errorf("failed to update batch cutoff: %w", err)
	}

	return nil
}

func (p *PostgresBackend) GetUserFees(
	ctx context.Context,
	publicKey string,
) (*types.UserFeeStatus, error) {
	query := `
    SELECT
        f.id,
        f.policy_id,
        f.plugin_id,
        f.public_key,
        f.transaction_type,
        f.amount,
        f.created_at,
        f.fee_type,
        f.metadata,
        f.underlying_type,
        f.underlying_id
    FROM fees f
    WHERE f.public_key = $1
    ORDER BY f.created_at ASC
    `

	rows, err := p.pool.Query(ctx, query, publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to query fees: %w", err)
	}
	defer rows.Close()

	result := &types.UserFeeStatus{
		PublicKey:    publicKey,
		Fees:         make([]*types.Fee, 0),
		Balance:      0,
		UnpaidAmount: 0,
	}

	for rows.Next() {
		fee := &types.Fee{}
		var pluginID *string
		err := rows.Scan(
			&fee.ID,
			&fee.PolicyID,
			&pluginID,
			&fee.PublicKey,
			&fee.TxType,
			&fee.Amount,
			&fee.CreatedAt,
			&fee.FeeType,
			&fee.Metadata,
			&fee.UnderlyingType,
			&fee.UnderlyingID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan fee row: %w", err)
		}
		if pluginID != nil {
			fee.PluginID = *pluginID
		}

		if fee.TxType == types.TxTypeCredit {
			result.Balance += int64(fee.Amount)
		} else if fee.TxType == types.TxTypeDebit {
			result.Balance -= int64(fee.Amount)
			result.Fees = append(result.Fees, fee)
		}
	}

	result.UnpaidAmount = max(0, -result.Balance)

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating fee rows: %w", err)
	}

	return result, nil
}

func (p *PostgresBackend) UpdateBatchStatus(ctx context.Context, dbTx pgx.Tx, txHash string, status *rpc.TxOnChainStatus) error {
	if status == nil {
		return fmt.Errorf("status cannot be nil")
	}

	switch *status {
	case rpc.TxOnChainSuccess:
		_, err := dbTx.Exec(ctx, `
           UPDATE fee_batches
           SET status = 'COMPLETED'
           WHERE collection_tx_id = $1
       `, txHash)
		if err != nil {
			return fmt.Errorf("failed to update batch status to COMPLETED: %w", err)
		}
		return nil
	case rpc.TxOnChainFail:
		var batchID int64
		var originalFee types.Fee
		err := dbTx.QueryRow(ctx, `
    SELECT 
        fb.id,
        f.policy_id,
        f.public_key,
        f.amount,
        f.metadata,
        f.underlying_type,
        f.underlying_id
    FROM fee_batches fb
    JOIN fees f ON f.id = fb.batch_cutoff
    WHERE fb.collection_tx_id = $1
`, txHash).Scan(
			&batchID,
			&originalFee.PolicyID,
			&originalFee.PublicKey,
			&originalFee.Amount,
			&originalFee.Metadata,
			&originalFee.UnderlyingType,
			&originalFee.UnderlyingID,
		)
		if err != nil {
			return fmt.Errorf("failed to get batch and original fee: %w", err)
		}

		compensationFee := &types.Fee{
			PolicyID:       originalFee.PolicyID,
			PublicKey:      originalFee.PublicKey,
			TxType:         types.TxTypeDebit,
			Amount:         originalFee.Amount,
			FeeType:        types.FeeTypeBatchFailed,
			Metadata:       originalFee.Metadata,
			UnderlyingType: "batch",
			UnderlyingID:   fmt.Sprint(batchID),
		}

		_, err = p.InsertFee(ctx, dbTx, compensationFee)
		if err != nil {
			return fmt.Errorf("failed to insert compensation fee: %w", err)
		}

		_, err = dbTx.Exec(ctx, `
           UPDATE fee_batches
           SET status = 'FAILED'
           WHERE collection_tx_id = $1
       `, txHash)
		if err != nil {
			return fmt.Errorf("failed to update batch status to FAILED: %w", err)
		}

		return nil
	case rpc.TxOnChainPending:
		return nil
	default:
		return fmt.Errorf("unknown status: %s", *status)
	}
}

func (p *PostgresBackend) IsTrialActive(
	ctx context.Context,
	dbTx pgx.Tx,
	pubKey string,
) (bool, time.Duration, error) {
	if dbTx == nil {
		return false, 0, fmt.Errorf("dbTx is nil")
	}
	var createdAt time.Time
	err := dbTx.QueryRow(ctx, queryTrialStartDate, pubKey).Scan(&createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_, err := p.InsertFee(ctx, dbTx, &types.Fee{
				PublicKey:      pubKey,
				TxType:         types.TxTypeCredit,
				Amount:         1,
				FeeType:        types.FeeTypeTrial,
				UnderlyingType: "user",
				UnderlyingID:   "trial",
			})
			if err != nil {
				return false, 0, fmt.Errorf("failed to insert trial record: %w", err)
			}
			return true, trialDuration, nil
		}
		return false, 0, fmt.Errorf("failed to query TrialStartDate: %w", err)
	}

	trialExpiresAt := createdAt.Add(trialDuration)
	isActive := trialExpiresAt.After(time.Now())

	trialRemaining := time.Until(trialExpiresAt)
	if trialRemaining < 0 {
		trialRemaining = 0
	}

	return isActive, trialRemaining, nil
}
