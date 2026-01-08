# adjust to point to your local go-wrappers repo
# https://github.com/vultisig/go-wrappers.git
DYLD_LIBRARY=../go-wrappers/includes/darwin/:$LD_LIBRARY_PATH
VS_VERIFIER_CONFIG_NAME ?= verifier.example
VERIFIER_URL ?= http://localhost:8080

HURL_JOBS ?= 4
S3_ENDPOINT ?= http://localhost:9000
S3_ACCESS_KEY ?= minioadmin
S3_SECRET_KEY ?= minioadmin
S3_BUCKET ?= vultisig-verifier
ITUTIL := testdata/integration/bin/itutil

.PHONY: up up-dev down down-dev build build-dev seed-db run-server run-worker dump-schema build-itutil integration-seed test-integration itest

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


build-itutil:
	@go build -o $(ITUTIL) ./testdata/integration/cmd/itutil

integration-seed: build-itutil
	@echo "Seeding DB + vault fixtures..."
	@VS_VERIFIER_CONFIG_NAME=$(VS_VERIFIER_CONFIG_NAME) \
	$(ITUTIL) seed-db --proposed proposed.yaml --fixture testdata/integration/fixture.json
	@$(ITUTIL) seed-vault \
		--fixture testdata/integration/fixture.json \
		--proposed proposed.yaml \
		--s3-endpoint $(S3_ENDPOINT) \
		--s3-access-key $(S3_ACCESS_KEY) \
		--s3-secret-key $(S3_SECRET_KEY) \
		--s3-bucket $(S3_BUCKET)

test-integration: integration-seed
	@VERIFIER_URL=$(VERIFIER_URL) HURL_JOBS=$(HURL_JOBS) \
	bash testdata/integration/scripts/run-integration-tests.sh
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
	
