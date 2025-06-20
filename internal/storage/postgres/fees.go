package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/vultisig/verifier/types"
)

func (p *PostgresBackend) GetAllFeesByPolicyId(ctx context.Context, policyID uuid.UUID) ([]types.Fee, error) {
	fees := []types.Fee{}

	query := `SELECT id, type, transaction_id, amount, charged_at, created_at, collected_at FROM fees_view WHERE policy_id = $1`
	rows, err := p.pool.Query(ctx, query, policyID)
	if err != nil {
		return []types.Fee{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var fee types.Fee
		err := rows.Scan(
			&fee.ID,
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
