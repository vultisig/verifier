package main

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/tx_indexer"

	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/api"
	"github.com/vultisig/verifier/internal/storage"
	"github.com/vultisig/verifier/internal/storage/postgres"
	tx_indexer_storage "github.com/vultisig/verifier/tx_indexer/pkg/storage"
	"github.com/vultisig/verifier/vault"
)

// @title Vultisig Verifier API
// @version 1.0
// @description The Verifier (trusted counterparty) API for Vultisig plugins.
// @termsOfService http://todo.com

// @contact.name API Support
// @contact.url http://todo.com
// @contact.email todo@todo.com

// @license.name MIT Licence
// @license.url https://raw.githubusercontent.com/vultisig/verifier/refs/heads/main/LICENSE

// @host todo.com
func main() {
	ctx := context.Background()

	cfg, err := config.ReadVerifierConfig()
	if err != nil {
		panic(err)
	}

	logger := logrus.New()

	sdClient, err := statsd.New(fmt.Sprintf("%s:%s", cfg.Datadog.Host, cfg.Datadog.Port))
	if err != nil {
		panic(err)
	}

	redisStorage, err := storage.NewRedisStorage(cfg.Redis)
	if err != nil {
		panic(err)
	}

	redisOptions := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Host + ":" + cfg.Redis.Port,
		Username: cfg.Redis.User,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	client := asynq.NewClient(redisOptions)
	defer func() {
		if err := client.Close(); err != nil {
			fmt.Println("fail to close asynq client,", err)
		}
	}()

	inspector := asynq.NewInspector(redisOptions)

	vaultStorage, err := vault.NewBlockStorageImp(cfg.BlockStorageConfig)
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

	txIndexerService := tx_indexer.NewService(
		logger,
		txIndexerStore,
		tx_indexer.Chains(),
	)

	server := api.NewServer(
		*cfg,
		db,
		redisStorage,
		vaultStorage,
		client,
		inspector,
		sdClient,
		cfg.Server.JWTSecret,
		txIndexerService,
	)
	if err := server.StartServer(); err != nil {
		panic(err)
	}
}
