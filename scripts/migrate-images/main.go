package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/storage"
	itypes "github.com/vultisig/verifier/internal/types"
)

const (
	verifierAssetsBase   = "https://raw.githubusercontent.com/vultisig/verifier/48ee5b48c5de5e64e35bbd3f5dc6b0deecd6de27/assets/plugins"
	marketplaceMediaBase = "https://raw.githubusercontent.com/vultisig/app-marketplace/main/public/media"
)

var (
	uploadedBy = flag.String("uploaded-by", "", "Public key to use as uploader (required)")
	dryRun     = flag.Bool("dry-run", false, "Print what would be migrated without actually migrating")
	verbose    = flag.Bool("verbose", false, "Enable verbose logging")
)

type pluginData struct {
	ID        string
	LogoURL   string
	BannerURL string
	MediaURLs []string
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

	plugins := getDefaultMapping()

	existingPlugins, err := getExistingPluginIDs(ctx, pool)
	if err != nil {
		logger.Fatalf("failed to get existing plugins: %v", err)
	}

	logger.Infof("found %d plugins in mapping, %d in database", len(plugins), len(existingPlugins))

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	var migratedLogo, migratedBanner, migratedMedia int
	var skippedLogo, skippedBanner, skippedMedia int
	var failedLogo, failedBanner, failedMedia int

	for _, p := range plugins {
		if !existingPlugins[string(p.ID)] {
			logger.Warnf("plugin %s not found in database, skipping", p.ID)
			continue
		}

		logger.Debugf("processing plugin: %s", p.ID)

		if p.LogoURL != "" {
			hasLogo, err := hasActiveImage(ctx, pool, p.ID, itypes.PluginImageTypeLogo)
			if err != nil {
				logger.Errorf("failed to check logo for plugin %s: %v", p.ID, err)
				failedLogo++
			} else if hasLogo {
				logger.Debugf("plugin %s already has logo, skipping", p.ID)
				skippedLogo++
			} else {
				err = migrateImage(ctx, logger, httpClient, pool, assetStorage, p.ID, p.LogoURL, itypes.PluginImageTypeLogo, 0)
				if err != nil {
					logger.Errorf("failed to migrate logo for plugin %s: %v", p.ID, err)
					failedLogo++
				} else {
					migratedLogo++
				}
			}
		}

		if p.BannerURL != "" {
			hasBanner, err := hasActiveImage(ctx, pool, p.ID, itypes.PluginImageTypeBanner)
			if err != nil {
				logger.Errorf("failed to check banner for plugin %s: %v", p.ID, err)
				failedBanner++
			} else if hasBanner {
				logger.Debugf("plugin %s already has banner, skipping", p.ID)
				skippedBanner++
			} else {
				err = migrateImage(ctx, logger, httpClient, pool, assetStorage, p.ID, p.BannerURL, itypes.PluginImageTypeBanner, 0)
				if err != nil {
					logger.Errorf("failed to migrate banner for plugin %s: %v", p.ID, err)
					failedBanner++
				} else {
					migratedBanner++
				}
			}
		}

		if len(p.MediaURLs) > 0 {
			hasMedia, err := hasActiveImage(ctx, pool, p.ID, itypes.PluginImageTypeMedia)
			if err != nil {
				logger.Errorf("failed to check media for plugin %s: %v", p.ID, err)
				failedMedia += len(p.MediaURLs)
			} else if hasMedia {
				logger.Debugf("plugin %s already has media, skipping", p.ID)
				skippedMedia += len(p.MediaURLs)
			} else {
				for i, mediaURL := range p.MediaURLs {
					err = migrateImage(ctx, logger, httpClient, pool, assetStorage, p.ID, mediaURL, itypes.PluginImageTypeMedia, i)
					if err != nil {
						logger.Errorf("failed to migrate media %d for plugin %s: %v", i, p.ID, err)
						failedMedia++
					} else {
						migratedMedia++
					}
				}
			}
		}
	}

	logger.Info("=== Migration Summary ===")
	logger.Infof("Logo:   migrated=%d, skipped=%d, failed=%d", migratedLogo, skippedLogo, failedLogo)
	logger.Infof("Banner: migrated=%d, skipped=%d, failed=%d", migratedBanner, skippedBanner, failedBanner)
	logger.Infof("Media:  migrated=%d, skipped=%d, failed=%d", migratedMedia, skippedMedia, failedMedia)
}

