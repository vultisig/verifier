package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/types"
)

func (p *PostgresBackend) GetPluginPolicy(ctx context.Context, id uuid.UUID) (*types.PluginPolicy, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	var policy types.PluginPolicy

	query := `
				SELECT id, public_key, plugin_id, plugin_version, policy_version, signature, active, recipe
        FROM plugin_policies 
        WHERE id = $1`

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
		var freq sql.NullString
		err := billingRows.Scan(
			&billing.ID,
			&billing.Type,
			&freq,
			&billing.StartDate,
			&billing.Amount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan billing info: %w", err)
		}
		if freq.Valid {
			billing.Frequency = freq.String
		}
		policy.Billing = append(policy.Billing, billing)
	}

	return &policy, nil
}

func (p *PostgresBackend) GetAllPluginPolicies(ctx context.Context, publicKey string, pluginID types.PluginID, take int, skip int) (*itypes.PluginPolicyPaginatedList, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
  	SELECT id, public_key, plugin_id, plugin_version, policy_version, signature, active, recipe,
		COUNT(*) OVER() AS total_count
		FROM plugin_policies
		WHERE public_key = $1
		AND plugin_id = $2
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
			var freq sql.NullString
			err := billingRows.Scan(
				&billing.ID,
				&billing.Type,
				&freq,
				&billing.StartDate,
				&billing.Amount,
			)
			if err != nil {
				billingRows.Close()
				return nil, fmt.Errorf("failed to scan billing info: %w", err)
			}
			if freq.Valid {
				billing.Frequency = freq.String
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

	billingQueryWithFrequency := `
	INSERT INTO plugin_policy_billing (id, plugin_policy_id, type, frequency, start_date, amount)
	VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, type, frequency, start_date, amount`
	billingQueryWithoutFrequency := `
	INSERT INTO plugin_policy_billing (id, plugin_policy_id, type, start_date, amount)
	VALUES ($1, $2, $3, $4, $5) RETURNING id, type, start_date, amount`
	for _, billing := range policy.Billing {
		var err error
		if billing.Frequency == "" {
			err = tx.QueryRow(ctx, billingQueryWithoutFrequency,
				billing.ID,
				policy.ID,
				billing.Type,
				billing.StartDate,
				billing.Amount,
			).Scan(
				&billing.ID,
				&billing.Type,
				&billing.StartDate,
				&billing.Amount,
			)
		} else {
			err = tx.QueryRow(ctx, billingQueryWithFrequency,
				billing.ID,
				policy.ID,
				billing.Type,
				billing.Frequency,
				billing.StartDate,
				billing.Amount,
			).Scan(
				&billing.ID,
				&billing.Type,
				&billing.Frequency,
				&billing.StartDate,
				&billing.Amount,
			)
		}
		if err != nil {
			fmt.Println("Error inserting billing policy XX:", err)
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
	DELETE FROM plugin_policies
	WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
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
	qry := `DELETE FROM plugin_policy_sync WHERE id = $1`
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
	deleteBilling := `DELETE FROM plugin_policy_billing WHERE plugin_policy_id IN (SELECT id FROM plugin_policies WHERE plugin_id = $1 AND public_key = $2)`
	_, err := dbTx.Exec(ctx, deleteBilling, pluginID, publicKey)
	if err != nil {
		return fmt.Errorf("failed to delete billing info for plugin %s: %w", pluginID, err)
	}
	query := `DELETE FROM plugin_policies WHERE plugin_id = $1 AND public_key = $2`
	_, err = dbTx.Exec(ctx, query, pluginID, publicKey)
	if err != nil {
		return fmt.Errorf("failed to delete all policies for plugin %s: %w", pluginID, err)
	}

	return nil
}
