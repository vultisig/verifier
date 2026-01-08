# adjust to point to your local go-wrappers repo
# https://github.com/vultisig/go-wrappers.git
DYLD_LIBRARY=../go-wrappers/includes/darwin/:$LD_LIBRARY_PATH
VS_VERIFIER_CONFIG_NAME ?= verifier.example
VERIFIER_URL ?= http://localhost:8080

S3_ENDPOINT ?= http://localhost:9000
S3_ACCESS_KEY ?= minioadmin
S3_SECRET_KEY ?= minioadmin
S3_BUCKET ?= vultisig-verifier

.PHONY: up up-dev down down-dev build build-dev seed-db run-server run-worker dump-schema test-integration itest

up:
	@docker compose up -d --remove-orphans;

up-dev:
	@docker compose -f docker-compose.yaml -f docker-compose.dev.yaml up -d --remove-orphans;

down:
	@docker compose down

down-dev:
	@docker compose -f docker-compose.yaml -f docker-compose.dev.yaml down

build:
	@docker compose build

build-dev:
	@docker compose -f docker-compose.yaml -f docker-compose.dev.yaml build

seed-db:
	VS_VERIFIER_CONFIG_NAME=verifier.example go run testdata/scripts/seed_db.go

test-integration:
	@echo "Running integration tests..."
	@VERIFIER_URL=$(VERIFIER_URL) \
	VS_VERIFIER_CONFIG_NAME=$(VS_VERIFIER_CONFIG_NAME) \
	S3_ENDPOINT=$(S3_ENDPOINT) \
	S3_ACCESS_KEY=$(S3_ACCESS_KEY) \
	S3_SECRET_KEY=$(S3_SECRET_KEY) \
	S3_BUCKET=$(S3_BUCKET) \
	go test -v -count=1 -timeout 10m ./testdata/integration/gotest/...
itest: test-integration

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
		-e '/^\\restrict /d' \
		-e '/^\\unrestrict /d' \
		-e 's/"public"\.//' | awk '/./ { e=0 } /^$$/ { e += 1 } e <= 1' \
		> ./internal/storage/postgres/schema/schema.sql
