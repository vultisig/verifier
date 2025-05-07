package main

import (
	"fmt"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/api"
	"github.com/vultisig/verifier/internal/storage"
	"github.com/vultisig/verifier/internal/storage/postgres"
	"github.com/vultisig/verifier/vault"
)

func main() {
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

	db, err := postgres.NewPostgresBackend(cfg.Server.Database.DSN)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}

	server := api.NewServer(
		*cfg,
		db,
		redisStorage,
		vaultStorage,
		client,
		inspector,
		sdClient,
		cfg.Server.JWTSecret,
	)
	if err := server.StartServer(); err != nil {
		panic(err)
	}
}
