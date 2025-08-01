package postgres

import (
	"context"
	"fmt"

	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/vultisig/verifier/types"
)

func (p *PostgresBackend) GetFees(ctx context.Context, ids ...uuid.UUID) ([]types.Fee, error) {
	query := `SELECT id, public_key, plugin_id, policy_id, type, transaction_id, amount, charged_at, created_at, collected_at FROM fees_view WHERE id = ANY($1)`
	rows, err := p.pool.Query(ctx, query, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fees := []types.Fee{}
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

func (p *PostgresBackend) GetFeesByPublicKey(ctx context.Context, publicKey string, includeCollected bool) ([]types.Fee, error) {
	fees := []types.Fee{}
	query := `SELECT id, public_key, plugin_id, policy_id, type, transaction_id, amount, charged_at, created_at, collected_at FROM fees_view WHERE public_key = $1 AND collected_at IS NULL`
	if includeCollected {
		query = `SELECT id, public_key, plugin_id, policy_id, type, transaction_id, amount, charged_at, created_at, collected_at FROM fees_view WHERE public_key = $1`
	}
	rows, err := p.pool.Query(ctx, query, publicKey)
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

func (p *PostgresBackend) MarkFeesCollected(ctx context.Context, tx pgx.Tx, collectedAt time.Time, ids []uuid.UUID, txid string) error {
	if txid == "" {
		return fmt.Errorf("transaction hash cannot be empty")
	}
	for _, id := range ids {
		query := `UPDATE fees SET collected_at = $1, transaction_hash = $2 WHERE id = $3`
		_, err := tx.Exec(ctx, query, collectedAt, txid, id)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *PostgresBackend) CreateTreasuryLedgerRecord(ctx context.Context, tx pgx.Tx, feeAccountRecord types.TreasuryLedgerRecord) error {
	query := `INSERT INTO treasury_ledger (
		amount, 
		type, 
		fee_id,
		developer_id,
		tx_hash,
		reference, 
		created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := p.pool.Exec(ctx, query,
		feeAccountRecord.Amount,
		feeAccountRecord.Type,
		feeAccountRecord.FeeID,
		feeAccountRecord.DeveloperID,
		feeAccountRecord.TxHash,
		feeAccountRecord.Reference,
		feeAccountRecord.CreatedAt)
	if err != nil {
		return err
	}
	return nil
}
