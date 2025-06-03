package common

import "embed"

//go:embed migrations/system/*.sql
var systemMigrations embed.FS

func SystemMigrations() embed.FS {
	return systemMigrations
}
