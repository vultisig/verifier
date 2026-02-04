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
		INSERT INTO plugin_images (plugin_id, image_type, s3_path, image_order, uploaded_by_public_key, content_type, filename, visible)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, plugin_id, image_type, s3_path, image_order, uploaded_by_public_key, visible, deleted, content_type, filename, created_at, updated_at
	`

	var record itypes.PluginImageRecord
	err := p.pool.QueryRow(ctx, query,
		params.PluginID,
		params.ImageType,
		params.S3Path,
		params.ImageOrder,
		params.UploadedByPublicKey,
		params.ContentType,
		params.Filename,
		params.Visible,
	).Scan(
		&record.ID,
		&record.PluginID,
		&record.ImageType,
		&record.S3Path,
		&record.ImageOrder,
		&record.UploadedByPublicKey,
		&record.Visible,
		&record.Deleted,
		&record.ContentType,
		&record.Filename,
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
		SELECT id, plugin_id, image_type, s3_path, image_order, uploaded_by_public_key, visible, deleted, content_type, filename, created_at, updated_at
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
			&r.ContentType,
			&r.Filename,
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
		SELECT id, plugin_id, image_type, s3_path, image_order, uploaded_by_public_key, visible, deleted, content_type, filename, created_at, updated_at
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
		&r.ContentType,
		&r.Filename,
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
	query := `UPDATE plugin_images SET deleted = true, visible = false, updated_at = NOW() WHERE plugin_id = $1 AND id = $2 AND deleted = false RETURNING s3_path`
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

func (p *PostgresBackend) SoftDeletePluginImageTx(ctx context.Context, tx pgx.Tx, pluginID types.PluginID, imageID uuid.UUID) (string, error) {
	query := `UPDATE plugin_images SET deleted = true, visible = false, updated_at = NOW() WHERE plugin_id = $1 AND id = $2 AND deleted = false RETURNING s3_path`
	var s3Path string
	err := tx.QueryRow(ctx, query, pluginID, imageID).Scan(&s3Path)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("plugin image not found: %s", imageID)
		}
		return "", fmt.Errorf("failed to soft delete plugin image: %w", err)
	}
	return s3Path, nil
}

func (p *PostgresBackend) ReplacePluginImage(ctx context.Context, pluginID types.PluginID, imageType itypes.PluginImageType, s3Path, contentType, filename, uploadedBy string) (*itypes.PluginImageRecord, error) {
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
			INSERT INTO plugin_images (plugin_id, image_type, s3_path, image_order, uploaded_by_public_key, content_type, filename, visible)
			VALUES ($1, $2, $3, 0, $4, $5, $6, true)
			RETURNING id, plugin_id, image_type, s3_path, image_order, uploaded_by_public_key, visible, deleted, content_type, filename, created_at, updated_at
		`
		var record itypes.PluginImageRecord
		err = tx.QueryRow(ctx, insertQuery, pluginID, imageType, s3Path, uploadedBy, contentType, filename).Scan(
			&record.ID,
			&record.PluginID,
			&record.ImageType,
			&record.S3Path,
			&record.ImageOrder,
			&record.UploadedByPublicKey,
			&record.Visible,
			&record.Deleted,
			&record.ContentType,
			&record.Filename,
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

func (p *PostgresBackend) GetNextMediaOrderTx(ctx context.Context, tx pgx.Tx, pluginID types.PluginID) (int, error) {
	query := `
		SELECT COALESCE(MAX(image_order) + 1, 0)
		FROM plugin_images
		WHERE plugin_id = $1 AND image_type = 'media' AND deleted = false
	`

	var nextOrder int
	err := tx.QueryRow(ctx, query, pluginID).Scan(&nextOrder)
	if err != nil {
		return 0, fmt.Errorf("failed to get next media order: %w", err)
	}

	return nextOrder, nil
}

func (p *PostgresBackend) LockPluginForUpdate(ctx context.Context, tx pgx.Tx, pluginID types.PluginID) error {
	_, err := tx.Exec(ctx, `SELECT 1 FROM plugins WHERE id = $1 FOR UPDATE`, pluginID)
	if err != nil {
		return fmt.Errorf("failed to lock plugin: %w", err)
	}
	return nil
}

func (p *PostgresBackend) CountVisibleMediaImages(ctx context.Context, tx pgx.Tx, pluginID types.PluginID) (int, error) {
	query := `SELECT COUNT(*) FROM plugin_images WHERE plugin_id = $1 AND image_type = 'media' AND visible = true AND deleted = false`
	var count int
	err := tx.QueryRow(ctx, query, pluginID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count visible media images: %w", err)
	}
	return count, nil
}

func (p *PostgresBackend) CreatePendingPluginImage(ctx context.Context, tx pgx.Tx, params itypes.PluginImageCreateParams) (*itypes.PluginImageRecord, error) {
	query := `
		INSERT INTO plugin_images (id, plugin_id, image_type, s3_path, image_order, uploaded_by_public_key, content_type, filename, visible)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, false)
		RETURNING id, plugin_id, image_type, s3_path, image_order, uploaded_by_public_key, visible, deleted, content_type, filename, created_at, updated_at
	`

	var record itypes.PluginImageRecord
	err := tx.QueryRow(ctx, query,
		params.ID,
		params.PluginID,
		params.ImageType,
		params.S3Path,
		params.ImageOrder,
		params.UploadedByPublicKey,
		params.ContentType,
		params.Filename,
	).Scan(
		&record.ID,
		&record.PluginID,
		&record.ImageType,
		&record.S3Path,
		&record.ImageOrder,
		&record.UploadedByPublicKey,
		&record.Visible,
		&record.Deleted,
		&record.ContentType,
		&record.Filename,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create pending plugin image: %w", err)
	}

	return &record, nil
}

func (p *PostgresBackend) GetPluginImageByID(ctx context.Context, pluginID types.PluginID, imageID uuid.UUID) (*itypes.PluginImageRecord, error) {
	query := `
		SELECT id, plugin_id, image_type, s3_path, image_order, uploaded_by_public_key, visible, deleted, content_type, filename, created_at, updated_at
		FROM plugin_images
		WHERE plugin_id = $1 AND id = $2
	`

	var r itypes.PluginImageRecord
	err := p.pool.QueryRow(ctx, query, pluginID, imageID).Scan(
		&r.ID,
		&r.PluginID,
		&r.ImageType,
		&r.S3Path,
		&r.ImageOrder,
		&r.UploadedByPublicKey,
		&r.Visible,
		&r.Deleted,
		&r.ContentType,
		&r.Filename,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get plugin image by id: %w", err)
	}

	return &r, nil
}

type ListPluginImagesParams struct {
	PluginID       types.PluginID
	ImageType      *itypes.PluginImageType
	IncludeHidden  bool
	IncludeDeleted bool
}

func (p *PostgresBackend) ListPluginImages(ctx context.Context, params ListPluginImagesParams) ([]itypes.PluginImageRecord, error) {
	query := `
		SELECT id, plugin_id, image_type, s3_path, image_order, uploaded_by_public_key, visible, deleted, content_type, filename, created_at, updated_at
		FROM plugin_images
		WHERE plugin_id = $1
	`
	args := []interface{}{params.PluginID}
	argIdx := 2

	if !params.IncludeDeleted {
		query += " AND deleted = false"
	}
	if !params.IncludeHidden {
		query += " AND visible = true"
	}
	if params.ImageType != nil {
		query += fmt.Sprintf(" AND image_type = $%d", argIdx)
		args = append(args, *params.ImageType)
	}

	query += " ORDER BY image_type, image_order ASC"

	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list plugin images: %w", err)
	}
	defer rows.Close()

	var records []itypes.PluginImageRecord
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
			&r.ContentType,
			&r.Filename,
			&r.CreatedAt,
			&r.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan plugin image: %w", err)
		}
		records = append(records, r)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating plugin images: %w", err)
	}

	return records, nil
}