func getDefaultMapping() []pluginData {
	return []pluginData{
		{
			ID:        "vultisig-recurring-sends-0000",
			LogoURL:   verifierAssetsBase + "/recurring_send/icon.jpg",
			BannerURL: verifierAssetsBase + "/recurring_send/banner.jpg",
			MediaURLs: []string{
				marketplaceMediaBase + "/recurring-sends-img-01.jpg",
				marketplaceMediaBase + "/recurring-sends-img-02.jpg",
				marketplaceMediaBase + "/recurring-sends-img-03.jpg",
				marketplaceMediaBase + "/recurring-sends-img-04.jpg",
			},
		},
		{
			ID:        "vultisig-dca-0000",
			LogoURL:   verifierAssetsBase + "/recurring_swap/icon.jpg",
			BannerURL: verifierAssetsBase + "/recurring_swap/banner.jpg",
			MediaURLs: []string{
				marketplaceMediaBase + "/recurring-swaps-img-01.jpg",
				marketplaceMediaBase + "/recurring-swaps-img-02.jpg",
				marketplaceMediaBase + "/recurring-swaps-img-03.jpg",
				marketplaceMediaBase + "/recurring-swaps-img-04.jpg",
			},
		},
		{
			ID:        "vultisig-fees-feee",
			LogoURL:   verifierAssetsBase + "/payment/icon.jpg",
			BannerURL: verifierAssetsBase + "/payment/banner.jpg",
		},
	}
}

func getExistingPluginIDs(ctx context.Context, pool *pgxpool.Pool) (map[string]bool, error) {
	rows, err := pool.Query(ctx, "SELECT id FROM plugins")
	if err != nil {
		return nil, fmt.Errorf("failed to query plugins: %w", err)
	}
	defer rows.Close()

	result := make(map[string]bool)
	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("failed to scan: %w", err)
		}
		result[id] = true
	}
	return result, rows.Err()
}

func hasActiveImage(ctx context.Context, pool *pgxpool.Pool, pluginID string, imageType itypes.PluginImageType) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM plugin_images WHERE plugin_id = $1 AND image_type = $2 AND deleted = false AND visible = true)`
	var exists bool
	err := pool.QueryRow(ctx, query, pluginID, string(imageType)).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check existing image: %w", err)
	}
	return exists, nil
}

func migrateImage(ctx context.Context, logger *logrus.Logger, client *http.Client, pool *pgxpool.Pool, assetStorage storage.PluginAssetStorage, pluginID string, source string, imageType itypes.PluginImageType, imageOrder int) error {
	if *dryRun {
		logger.Infof("[DRY-RUN] would migrate %s for plugin %s from %s", imageType, pluginID, source)
		return nil
	}

	logger.Infof("migrating %s for plugin %s from %s", imageType, pluginID, source)

	data, contentType, err := downloadImage(ctx, client, source)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}

	ext := extensionFromContentType(contentType)
	if ext == "" {
		ext = extensionFromPath(source)
	}

	s3Key := fmt.Sprintf("plugins/%s/%s/%s%s", pluginID, imageType, uuid.New().String(), ext)

	err = assetStorage.Upload(ctx, s3Key, data, contentType)
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	filename := filepath.Base(source)
	filename = strings.TrimSuffix(filename, ext)

	insertQuery := `
		INSERT INTO plugin_images (plugin_id, image_type, s3_path, image_order, uploaded_by_public_key, content_type, filename)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = pool.Exec(ctx, insertQuery, pluginID, string(imageType), s3Key, imageOrder, *uploadedBy, contentType, filename)
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

func extensionFromPath(p string) string {
	ext := path.Ext(p)
	if ext == "" {
		return ".png"
	}
	if idx := strings.Index(ext, "?"); idx > 0 {
		ext = ext[:idx]
	}
	return ext
}
