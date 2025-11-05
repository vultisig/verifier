package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/vultisig/verifier/types"
)

func (p *PostgresBackend) MarkFeesCollected(ctx context.Context, collectedAt time.Time, ids []uuid.UUID, txid string) ([]types.Fee, error) {
	if txid == "" {
		return nil, fmt.Errorf("transaction hash cannot be empty")
	}

	var err error
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	query := `UPDATE fees SET collected_at = $1, transaction_hash = $2 WHERE id = ANY($3)`
	result, err := tx.Exec(ctx, query, collectedAt, txid, ids)
	if err != nil {
		return nil, err
	}

	// Check if all specified fees were updated
	rowsAffected := result.RowsAffected()
	if rowsAffected != int64(len(ids)) {
		return nil, fmt.Errorf("expected to update %d fees, but only updated %d", len(ids), rowsAffected)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (p *PostgresBackend) InsertFee(ctx context.Context, dbTx pgx.Tx, fee *types.Fee) error {
	query := `
  	INSERT INTO fees (
      policy_id, public_key, transaction_type, amount, fee_type, metadata, underlying_type, underlying_id
    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) ON CONFLICT (underlying_id, public_key)
	WHERE fee_type = 'installation_fee' AND underlying_type = 'plugin'
	DO NOTHING;`

	var (
		err error
	)
	if dbTx != nil {
		_, err = dbTx.Exec(ctx, query,
			fee.PolicyID, fee.PublicKey, fee.TxType, fee.Amount, fee.FeeType, fee.Metadata, fee.UnderlyingType, fee.UnderlyingID)
	} else {
		_, err = p.pool.Exec(ctx, query,
			fee.PolicyID, fee.PublicKey, fee.TxType, fee.Amount, fee.FeeType, fee.Metadata, fee.UnderlyingType, fee.UnderlyingID)
	}

	if err != nil {
		return err
	}
	return nil
}
