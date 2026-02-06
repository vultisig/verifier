package main

import (
	"fmt"
	"strings"

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

	server := portal.NewServer(*cfg, pool, db, assetStorage)
	if err := server.Start(); err != nil {
		panic(err)
	}
}
