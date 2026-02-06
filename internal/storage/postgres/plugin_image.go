package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/types"
)

func (p *PostgresBackend) CreatePluginImage(ctx context.Context, params itypes.PluginImageCreateParams) (*itypes.PluginImageRecord, error) {
	query := `
		INSERT INTO plugin_images (plugin_id, image_type, s3_path, image_order, uploaded_by_public_key)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, plugin_id, image_type, s3_path, image_order, uploaded_by_public_key, visible, deleted, created_at, updated_at
	`

	var record itypes.PluginImageRecord
	err := p.pool.QueryRow(ctx, query,
		params.PluginID,
		params.ImageType,
		params.S3Path,
		params.ImageOrder,
		params.UploadedByPublicKey,
	).Scan(
		&record.ID,
		&record.PluginID,
		&record.ImageType,
		&record.S3Path,
		&record.ImageOrder,
		&record.UploadedByPublicKey,
		&record.Visible,
		&record.Deleted,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin image: %w", err)
	}

	return &record, nil
}

func (p *PostgresBackend) GetPluginImagesByPluginIDs(ctx context.Context, pluginIDs []types.PluginID) ([]itypes.PluginImageRecord, error) {
	if len(pluginIDs) == 0 {
		return []itypes.PluginImageRecord{}, nil
	}

	ids := make([]string, 0, len(pluginIDs))
	for _, id := range pluginIDs {
		ids = append(ids, string(id))
	}

	query := `
		SELECT id, plugin_id, image_type, s3_path, image_order, uploaded_by_public_key, visible, deleted, created_at, updated_at
		FROM plugin_images
		WHERE plugin_id::text = ANY($1::text[]) AND deleted = false AND visible = true
		ORDER BY plugin_id, image_type, image_order ASC
	`

	rows, err := p.pool.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to query plugin images: %w", err)
	}
	defer rows.Close()

	records := make([]itypes.PluginImageRecord, 0, len(pluginIDs))
	for rows.Next() {
		var r itypes.PluginImageRecord
		err := rows.Scan(
			&r.ID,
			&r.PluginID,
			&r.ImageType,
			&r.S3Path,
			&r.ImageOrder,
			&r.UploadedByPublicKey,
			&r.Visible,
			&r.Deleted,
			&r.CreatedAt,
			&r.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan plugin image: %w", err)
		}
		records = append(records, r)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating plugin images: %w", err)
	}

	return records, nil
}

func (p *PostgresBackend) GetPluginImageByType(ctx context.Context, pluginID types.PluginID, imageType itypes.PluginImageType) (*itypes.PluginImageRecord, error) {
	query := `
		SELECT id, plugin_id, image_type, s3_path, image_order, uploaded_by_public_key, visible, deleted, created_at, updated_at
		FROM plugin_images
		WHERE plugin_id = $1 AND image_type = $2 AND deleted = false AND visible = true
		ORDER BY updated_at DESC
		LIMIT 1
	`

	var r itypes.PluginImageRecord
	err := p.pool.QueryRow(ctx, query, pluginID, imageType).Scan(
		&r.ID,
		&r.PluginID,
		&r.ImageType,
		&r.S3Path,
		&r.ImageOrder,
		&r.UploadedByPublicKey,
		&r.Visible,
		&r.Deleted,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get plugin image by type: %w", err)
	}

	return &r, nil
}

func (p *PostgresBackend) SoftDeletePluginImage(ctx context.Context, pluginID types.PluginID, imageID uuid.UUID) (string, error) {
	query := `UPDATE plugin_images SET deleted = true, visible = false, updated_at = NOW() WHERE plugin_id = $1 AND id = $2 AND deleted = false AND visible = true RETURNING s3_path`
	var s3Path string
	err := p.pool.QueryRow(ctx, query, pluginID, imageID).Scan(&s3Path)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("plugin image not found: %s", imageID)
		}
		return "", fmt.Errorf("failed to soft delete plugin image: %w", err)
	}
	return s3Path, nil
}

func (p *PostgresBackend) ReplacePluginImage(ctx context.Context, pluginID types.PluginID, imageType itypes.PluginImageType, s3Path string, uploadedBy string) (*itypes.PluginImageRecord, error) {
	if imageType == itypes.PluginImageTypeMedia {
		return nil, fmt.Errorf("ReplacePluginImage not valid for media type, use CreatePluginImage instead")
	}

	var result *itypes.PluginImageRecord

	err := p.WithTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		softDeleteQuery := `
			UPDATE plugin_images
			SET deleted = true, visible = false, updated_at = NOW()
			WHERE plugin_id = $1 AND image_type = $2 AND deleted = false AND visible = true
		`
		_, err := tx.Exec(ctx, softDeleteQuery, pluginID, imageType)
		if err != nil {
			return fmt.Errorf("failed to soft delete existing image: %w", err)
		}

		insertQuery := `
			INSERT INTO plugin_images (plugin_id, image_type, s3_path, image_order, uploaded_by_public_key)
			VALUES ($1, $2, $3, 0, $4)
			RETURNING id, plugin_id, image_type, s3_path, image_order, uploaded_by_public_key, visible, deleted, created_at, updated_at
		`
		var record itypes.PluginImageRecord
		err = tx.QueryRow(ctx, insertQuery, pluginID, imageType, s3Path, uploadedBy).Scan(
			&record.ID,
			&record.PluginID,
			&record.ImageType,
			&record.S3Path,
			&record.ImageOrder,
			&record.UploadedByPublicKey,
			&record.Visible,
			&record.Deleted,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert new image: %w", err)
		}

		result = &record
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (p *PostgresBackend) GetNextMediaOrder(ctx context.Context, pluginID types.PluginID) (int, error) {
	query := `
		SELECT COALESCE(MAX(image_order) + 1, 0)
		FROM plugin_images
		WHERE plugin_id = $1 AND image_type = 'media' AND deleted = false AND visible = true
	`

	var nextOrder int
	err := p.pool.QueryRow(ctx, query, pluginID).Scan(&nextOrder)
	if err != nil {
		return 0, fmt.Errorf("failed to get next media order: %w", err)
	}

	return nextOrder, nil
}
