package main

import (
	"fmt"
	"strings"

	"github.com/hibiken/asynq"

	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/portal"
	"github.com/vultisig/verifier/internal/storage"
	"github.com/vultisig/verifier/internal/storage/postgres"
)

func main() {
	cfg, err := config.ReadPortalConfig()
	if err != nil {
		panic(err)
	}

	db, err := postgres.NewPostgresBackend(cfg.Database.DSN, nil)
	if err != nil {
		panic(err)
	}

	pool := db.Pool()
	defer pool.Close()

	missing := cfg.PluginAssets.Validate()
	if len(missing) > 0 {
		panic(fmt.Sprintf("plugin_assets configuration missing: %s", strings.Join(missing, ", ")))
	}

	assetStorage, err := storage.NewS3PluginAssetStorage(cfg.PluginAssets)
	if err != nil {
		panic(err)
	}

	var queueClient *asynq.Client
	redisCfg := cfg.Redis
	if redisCfg.URI != "" {
		redisConnOpt, parseErr := asynq.ParseRedisURI(redisCfg.URI)
		if parseErr != nil {
			panic(parseErr)
		}
		queueClient = asynq.NewClient(redisConnOpt)
	} else if redisCfg.Host != "" {
		redisConnOpt := asynq.RedisClientOpt{
			Addr:     redisCfg.Host + ":" + redisCfg.Port,
			Username: redisCfg.User,
			Password: redisCfg.Password,
			DB:       redisCfg.DB,
		}
		queueClient = asynq.NewClient(redisConnOpt)
	}
	if queueClient != nil {
		defer queueClient.Close()
	}

	server := portal.NewServer(*cfg, pool, db, assetStorage, queueClient)
	if err := server.Start(); err != nil {
		panic(err)
	}
}
