package main

import (
	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/portal"
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

	server := portal.NewServer(*cfg, pool)
	if err := server.Start(); err != nil {
		panic(err)
	}
}
