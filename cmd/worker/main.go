package main

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/service"
	"github.com/vultisig/verifier/internal/storage/postgres"
	"github.com/vultisig/verifier/plugin/tasks"
	"github.com/vultisig/verifier/tx_indexer"
	"github.com/vultisig/verifier/tx_indexer/pkg/storage"
	"github.com/vultisig/verifier/vault"
)

func main() {
	ctx := context.Background()

	cfg, err := config.GetConfigure()
	if err != nil {
		panic(err)
	}

	sdClient, err := statsd.New(cfg.Datadog.Host + ":" + cfg.Datadog.Port)
	if err != nil {
		panic(err)
	}
	redisCfg := cfg.Redis
	redisOptions := asynq.RedisClientOpt{
		Addr:     redisCfg.Host + ":" + redisCfg.Port,
		Username: redisCfg.User,
		Password: redisCfg.Password,
		DB:       redisCfg.DB,
	}
	logger := logrus.StandardLogger()
	client := asynq.NewClient(redisOptions)
	vaultStorage, err := vault.NewBlockStorageImp(cfg.BlockStorage)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize vault storage: %v", err))
	}

	backendDB, err := postgres.NewPostgresBackend(cfg.Database.DSN, nil)
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
		redisOptions,
		asynq.Config{
			Logger:      logger,
			Concurrency: 10,
			Queues: map[string]int{
				tasks.QUEUE_NAME:         10,
				vault.EmailQueueName:     100,
				"scheduled_plugin_queue": 10, // new queue
			},
		},
	)

	txIndexerStore, err := storage.NewPostgresTxIndexStore(ctx, cfg.Database.DSN)
	if err != nil {
		panic(fmt.Sprintf("storage.NewPostgresTxIndexStore: %v", err))
	}

	txIndexerService := tx_indexer.NewService(
		logger,
		txIndexerStore,
		tx_indexer.Chains(),
	)

	vaultMgmService, err := vault.NewManagementService(
		cfg.VaultService,
		client,
		sdClient,
		vaultStorage,
		txIndexerService,
	)

	mux := asynq.NewServeMux()
	// mux.HandleFunc(tasks.TypePluginTransaction, workerService.HandlePluginTransaction)
	mux.HandleFunc(tasks.TypeKeyGenerationDKLS, vaultMgmService.HandleKeyGenerationDKLS)
	mux.HandleFunc(tasks.TypeKeySignDKLS, vaultMgmService.HandleKeySignDKLS)
	mux.HandleFunc(tasks.TypeReshareDKLS, vaultMgmService.HandleReshareDKLS)
	mux.HandleFunc(tasks.TypeRecurringFeeRecord, policyService.HandleScheduledFees)

	if err := srv.Run(mux); err != nil {
		panic(fmt.Errorf("could not run server: %w", err))
	}
}
