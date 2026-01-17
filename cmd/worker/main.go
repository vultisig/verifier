package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"

	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/fee_manager"
	"github.com/vultisig/verifier/internal/health"
	"github.com/vultisig/verifier/internal/logging"
	internalMetrics "github.com/vultisig/verifier/internal/metrics"
	"github.com/vultisig/verifier/internal/safety"
	"github.com/vultisig/verifier/internal/service"
	"github.com/vultisig/verifier/internal/storage/postgres"
	"github.com/vultisig/verifier/plugin/tasks"
	"github.com/vultisig/verifier/plugin/tx_indexer"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/storage"
	"github.com/vultisig/verifier/vault"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	cfg, err := config.GetConfigure()
	if err != nil {
		panic(err)
	}

	logger := logging.NewLogger(cfg.LogFormat)

	// Start health server for K8s probes
	healthServer := health.New(cfg.HealthPort)
	go func() {
		if err := healthServer.Start(ctx, logger); err != nil {
			logger.Errorf("health server error: %v", err)
		}
	}()

	redisCfg := cfg.Redis
	var redisConnOpt asynq.RedisConnOpt
	if redisCfg.URI != "" {
		redisConnOpt, err = asynq.ParseRedisURI(redisCfg.URI)
		if err != nil {
			panic(err)
		}
	} else {
		redisConnOpt = asynq.RedisClientOpt{
			Addr:     redisCfg.Host + ":" + redisCfg.Port,
			Username: redisCfg.User,
			Password: redisCfg.Password,
			DB:       redisCfg.DB,
		}
	}
	client := asynq.NewClient(redisConnOpt)
	vaultStorage, err := vault.NewBlockStorageImp(cfg.BlockStorage)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize vault storage: %v", err))
	}

	backendDB, err := postgres.NewPostgresBackend(cfg.Database.DSN, nil, "")
	if err != nil {
		panic(fmt.Sprintf("failed to initialize database: %v", err))
	}

	policyService, err := service.NewPolicyService(
		backendDB,
		nil, // No syncer needed for async operations
	)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize policy service: %v", err))
	}

	srv := asynq.NewServer(
		redisConnOpt,
		asynq.Config{
			Logger:      logger,
			Concurrency: 10,
			Queues: map[string]int{
				tasks.QUEUE_NAME:         10,
				vault.EmailQueueName:     100,
				"scheduled_plugin_queue": 10,
			},
		},
	)

	txIndexerStore, err := storage.NewPostgresTxIndexStore(ctx, cfg.Database.DSN)
	if err != nil {
		panic(fmt.Sprintf("storage.NewPostgresTxIndexStore: %v", err))
	}

	chains, err := tx_indexer.Chains()
	if err != nil {
		panic(fmt.Errorf("failed to initialize supported chains: %w", err))
	}

	txIndexerService := tx_indexer.NewService(
		logger,
		txIndexerStore,
		chains,
	)

	safetyMgm := safety.NewManager(backendDB, logger)

	vaultMgmService, err := vault.NewManagementService(
		cfg.VaultService,
		client,
		vaultStorage,
		txIndexerService,
		safetyMgm,
	)

	feeMgmService := fee_manager.NewFeeManagementService(
		logger,
		backendDB,
		vaultMgmService,
	)

	// Initialize metrics based on configuration
	var workerMetrics internalMetrics.WorkerMetricsInterface
	if cfg.Metrics.Enabled {
		logger.Info("Metrics enabled, setting up Prometheus metrics")

		// Start metrics HTTP server with worker metrics
		services := []string{internalMetrics.ServiceWorker}
		_ = internalMetrics.StartMetricsServer(internalMetrics.Config{
			Enabled: true,
			Host:    cfg.Metrics.Host,
			Port:    cfg.Metrics.Port,
		}, services, logger)

		// Create worker metrics instance
		workerMetrics = internalMetrics.NewWorkerMetrics()
	} else {
		logger.Info("Worker metrics disabled")
		workerMetrics = internalMetrics.NewNoOpWorkerMetrics()
	}

	mux := asynq.NewServeMux()

	// Wrap handlers with metrics collection
	mux.HandleFunc(tasks.TypeKeyGenerationDKLS,
		workerMetrics.Handler("keygen", vaultMgmService.HandleKeyGenerationDKLS))
	mux.HandleFunc(tasks.TypeKeySignDKLS,
		workerMetrics.Handler("keysign", vaultMgmService.HandleKeySignDKLS))
	mux.HandleFunc(tasks.TypeReshareDKLS,
		workerMetrics.Handler("reshare", feeMgmService.HandleReshareDKLS))
	mux.HandleFunc(tasks.TypeRecurringFeeRecord,
		workerMetrics.Handler("fees", policyService.HandleScheduledFees))
	mux.HandleFunc(tasks.TypePolicyDeactivate,
		workerMetrics.Handler("policy_deactivate", policyService.HandlePolicyDeactivate))

	if err := srv.Run(mux); err != nil {
		panic(fmt.Errorf("could not run server: %w", err))
	}
}
