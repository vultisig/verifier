# Verifier Stack - Run Cheatsheet

## Library Path (required)
```bash
export DYLD_LIBRARY_PATH=/Users/dev/dev/vultisig/go-wrappers/includes/darwin/:$DYLD_LIBRARY_PATH
```

## 1. Start Infrastructure
```bash
docker network create shared_network  # once
cd /Users/dev/dev/vultisig/verifier
docker compose up -d postgres redis minio
```

## 2. Seed Database
```bash
make seed-db
```

## 3. Update Plugin Endpoints & Make Free
```bash
docker exec verifier-postgres-1 psql -U myuser -d vultisig-verifier -c "
UPDATE plugins SET server_endpoint = 'http://localhost:8082' WHERE id = 'vultisig-dca-0000';
UPDATE plugins SET server_endpoint = 'http://localhost:8083' WHERE id = 'vultisig-recurring-sends-0000';
DELETE FROM pricings WHERE plugin_id IN ('vultisig-dca-0000', 'vultisig-recurring-sends-0000');
"
```

## 4. Copy Vault to App-Recurring MinIO
```bash
# Create bucket in app-recurring MinIO
docker exec app-recurring-minio mc alias set local http://localhost:9000 minioadmin minioadmin
docker exec app-recurring-minio mc mb local/vultisig-dca

# Copy vault file from verifier to app-recurring
docker exec verifier-minio-1 mc alias set local http://localhost:9000 minioadmin minioadmin
docker exec verifier-minio-1 mc cp local/vultisig-verifier/vultisig-dca-0000-*.vult /tmp/
docker cp verifier-minio-1:/tmp/vultisig-dca-0000-*.vult /tmp/
docker cp /tmp/vultisig-dca-0000-*.vult app-recurring-minio:/tmp/
docker exec app-recurring-minio mc cp /tmp/vultisig-dca-0000-*.vult local/vultisig-dca/
```

## 5. Run Services

### Verifier Server (port 8080)
```bash
AUTH_ENABLED=true VS_VERIFIER_CONFIG_NAME=verifier.example go run ./cmd/verifier
```

### Worker
```bash
VS_WORKER_CONFIG_NAME=worker.example go run ./cmd/worker
```

### TX Indexer
```bash
VS_TX_INDEXER_CONFIG_NAME=tx_indexer.example go run ./cmd/tx_indexer
```

## Quick Reference

| Service | Port | Check |
|---------|------|-------|
| Server | 8080 | `curl localhost:8080/healthz` |
| PostgreSQL | 5432 | `myuser:mypassword` / `vultisig-verifier` |
| Redis | 6379 | |
| MinIO | 9000 | `minioadmin:minioadmin` |

## Stop
```bash
docker compose down
pkill -f "go run ./cmd"
```
