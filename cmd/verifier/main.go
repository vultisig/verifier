package main

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/api"
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

	logger := logrus.New()

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

	server := api.NewServer(
		*cfg,
		db,
		redisStorage,
		vaultStorage,
		client,
		inspector,
		cfg.Server.JWTSecret,
		txIndexerService,
	)
	if err := server.StartServer(); err != nil {
		panic(err)
	}
}
