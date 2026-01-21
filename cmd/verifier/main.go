package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/hibiken/asynq"

	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/api"
	"github.com/vultisig/verifier/internal/logging"
	internalMetrics "github.com/vultisig/verifier/internal/metrics"
	"github.com/vultisig/verifier/internal/storage"
	"github.com/vultisig/verifier/internal/storage/postgres"
	"github.com/vultisig/verifier/plugin/tx_indexer"
	tx_indexer_storage "github.com/vultisig/verifier/plugin/tx_indexer/pkg/storage"
	"github.com/vultisig/verifier/vault"
)

func main() {
	ctx := context.Background()

	cfg, err := config.ReadVerifierConfig()
	if err != nil {
		panic(err)
	}

	logger := logging.NewLogger(cfg.LogFormat)

	redisStorage, err := storage.NewRedisStorage(cfg.Redis)
	if err != nil {
		panic(err)
	}

	var redisConnOpt asynq.RedisConnOpt
	if cfg.Redis.URI != "" {
		redisConnOpt, err = asynq.ParseRedisURI(cfg.Redis.URI)
		if err != nil {
			panic(err)
		}
	} else {
		redisConnOpt = asynq.RedisClientOpt{
			Addr:     cfg.Redis.Host + ":" + cfg.Redis.Port,
			Username: cfg.Redis.User,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		}
	}

	client := asynq.NewClient(redisConnOpt)
	defer func() {
		if err := client.Close(); err != nil {
			fmt.Println("fail to close asynq client,", err)
		}
	}()

	inspector := asynq.NewInspector(redisConnOpt)
	vaultStorage, err := vault.NewBlockStorageImp(cfg.BlockStorage)
	if err != nil {
		panic(err)
	}

	db, err := postgres.NewPostgresBackend(cfg.Database.DSN, nil)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}

	txIndexerStore, err := tx_indexer_storage.NewPostgresTxIndexStore(ctx, cfg.Database.DSN)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}

	supportedChains, err := tx_indexer.Chains()
	if err != nil {
		logger.Fatalf("failed to get supported chains: %v", err)
	}

	txIndexerService := tx_indexer.NewService(
		logger,
		txIndexerStore,
		supportedChains,
	)

	// Initialize metrics based on configuration
	var httpMetrics *internalMetrics.HTTPMetrics
	if cfg.Metrics.Enabled {
		logger.Info("Metrics enabled, setting up Prometheus metrics")

		// Start metrics HTTP server with HTTP metrics
		services := []string{internalMetrics.ServiceHTTP}
		_ = internalMetrics.StartMetricsServer(internalMetrics.Config{
			Enabled: true,
			Host:    cfg.Metrics.Host,
			Port:    cfg.Metrics.Port,
		}, services, logger)

		// Create HTTP metrics implementation
		httpMetrics = internalMetrics.NewHTTPMetrics()
	} else {
		logger.Info("Verifier metrics disabled")
	}

	// Initialize plugin asset storage
	var assetStorage storage.PluginAssetStorage = storage.NewNoopPluginAssetStorage()
	if cfg.PluginAssets.IsConfigured() {
		assetStorage, err = storage.NewS3PluginAssetStorage(cfg.PluginAssets)
		if err != nil {
			logger.Warnf("Failed to initialize plugin asset storage: %v", err)
			assetStorage = storage.NewNoopPluginAssetStorage()
		}
	} else {
		missing := cfg.PluginAssets.Validate()
		logger.Infof("Plugin asset storage not configured, missing: %s — image uploads disabled", strings.Join(missing, ", "))
	}

	server := api.NewServer(
		*cfg,
		db,
		redisStorage,
		vaultStorage,
		assetStorage,
		client,
		inspector,
		cfg.Server.JWTSecret,
		txIndexerService,
		httpMetrics,
	)
	if err := server.StartServer(); err != nil {
		panic(err)
	}
}
