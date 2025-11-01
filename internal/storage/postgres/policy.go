package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/types"
)

func (p *PostgresBackend) GetPluginPolicy(ctx context.Context, id uuid.UUID) (*types.PluginPolicy, error) {
	if p.pool == nil {
		return nil, errors.New("database pool is nil")
	}

	var policy types.PluginPolicy

	query := `SELECT id, public_key, plugin_id, plugin_version, policy_version, signature, active, recipe
        FROM plugin_policies 
        WHERE id = $1 AND deleted = false`

	err := p.pool.QueryRow(ctx, query, id).Scan(
		&policy.ID,
		&policy.PublicKey,
		&policy.PluginID,
		&policy.PluginVersion,
		&policy.PolicyVersion,
		&policy.Signature,
		&policy.Active,
		&policy.Recipe,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	query = `SELECT id, type, frequency, start_date, amount FROM plugin_policy_billing WHERE plugin_policy_id = $1`
	billingRows, err := p.pool.Query(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get billing info: %w", err)
	}
	defer billingRows.Close()
	for billingRows.Next() {
		var billing types.BillingPolicy
		err := billingRows.Scan(
			&billing.ID,
			&billing.Type,
			&billing.Frequency,
			&billing.StartDate,
			&billing.Amount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan billing info: %w", err)
		}
		policy.Billing = append(policy.Billing, billing)
	}

	return &policy, nil
}

func (p *PostgresBackend) GetPluginPolicies(ctx context.Context, publicKey string, pluginIds []types.PluginID, includeInactive bool) ([]types.PluginPolicy, error) {
	var rows pgx.Rows
	var err error

	if len(pluginIds) == 0 {
		if !includeInactive {
			rows, err = p.pool.Query(ctx, `
SELECT id, public_key, plugin_id, plugin_version, policy_version, signature, active, recipe 
FROM plugin_policies 
WHERE public_key = $1 AND active = true AND deleted = false`, publicKey)
		} else {
			rows, err = p.pool.Query(ctx, `
SELECT id, public_key, plugin_id, plugin_version, policy_version, signature, active, recipe 
FROM plugin_policies 
WHERE public_key = $1 AND deleted = false`, publicKey)
		}
	} else {
		pids := []string{}
		for _, pid := range pluginIds {
			pids = append(pids, pid.String())
		}
		if !includeInactive {
			rows, err = p.pool.Query(ctx, `
SELECT id, public_key, plugin_id, plugin_version, policy_version, signature, active, recipe 
FROM plugin_policies 
WHERE public_key = $1 AND plugin_id = ANY($2) AND active = true AND deleted = false`, publicKey, pids)
		} else {
			rows, err = p.pool.Query(ctx, `
SELECT id, public_key, plugin_id, plugin_version, policy_version, signature, active, recipe 
FROM plugin_policies 
WHERE public_key = $1 AND plugin_id = ANY($2) AND deleted = false`, publicKey, pids)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get plugin policies: %w", err)
	}
	defer rows.Close()

	var policies []types.PluginPolicy
	for rows.Next() {
		var policy types.PluginPolicy
		err := rows.Scan(
			&policy.ID,
			&policy.PublicKey,
			&policy.PluginID,
			&policy.PluginVersion,
			&policy.PolicyVersion,
			&policy.Signature,
			&policy.Active,
			&policy.Recipe,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan plugin policy: %w", err)
		}
		policies = append(policies, policy)
	}

	return policies, nil
}

func (p *PostgresBackend) GetPluginInstallationsCount(ctx context.Context, pluginID types.PluginID) (itypes.PluginTotalCount, error) {
	if p.pool == nil {
		return itypes.PluginTotalCount{}, fmt.Errorf("database pool is nil")
	}

	query := `
	SELECT COUNT(DISTINCT public_key) AS total_count
	FROM plugin_policies
	WHERE plugin_id = $1`

	var totalCount int
	err := p.pool.QueryRow(ctx, query, pluginID).Scan(&totalCount)
	if err != nil {
		return itypes.PluginTotalCount{}, err
	}

	resp := itypes.PluginTotalCount{
		ID:         pluginID,
		TotalCount: totalCount,
	}
	return resp, nil
}

func (p *PostgresBackend) GetAllPluginPolicies(ctx context.Context, publicKey string, pluginID types.PluginID, take int, skip int, includeInactive bool) (*itypes.PluginPolicyPaginatedList, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
  	SELECT id, public_key, plugin_id, plugin_version, policy_version, signature, active, recipe,
		COUNT(*) OVER() AS total_count
		FROM plugin_policies
		WHERE public_key = $1
		AND plugin_id = $2 AND deleted = false`

	if !includeInactive {
		query += ` AND active = true`
	}

	query += `
		ORDER BY policy_version DESC
		LIMIT $3 OFFSET $4`

	rows, err := p.pool.Query(ctx, query, publicKey, pluginID, take, skip)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []types.PluginPolicy
	var totalCount int
	for rows.Next() {
		var policy types.PluginPolicy
		err := rows.Scan(
			&policy.ID,
			&policy.PublicKey,
			&policy.PluginID,
			&policy.PluginVersion,
			&policy.PolicyVersion,
			&policy.Signature,
			&policy.Active,
			&policy.Recipe,
			&totalCount,
		)
		if err != nil {
			return nil, err
		}

		billingQuery := `SELECT id, "type", frequency, start_date, amount FROM plugin_policy_billing WHERE plugin_policy_id = $1`
		billingRows, err := p.pool.Query(ctx, billingQuery, policy.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get billing info: %w", err)
		}
		for billingRows.Next() {
			var billing types.BillingPolicy
			err := billingRows.Scan(
				&billing.ID,
				&billing.Type,
				&billing.Frequency,
				&billing.StartDate,
				&billing.Amount,
			)
			if err != nil {
				billingRows.Close()
				return nil, fmt.Errorf("failed to scan billing info: %w", err)
			}
			policy.Billing = append(policy.Billing, billing)
		}
		billingRows.Close()
		policies = append(policies, policy)
	}

	dto := itypes.PluginPolicyPaginatedList{
		Policies:   policies,
		TotalCount: totalCount,
	}

	return &dto, nil
}

func (p *PostgresBackend) InsertPluginPolicyTx(ctx context.Context, dbTx pgx.Tx, policy types.PluginPolicy) (*types.PluginPolicy, error) {
	query := `
  	INSERT INTO plugin_policies (
      id, public_key, plugin_id, plugin_version, policy_version, signature, active, recipe
    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    RETURNING id, public_key, plugin_id, plugin_version, policy_version, signature, active, recipe
	`

	var insertedPolicy types.PluginPolicy
	tx, err := dbTx.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	err = tx.QueryRow(ctx, query,
		policy.ID,
		policy.PublicKey,
		policy.PluginID,
		policy.PluginVersion,
		policy.PolicyVersion,
		policy.Signature,
		policy.Active,
		policy.Recipe,
	).Scan(
		&insertedPolicy.ID,
		&insertedPolicy.PublicKey,
		&insertedPolicy.PluginID,
		&insertedPolicy.PluginVersion,
		&insertedPolicy.PolicyVersion,
		&insertedPolicy.Signature,
		&insertedPolicy.Active,
		&insertedPolicy.Recipe,
	)
	if err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("failed to insert policy: %w", err)
	}

	billingQuery := `
	INSERT INTO plugin_policy_billing (id, plugin_policy_id, type, frequency, start_date, amount, asset)
	VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, type, frequency, start_date, amount, asset`
	for _, billing := range policy.Billing {
		var err error
		if billing.Frequency == nil {
			err = tx.QueryRow(ctx, billingQuery,
				billing.ID,
				policy.ID,
				billing.Type,
				billing.Frequency,
				billing.StartDate,
				billing.Amount,
				billing.Asset,
			).Scan(
				&billing.ID,
				&billing.Type,
				&billing.Frequency,
				&billing.StartDate,
				&billing.Amount,
				&billing.Asset,
			)
		}
		if err != nil {
			tx.Rollback(ctx)
			return nil, fmt.Errorf("failed to insert billing policy: %w", err)
		}
		insertedPolicy.Billing = append(insertedPolicy.Billing, billing)
	}

	return &insertedPolicy, nil
}

func (p *PostgresBackend) UpdatePluginPolicyTx(ctx context.Context, dbTx pgx.Tx, policy types.PluginPolicy) (*types.PluginPolicy, error) {
	query := `
		UPDATE plugin_policies 
		SET public_key = $2, 
				plugin_id = $3, 
				signature = $4,
				active = $5,
				recipe = $6
		WHERE id = $1
		RETURNING id, public_key, plugin_id, plugin_version, policy_version, signature, active, recipe
	`

	var updatedPolicy types.PluginPolicy
	err := dbTx.QueryRow(ctx, query,
		policy.ID,
		policy.PublicKey,
		policy.PluginID,
		policy.Signature,
		policy.Active,
		policy.Recipe).Scan(
		&updatedPolicy.ID,
		&updatedPolicy.PublicKey,
		&updatedPolicy.PluginID,
		&updatedPolicy.PluginVersion,
		&updatedPolicy.PolicyVersion,
		&updatedPolicy.Signature,
		&updatedPolicy.Active,
		&updatedPolicy.Recipe,
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
	UPDATE plugin_policies
	SET deleted = true, active = false
	WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("failed to soft delete policy: %w", err)
	}

	return nil
}

func (p *PostgresBackend) AddPluginPolicySync(ctx context.Context, dbTx pgx.Tx, policy itypes.PluginPolicySync) error {
	qry := `INSERT INTO plugin_policy_sync (id, policy_id, sync_type, signature, status, reason, plugin_id) 
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
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
	qry := `SELECT id, policy_id, sync_type, signature, status, reason, plugin_id 
		FROM plugin_policy_sync 
		WHERE id = $1`
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
	qry := `UPDATE plugin_policies SET deleted = true WHERE id = $1`
	_, err := p.pool.Exec(ctx, qry, id)
	if err != nil {
		return fmt.Errorf("failed to delete plugin policy sync: %w", err)
	}
	return nil
}

func (p *PostgresBackend) GetUnFinishedPluginPolicySyncs(ctx context.Context) ([]itypes.PluginPolicySync, error) {
	qry := `SELECT id, policy_id, sync_type, signature, status, reason, plugin_id 
		FROM plugin_policy_sync 
		WHERE status != $1`
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
	qry := `UPDATE plugin_policy_sync 
		SET status = $1, 
				reason = $2 
		WHERE id = $3`
	_, err := dbTx.Exec(ctx, qry,
		policy.Status,
		policy.FailReason,
		policy.ID)
	if err != nil {
		return fmt.Errorf("failed to update plugin policy sync: %w", err)
	}
	return nil
}

func (p *PostgresBackend) DeleteAllPolicies(ctx context.Context, dbTx pgx.Tx, pluginID types.PluginID, publicKey string) error {
	query := `UPDATE plugin_policies SET deleted = true, active = false WHERE plugin_id = $1 AND public_key = $2 AND deleted = false`
	_, err := dbTx.Exec(ctx, query, pluginID, publicKey)
	if err != nil {
		return fmt.Errorf("failed to delete all policies for plugin %s: %w", pluginID, err)
	}

	return nil
}
