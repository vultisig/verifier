package postgres

import (
	"context"

	"github.com/google/uuid"
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
