# adjust to point to your local go-wrappers repo
DYLD_LIBRARY=../go-wrappers/includes/darwin/:$LD_LIBRARY_PATH

up:
	@docker compose up -d --remove-orphans;

down:
	@docker compose down

seed-db:
	VS_VERIFIER_CONFIG_NAME=verifier.example go run testdata/scripts/seed_db.go

run-frontend:
	@cd plugins-ui && npm install && VITE_MARKETPLACE_URL=http://localhost:8080 npm run dev
# Run the verifier server
run-server:
	@DYLD_LIBRARY_PATH=$(DYLD_LIBRARY) VS_CONFIG_NAME=config go run cmd/verifier/main.go


# Run the worker process
run-worker:
	@DYLD_LIBRARY_PATH=$(DYLD_LIBRARY) VS_CONFIG_NAME=config go run cmd/worker/main.go

# Dump database schema
# Usage: make dump-schema CONFIG=config.json
dump-schema:
	@if [ -z "$(CONFIG)" ]; then \
		echo "Error: CONFIG parameter is required. Usage: make dump-schema CONFIG=config.json"; \
		exit 1; \
	fi
	@DSN=$$(jq -r '.database.dsn' $(CONFIG)); \
	pg_dump "$$DSN" --schema-only \
		--no-comments \
		--no-owner \
		--quote-all-identifiers \
		-T public.goose_db_version \
		-T public.goose_db_version_id_seq | sed \
		-e '/^--.*/d' \
		-e '/^SET /d' \
		-e '/^SELECT pg_catalog./d' \
		-e 's/"public"\.//' | awk '/./ { e=0 } /^$$/ { e += 1 } e <= 1' \
		> ./internal/storage/postgres/schema/schema.sql
	
