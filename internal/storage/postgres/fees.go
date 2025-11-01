package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/vultisig/verifier/types"
)

func (p *PostgresBackend) InsertFee(ctx context.Context, dbTx pgx.Tx, fee *types.Fee) error {
	query := `
  	INSERT INTO fees (
      policy_id, public_key, transaction_type, amount, fee_type, metadata, underlying_type, underlying_id
    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

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

func (p *PostgresBackend) GetFeesByPublicKey(ctx context.Context, publicKey string) ([]*types.Fee, error) {
	query := `
        SELECT 
            d.id,
            d.policy_id,
            d.public_key,
            d.transaction_type,
            d.amount,
            d.created_at,
            d.fee_type,
            d.metadata,
            d.underlying_type,
            d.underlying_id
        FROM fees d
        LEFT JOIN fees c ON c.transaction_type = 'credit' 
            AND c.metadata::jsonb->>'debit_fee_id' = d.id::text
        WHERE d.transaction_type = 'debit'
            AND d.public_key = $1
            AND c.id IS NULL
        ORDER BY d.created_at ASC
    `

	rows, err := p.pool.Query(ctx, query, publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to query debit fees for public key: %w", err)
	}
	defer rows.Close()

	var fees []*types.Fee
	for rows.Next() {
		fee := &types.Fee{}
		err := rows.Scan(
			&fee.ID,
			&fee.PolicyID,
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
	err := p.pool.QueryRow(ctx, query, id).Scan(
		&fee.ID,
		&fee.PolicyID,
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get fee by id: %w", err)
	}

	return fee, nil
}
