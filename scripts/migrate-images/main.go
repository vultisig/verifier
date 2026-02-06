package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/storage"
	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/types"
)

var (
	uploadedBy = flag.String("uploaded-by", "", "Public key to use as uploader (required)")
	dryRun     = flag.Bool("dry-run", false, "Print what would be migrated without actually migrating")
	verbose    = flag.Bool("verbose", false, "Enable verbose logging")
)

type legacyPluginData struct {
	ID           types.PluginID
	LogoURL      string
	ThumbnailURL string
	Images       []string
}

func main() {
	flag.Parse()

	if *uploadedBy == "" {
		envKey := os.Getenv("MIGRATE_IMAGES_UPLOADED_BY")
		if envKey != "" {
			*uploadedBy = envKey
		} else {
			fmt.Fprintln(os.Stderr, "Error: --uploaded-by flag or MIGRATE_IMAGES_UPLOADED_BY env var is required")
			flag.Usage()
			os.Exit(1)
		}
	}

	logger := logrus.New()
	if *verbose {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	ctx := context.Background()

	cfg, err := config.ReadVerifierConfig()
	if err != nil {
		logger.Fatalf("failed to read config: %v", err)
	}

	if !cfg.PluginAssets.IsConfigured() {
		logger.Fatal("plugin assets S3 not configured")
	}

	pool, err := pgxpool.New(ctx, cfg.Database.DSN)
	if err != nil {
		logger.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	assetStorage, err := storage.NewS3PluginAssetStorage(cfg.PluginAssets)
	if err != nil {
		logger.Fatalf("failed to create S3 storage: %v", err)
	}

	plugins, err := fetchLegacyPluginData(ctx, pool)
	if err != nil {
		logger.Fatalf("failed to fetch plugins: %v", err)
	}

	logger.Infof("found %d plugins to process", len(plugins))

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	var migratedLogo, migratedThumb, migratedMedia int
	var skippedLogo, skippedThumb, skippedMedia int
	var failedLogo, failedThumb, failedMedia int

	for _, p := range plugins {
		logger.Debugf("processing plugin: %s", p.ID)

		hasLogo, err := hasActiveImage(ctx, pool, p.ID, itypes.PluginImageTypeLogo)
		if err != nil {
			logger.Errorf("failed to check logo for plugin %s: %v", p.ID, err)
			continue
		}
		if hasLogo {
			logger.Debugf("plugin %s already has logo, skipping", p.ID)
			skippedLogo++
		} else if p.LogoURL != "" {
			err = migrateImage(ctx, logger, httpClient, pool, assetStorage, p.ID, p.LogoURL, itypes.PluginImageTypeLogo, 0)
			if err != nil {
				logger.Errorf("failed to migrate logo for plugin %s: %v", p.ID, err)
				failedLogo++
			} else {
				migratedLogo++
			}
		}

		hasThumb, err := hasActiveImage(ctx, pool, p.ID, itypes.PluginImageTypeThumbnail)
		if err != nil {
			logger.Errorf("failed to check thumbnail for plugin %s: %v", p.ID, err)
			continue
		}
		if hasThumb {
			logger.Debugf("plugin %s already has thumbnail, skipping", p.ID)
			skippedThumb++
		} else if p.ThumbnailURL != "" {
			err = migrateImage(ctx, logger, httpClient, pool, assetStorage, p.ID, p.ThumbnailURL, itypes.PluginImageTypeThumbnail, 0)
			if err != nil {
				logger.Errorf("failed to migrate thumbnail for plugin %s: %v", p.ID, err)
				failedThumb++
			} else {
				migratedThumb++
			}
		}

		hasMedia, err := hasActiveImage(ctx, pool, p.ID, itypes.PluginImageTypeMedia)
		if err != nil {
			logger.Errorf("failed to check media for plugin %s: %v", p.ID, err)
			continue
		}
		if hasMedia {
			logger.Debugf("plugin %s already has media images, skipping", p.ID)
			skippedMedia += len(p.Images)
		} else if len(p.Images) > 0 {
			for i, imgURL := range p.Images {
				err = migrateImage(ctx, logger, httpClient, pool, assetStorage, p.ID, imgURL, itypes.PluginImageTypeMedia, i)
				if err != nil {
					logger.Errorf("failed to migrate media image %d for plugin %s: %v", i, p.ID, err)
					failedMedia++
				} else {
					migratedMedia++
				}
			}
		}
	}

	logger.Info("=== Migration Summary ===")
	logger.Infof("Logo:      migrated=%d, skipped=%d, failed=%d", migratedLogo, skippedLogo, failedLogo)
	logger.Infof("Thumbnail: migrated=%d, skipped=%d, failed=%d", migratedThumb, skippedThumb, failedThumb)
	logger.Infof("Media:     migrated=%d, skipped=%d, failed=%d", migratedMedia, skippedMedia, failedMedia)
}

func fetchLegacyPluginData(ctx context.Context, pool *pgxpool.Pool) ([]legacyPluginData, error) {
	query := `SELECT id, COALESCE(logo_url, ''), COALESCE(thumbnail_url, ''), COALESCE(images, '[]'::jsonb) FROM plugins`
	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query plugins: %w", err)
	}
	defer rows.Close()

	var result []legacyPluginData
	for rows.Next() {
		var p legacyPluginData
		var imagesJSON []byte
		err = rows.Scan(&p.ID, &p.LogoURL, &p.ThumbnailURL, &imagesJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan plugin: %w", err)
		}
		if len(imagesJSON) > 0 {
			err = json.Unmarshal(imagesJSON, &p.Images)
			if err != nil {
				p.Images = nil
			}
		}
		result = append(result, p)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating plugins: %w", err)
	}

	return result, nil
}

func hasActiveImage(ctx context.Context, pool *pgxpool.Pool, pluginID types.PluginID, imageType itypes.PluginImageType) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM plugin_images WHERE plugin_id = $1 AND image_type = $2 AND deleted = false AND visible = true)`
	var exists bool
	err := pool.QueryRow(ctx, query, pluginID, string(imageType)).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check existing image: %w", err)
	}
	return exists, nil
}

func migrateImage(ctx context.Context, logger *logrus.Logger, client *http.Client, pool *pgxpool.Pool, assetStorage storage.PluginAssetStorage, pluginID types.PluginID, sourceURL string, imageType itypes.PluginImageType, imageOrder int) error {
	if *dryRun {
		logger.Infof("[DRY-RUN] would migrate %s for plugin %s from %s", imageType, pluginID, sourceURL)
		return nil
	}

	logger.Infof("migrating %s for plugin %s from %s", imageType, pluginID, sourceURL)

	data, contentType, err := downloadImage(ctx, client, sourceURL)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}

	ext := extensionFromContentType(contentType)
	if ext == "" {
		ext = extensionFromURL(sourceURL)
	}

	s3Key := fmt.Sprintf("plugins/%s/%s/%s%s", pluginID, imageType, uuid.New().String(), ext)

	err = assetStorage.Upload(ctx, s3Key, data, contentType)
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	filename := path.Base(sourceURL)
	filename = strings.TrimSuffix(filename, ext)

	insertQuery := `
		INSERT INTO plugin_images (plugin_id, image_type, s3_path, image_order, uploaded_by_public_key, content_type, filename)
		VALUES ($1, $2, $3, $4, $5, 'image/jpeg', $6)
	`
	_, err = pool.Exec(ctx, insertQuery, pluginID, string(imageType), s3Key, imageOrder, *uploadedBy, filename)
	if err != nil {
		delErr := assetStorage.Delete(ctx, s3Key)
		if delErr != nil {
			logger.Warnf("failed to cleanup S3 after DB insert failure: %v", delErr)
		}
		return fmt.Errorf("failed to insert record: %w", err)
	}

	logger.Infof("migrated %s for plugin %s -> %s", imageType, pluginID, s3Key)
	return nil
}

func downloadImage(ctx context.Context, client *http.Client, url string) ([]byte, string, error) {
	var resp *http.Response
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create request: %w", err)
		}
		resp, lastErr = client.Do(req)
		if lastErr == nil && resp.StatusCode == http.StatusOK {
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	if lastErr != nil {
		return nil, "", fmt.Errorf("failed to download after retries: %w", lastErr)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	return data, contentType, nil
}

func extensionFromContentType(contentType string) string {
	switch {
	case strings.Contains(contentType, "image/png"):
		return ".png"
	case strings.Contains(contentType, "image/jpeg"):
		return ".jpg"
	case strings.Contains(contentType, "image/gif"):
		return ".gif"
	case strings.Contains(contentType, "image/webp"):
		return ".webp"
	case strings.Contains(contentType, "image/svg"):
		return ".svg"
	default:
		return ""
	}
}

func extensionFromURL(url string) string {
	ext := path.Ext(url)
	if ext == "" {
		return ".png"
	}
	if idx := strings.Index(ext, "?"); idx > 0 {
		ext = ext[:idx]
	}
	return ext
}