func (p *PostgresBackend) ConfirmPluginImage(ctx context.Context, tx pgx.Tx, pluginID types.PluginID, imageID uuid.UUID) (*itypes.PluginImageRecord, error) {
	var imageType string
	err := tx.QueryRow(ctx, `SELECT image_type FROM plugin_images WHERE id = $1 AND plugin_id = $2 AND visible = false AND deleted = false`, imageID, pluginID).Scan(&imageType)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("pending image not found")
		}
		return nil, fmt.Errorf("failed to get pending image type: %w", err)
	}

	if imageType != "media" {
		_, err = tx.Exec(ctx, `
			UPDATE plugin_images
			SET deleted = true, visible = false, updated_at = NOW()
			WHERE plugin_id = $1 AND image_type = $2 AND deleted = false AND visible = true AND id <> $3
		`, pluginID, imageType, imageID)
		if err != nil {
			return nil, fmt.Errorf("failed to soft delete existing singleton: %w", err)
		}
	}

	var r itypes.PluginImageRecord
	err = tx.QueryRow(ctx, `
		UPDATE plugin_images
		SET visible = true, updated_at = NOW()
		WHERE id = $1 AND plugin_id = $2 AND visible = false AND deleted = false
		RETURNING id, plugin_id, image_type, s3_path, image_order, uploaded_by_public_key, visible, deleted, content_type, filename, created_at, updated_at
	`, imageID, pluginID).Scan(
		&r.ID,
		&r.PluginID,
		&r.ImageType,
		&r.S3Path,
		&r.ImageOrder,
		&r.UploadedByPublicKey,
		&r.Visible,
		&r.Deleted,
		&r.ContentType,
		&r.Filename,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("pending image not found or already confirmed")
		}
		return nil, fmt.Errorf("failed to confirm plugin image: %w", err)
	}

	return &r, nil
}

