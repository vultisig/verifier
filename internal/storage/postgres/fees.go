package postgres

import (
	"context"
	"fmt"

	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/vultisig/verifier/types"
)

func (p *PostgresBackend) GetAllFeesByPolicyId(ctx context.Context, policyID uuid.UUID) ([]types.Fee, error) {
	fees := []types.Fee{}

	query := `SELECT id, public_key, plugin_id, policy_id, type, transaction_id, amount, charged_at, created_at, collected_at FROM fees_view WHERE policy_id = $1`
	rows, err := p.pool.Query(ctx, query, policyID)
	if err != nil {
		return []types.Fee{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var fee types.Fee
		err := rows.Scan(
			&fee.ID,
			&fee.PublicKey,
			&fee.PluginID,
			&fee.PolicyID,
			&fee.Type,
			&fee.TransactionID,
			&fee.Amount,
			&fee.ChargedAt,
			&fee.CreatedAt,
			&fee.CollectedAt,
		)
		if err != nil {
			return []types.Fee{}, err
		}
		fees = append(fees, fee)
	}
	return fees, nil
}

func (p *PostgresBackend) GetFeesByPublicKey(ctx context.Context, publicKey string, since *time.Time) ([]types.Fee, error) {
	fees := []types.Fee{}
	var rows pgx.Rows
	var err error
	if since != nil {
		query := `SELECT id, public_key, plugin_id, policy_id, type, transaction_id, amount, charged_at, created_at, collected_at FROM fees_view WHERE public_key = $1 AND created_at >= $2`
		rows, err = p.pool.Query(ctx, query, publicKey, since)
	} else {
		query := `SELECT id, public_key, plugin_id, policy_id, type, transaction_id, amount, charged_at, created_at, collected_at FROM fees_view WHERE public_key = $1 AND collected_at IS NULL`
		rows, err = p.pool.Query(ctx, query, publicKey)
	}
	if err != nil {
		return []types.Fee{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var fee types.Fee
		err := rows.Scan(
			&fee.ID,
			&fee.PublicKey,
			&fee.PluginID,
			&fee.PolicyID,
			&fee.Type,
			&fee.TransactionID,
			&fee.Amount,
			&fee.ChargedAt,
			&fee.CreatedAt,
			&fee.CollectedAt,
		)
		if err != nil {
			return []types.Fee{}, err
		}
		fees = append(fees, fee)
	}
	return fees, nil
}

func (p *PostgresBackend) GetAllFeesByPublicKey(ctx context.Context, includeCollected bool) ([]types.Fee, error) {
	fees := []types.Fee{}
	query := `SELECT id, public_key, plugin_id, policy_id, type, transaction_id, amount, charged_at, created_at, collected_at FROM fees_view WHERE collected_at IS NULL ORDER BY public_key, created_at`
	if includeCollected {
		query = `SELECT id, public_key, plugin_id, policy_id, type, transaction_id, amount, charged_at, created_at, collected_at FROM fees_view ORDER BY public_key, created_at`
	}
	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return []types.Fee{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var fee types.Fee
		err := rows.Scan(
			&fee.ID,
			&fee.PublicKey,
			&fee.PluginID,
			&fee.PolicyID,
			&fee.Type,
			&fee.TransactionID,
			&fee.Amount,
			&fee.ChargedAt,
			&fee.CreatedAt,
			&fee.CollectedAt,
		)
		if err != nil {
			return []types.Fee{}, err
		}
		fees = append(fees, fee)
	}
	return fees, nil
}

func (p *PostgresBackend) GetFeesByIds(ctx context.Context, ids []uuid.UUID) ([]types.Fee, error) {
	fees := []types.Fee{}
	query := `SELECT id, public_key, plugin_id, policy_id, type, transaction_id, amount, charged_at, created_at, collected_at FROM fees_view WHERE id = ANY($1)`
	rows, err := p.pool.Query(ctx, query, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var fee types.Fee
		err := rows.Scan(
			&fee.ID,
			&fee.PublicKey,
			&fee.PluginID,
			&fee.PolicyID,
			&fee.Type,
			&fee.TransactionID,
			&fee.Amount,
			&fee.ChargedAt,
			&fee.CreatedAt,
			&fee.CollectedAt,
		)
		if err != nil {
			return nil, err
		}
		fees = append(fees, fee)
	}
	return fees, nil
}

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

	return p.GetFeesByIds(ctx, ids)
}
