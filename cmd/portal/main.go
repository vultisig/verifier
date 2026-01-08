package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/portal"
)

func main() {
	ctx := context.Background()

	cfg, err := config.ReadPortalConfig()
	if err != nil {
		panic(err)
	}

	pool, err := pgxpool.New(ctx, cfg.Database.DSN)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	server := portal.NewServer(*cfg, pool)
	if err := server.Start(); err != nil {
		panic(err)
	}
}