func (p *PostgresBackend) UpdatePluginImage(ctx context.Context, pluginID types.PluginID, imageID uuid.UUID, visible *bool, imageOrder *int) (*itypes.PluginImageRecord, error) {
	query := `
		UPDATE plugin_images
		SET visible = COALESCE($3, visible),
		    image_order = COALESCE($4, image_order),
		    updated_at = NOW()
		WHERE plugin_id = $1 AND id = $2 AND deleted = false
		RETURNING id, plugin_id, image_type, s3_path, image_order, uploaded_by_public_key, visible, deleted, content_type, filename, created_at, updated_at
	`

	var r itypes.PluginImageRecord
	err := p.pool.QueryRow(ctx, query, pluginID, imageID, visible, imageOrder).Scan(
		&r.ID,
		&r.PluginID,
		&r.ImageType,
		&r.S3Path,
		&r.ImageOrder,
		&r.UploadedByPublicKey,
		&r.Visible,
		&r.Deleted,
		&r.ContentType,
		&r.Filename,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("image not found")
		}
		return nil, fmt.Errorf("failed to update plugin image: %w", err)
	}

	return &r, nil
}

func (p *PostgresBackend) ReorderMediaImages(ctx context.Context, tx pgx.Tx, pluginID types.PluginID, imageIDs []uuid.UUID) error {
	if len(imageIDs) == 0 {
		return nil
	}

	seen := make(map[uuid.UUID]bool)
	for _, id := range imageIDs {
		if seen[id] {
			return fmt.Errorf("duplicate image id: %s", id)
		}
		seen[id] = true
	}

	stageQuery := `
		WITH ranked AS (
			SELECT unnest($2::uuid[]) AS id, generate_series(1, array_length($2::uuid[], 1)) AS rn
		)
		UPDATE plugin_images pi
		SET image_order = -ranked.rn, updated_at = NOW()
		FROM ranked
		WHERE pi.plugin_id = $1
		  AND pi.id = ranked.id
		  AND pi.image_type = 'media'
		  AND pi.deleted = false
		  AND pi.visible = true
	`
	result, err := tx.Exec(ctx, stageQuery, pluginID, imageIDs)
	if err != nil {
		return fmt.Errorf("failed to stage reorder: %w", err)
	}
	if int(result.RowsAffected()) != len(imageIDs) {
		return fmt.Errorf("some images not found or not media type")
	}

	finalQuery := `
		WITH ranked AS (
			SELECT unnest($2::uuid[]) AS id, generate_series(0, array_length($2::uuid[], 1) - 1) AS ord
		)
		UPDATE plugin_images pi
		SET image_order = ranked.ord, updated_at = NOW()
		FROM ranked
		WHERE pi.plugin_id = $1
		  AND pi.id = ranked.id
		  AND pi.image_type = 'media'
		  AND pi.deleted = false
		  AND pi.visible = true
	`
	_, err = tx.Exec(ctx, finalQuery, pluginID, imageIDs)
	if err != nil {
		return fmt.Errorf("failed to finalize reorder: %w", err)
	}

	return nil
}
