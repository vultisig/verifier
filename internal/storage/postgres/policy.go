package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/types"
)

func (p *PostgresBackend) GetPluginPolicy(ctx context.Context, id uuid.UUID) (types.PluginPolicy, error) {
	if p.pool == nil {
		return types.PluginPolicy{}, fmt.Errorf("database pool is nil")
	}

	var policy types.PluginPolicy
	var policyJSON []byte

	query := `
        SELECT id, public_key, is_ecdsa, chain_code_hex, plugin_id, plugin_version, policy_version, plugin_type, signature, active, policy 
        FROM plugin_policies 
        WHERE id = $1`

	err := p.pool.QueryRow(ctx, query, id).Scan(
		&policy.ID,
		&policy.PublicKey,
		&policy.IsEcdsa,
		&policy.ChainCodeHex,
		&policy.PluginID,
		&policy.PluginVersion,
		&policy.PolicyVersion,
		&policy.PluginType,
		&policy.Signature,
		&policy.Active,
		&policyJSON,
	)

	if err != nil {
		return types.PluginPolicy{}, fmt.Errorf("failed to get policy: %w", err)
	}
	policy.Policy = json.RawMessage(policyJSON)

	return policy, nil
}

func (p *PostgresBackend) GetAllPluginPolicies(ctx context.Context, publicKey string, pluginType string) ([]types.PluginPolicy, error) {
	if p.pool == nil {
		return []types.PluginPolicy{}, fmt.Errorf("database pool is nil")
	}

	query := `
  	SELECT id, public_key, is_ecdsa, chain_code_hex, plugin_id, plugin_version, policy_version, plugin_type, signature, active, policy 
		FROM plugin_policies
		WHERE public_key = $1
		AND plugin_type = $2`

	rows, err := p.pool.Query(ctx, query, publicKey, pluginType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var policies []types.PluginPolicy
	for rows.Next() {
		var policy types.PluginPolicy
		err := rows.Scan(
			&policy.ID,
			&policy.PublicKey,
			&policy.IsEcdsa,
			&policy.ChainCodeHex,
			&policy.PluginID,
			&policy.PluginVersion,
			&policy.PolicyVersion,
			&policy.PluginType,
			&policy.Signature,
			&policy.Active,
			&policy.Policy,
		)
		if err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}

	return policies, nil
}

func (p *PostgresBackend) InsertPluginPolicyTx(ctx context.Context, dbTx pgx.Tx, policy types.PluginPolicy) (*types.PluginPolicy, error) {
	policyJSON, err := json.Marshal(policy.Policy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal policy: %w", err)
	}

	query := `
  	INSERT INTO plugin_policies (
      id, public_key, is_ecdsa, chain_code_hex, plugin_id, plugin_version, policy_version, plugin_type, signature, active, policy
    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    RETURNING id, public_key, is_ecdsa, chain_code_hex, plugin_id, plugin_version, policy_version, plugin_type, signature, active, policy
	`

	var insertedPolicy types.PluginPolicy
	err = dbTx.QueryRow(ctx, query,
		policy.ID,
		policy.PublicKey,
		policy.IsEcdsa,
		policy.ChainCodeHex,
		policy.PluginID,
		policy.PluginVersion,
		policy.PolicyVersion,
		policy.PluginType,
		policy.Signature,
		policy.Active,
		policyJSON,
	).Scan(
		&insertedPolicy.ID,
		&insertedPolicy.PublicKey,
		&insertedPolicy.IsEcdsa,
		&insertedPolicy.ChainCodeHex,
		&insertedPolicy.PluginID,
		&insertedPolicy.PluginVersion,
		&insertedPolicy.PolicyVersion,
		&insertedPolicy.PluginType,
		&insertedPolicy.Signature,
		&insertedPolicy.Active,
		&insertedPolicy.Policy,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert policy: %w", err)
	}

	return &insertedPolicy, nil
}

func (p *PostgresBackend) UpdatePluginPolicyTx(ctx context.Context, dbTx pgx.Tx, policy types.PluginPolicy) (*types.PluginPolicy, error) {
	policyJSON, err := json.Marshal(policy.Policy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal policy: %w", err)
	}

	// TODO: update other fields
	query := `
		UPDATE plugin_policies 
		SET public_key = $2, 
				plugin_type = $3, 
				signature = $4,
				active = $5,
				policy = $6
		WHERE id = $1
		RETURNING id, public_key, plugin_id, plugin_version, policy_version, plugin_type, signature, active, policy
	`

	var updatedPolicy types.PluginPolicy
	err = dbTx.QueryRow(ctx, query,
		policy.ID,
		policy.PublicKey,
		policy.PluginType,
		policy.Signature,
		policy.Active,
		policyJSON,
	).Scan(
		&updatedPolicy.ID,
		&updatedPolicy.PublicKey,
		&updatedPolicy.PluginID,
		&updatedPolicy.PluginVersion,
		&updatedPolicy.PolicyVersion,
		&updatedPolicy.PluginType,
		&updatedPolicy.Signature,
		&updatedPolicy.Active,
		&updatedPolicy.Policy,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("policy not found with ID: %s", policy.ID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update policy: %w", err)
	}

	return &updatedPolicy, nil
}

func (p *PostgresBackend) DeletePluginPolicyTx(ctx context.Context, dbTx pgx.Tx, id uuid.UUID) error {
	_, err := dbTx.Exec(ctx, `
	DELETE FROM transaction_history
	WHERE policy_id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("failed to delete transaction history: %w", err)
	}
	_, err = dbTx.Exec(ctx, `
	DELETE FROM time_triggers
	WHERE policy_id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("failed to delete time triggers: %w", err)
	}
	_, err = dbTx.Exec(ctx, `
	DELETE FROM plugin_policies
	WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	return nil
}

func (p *PostgresBackend) AddPluginPolicySync(ctx context.Context, dbTx pgx.Tx, policy itypes.PluginPolicySync) error {
	qry := `INSERT INTO plugin_policy_sync (id,policy_id,sync_type,signature, status, reason,plugin_id) values ($1, $2, $3, $4,$5,$6,$7)`
	_, err := dbTx.Exec(ctx, qry,
		policy.ID,
		policy.PolicyID,
		policy.SyncType,
		policy.Signature,
		policy.Status,
		policy.FailReason,
		policy.PluginID)
	if err != nil {
		return fmt.Errorf("failed to insert plugin policy sync: %w", err)
	}
	return nil
}

func (p *PostgresBackend) GetPluginPolicySync(ctx context.Context, id uuid.UUID) (*itypes.PluginPolicySync, error) {
	qry := `SELECT id, policy_id, sync_type,signature,status, reason,plugin_id FROM plugin_policy_sync WHERE id = $1`
	var policy itypes.PluginPolicySync
	err := p.pool.QueryRow(ctx, qry, id).Scan(
		&policy.ID,
		&policy.PolicyID,
		&policy.SyncType,
		&policy.Signature,
		&policy.Status,
		&policy.FailReason,
		&policy.PluginID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin policy sync: %w", err)
	}
	return &policy, nil
}

func (p *PostgresBackend) DeletePluginPolicySync(ctx context.Context, id uuid.UUID) error {
	qry := `DELETE FROM plugin_policy_sync WHERE id = $1`
	_, err := p.pool.Exec(ctx, qry, id)
	if err != nil {
		return fmt.Errorf("failed to delete plugin policy sync: %w", err)
	}
	return nil
}

func (p *PostgresBackend) GetUnFinishedPluginPolicySyncs(ctx context.Context) ([]itypes.PluginPolicySync, error) {
	qry := `SELECT id, policy_id,sync_type,signature, status, reason,plugin_id FROM plugin_policy_sync WHERE status != $1`
	rows, err := p.pool.Query(ctx, qry, itypes.Synced)
	if err != nil {
		return nil, fmt.Errorf("failed to get unfinished plugin policy syncs: %w", err)
	}
	defer rows.Close()

	var policies []itypes.PluginPolicySync
	for rows.Next() {
		var policy itypes.PluginPolicySync
		err := rows.Scan(
			&policy.ID,
			&policy.PolicyID,
			&policy.SyncType,
			&policy.Signature,
			&policy.Status,
			&policy.FailReason,
			&policy.PluginID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan plugin policy sync: %w", err)
		}
		policies = append(policies, policy)
	}
	return policies, nil
}

func (p *PostgresBackend) UpdatePluginPolicySync(ctx context.Context, dbTx pgx.Tx, policy itypes.PluginPolicySync) error {
	qry := `UPDATE plugin_policy_sync SET status = $1, reason = $2 WHERE id = $3`
	_, err := dbTx.Exec(ctx, qry,
		policy.Status,
		policy.FailReason,
		policy.ID)
	if err != nil {
		return fmt.Errorf("failed to update plugin policy sync: %w", err)
	}
	return nil
}
