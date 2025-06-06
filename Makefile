# adjust to point to your local go-wrappers repo
DYLD_LIBRARY=../go-wrappers/includes/darwin/:$LD_LIBRARY_PATH

up:
	@docker compose up -d --remove-orphans;

down:
	@docker compose down

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
		--quote-all-identifiers \
		-T public.goose_db_version \
		-T public.goose_db_version_id_seq | sed \
		-e '/^--.*/d' \
		-e '/^SET /d' \
		-e '/^[[:space:]]*$$/d' \
		-e '/^SELECT pg_catalog./d' \
		-e '/^ALTER TABLE .* OWNER TO "[^"]*";/d' \
		-e '/^ALTER TYPE .* OWNER TO "[^"]*";/d' \
		-e '/^ALTER FUNCTION .* OWNER TO "[^"]*";/d' \
		-e 's/"public"\.//' \
		> ./internal/storage/postgres/schema/schema.sql
	
