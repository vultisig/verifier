# Verifier Stack - Local Deployment Cheatsheet

## Prerequisites
```bash
# Ensure Docker is running
open -a Docker

# Create shared network (once)
docker network create shared_network

# Set library path (required for TSS operations - macOS only)
# Linux/Windows users: skip this step, the library is statically linked
export DYLD_LIBRARY_PATH=/Users/dev/dev/vultisig/go-wrappers/includes/darwin/:$DYLD_LIBRARY_PATH
```

## 1. Start Infrastructure
```bash
cd /Users/dev/dev/vultisig/verifier
docker compose up -d postgres redis minio
```

Wait for services to be ready:
```bash
docker exec verifier-postgres-1 pg_isready -U myuser -d vultisig-verifier
```

Create MinIO bucket:
```bash
docker exec verifier-minio-1 mc alias set myminio http://localhost:9000 minioadmin minioadmin
docker exec verifier-minio-1 mc mb --ignore-existing myminio/vultisig-verifier
```

## 2. Config Files

### Server Config (`config.json`)
Copy from `verifier.example.json`:
```bash
cp verifier.example.json config.json
```

### Worker Config (`worker-config.json`)
**IMPORTANT**: Worker uses `VS_WORKER_CONFIG_NAME` env var, not `VS_CONFIG_NAME`

Create `worker-config.json` with relay server configured:
```json
{
  "log_format": "text",
  "vault_service": {
    "relay": {
      "server": "https://api.vultisig.com/router"
    },
    "local_party_prefix": "verifier",
    "encryption_secret": "test123",
    "do_setup_msg": false
  },
  "redis": {
    "host": "localhost",
    "port": "6379"
  },
  "block_storage": {
    "host": "http://localhost:9000",
    "region": "us-east-1",
    "access_key": "minioadmin",
    "secret": "minioadmin",
    "bucket": "vultisig-verifier"
  },
  "database": {
    "dsn": "postgres://myuser:mypassword@localhost:5432/vultisig-verifier?sslmode=disable"
  },
  "plugin": {},
  "fees": {
    "usdc_address": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
  },
  "metrics": {
    "enabled": true,
    "host": "0.0.0.0",
    "port": 8089
  }
}
```

## 3. Run Services

### Verifier Server (port 8080)
```bash
VS_CONFIG_NAME=config go run cmd/verifier/main.go
```

### Verifier Worker
```bash
VS_WORKER_CONFIG_NAME=worker-config go run cmd/worker/main.go
```

**Task Queue Isolation**: When running multiple workers (e.g., verifier + DCA plugin), use `TASK_QUEUE_NAME` to prevent workers from stealing each other's tasks:
```bash
# Verifier worker (default queue)
VS_WORKER_CONFIG_NAME=worker-config go run cmd/worker/main.go

# DCA plugin worker (separate queue)
TASK_QUEUE_NAME=dca_plugin_queue VS_WORKER_CONFIG_NAME=worker-config go run cmd/worker/main.go
```
For single-worker setups, the default queue (`default_queue`) is sufficient.

## 4. Update Plugin Endpoints in Database

After starting, update plugin endpoints to point to local services:
```bash
docker exec verifier-postgres-1 psql -U myuser -d vultisig-verifier -c \
  "UPDATE plugins SET server_endpoint = 'http://localhost:8085' WHERE id = 'vultisig-fees-feee';"

docker exec verifier-postgres-1 psql -U myuser -d vultisig-verifier -c \
  "UPDATE plugins SET server_endpoint = 'http://localhost:8082' WHERE id = 'vultisig-dca-0000';"

docker exec verifier-postgres-1 psql -U myuser -d vultisig-verifier -c \
  "UPDATE plugins SET server_endpoint = 'http://localhost:8083' WHERE id = 'vultisig-recurring-sends-0000';"
```

## Quick Reference

| Service | Port | Config Env Var |
|---------|------|----------------|
| Verifier Server | 8080 | `VS_CONFIG_NAME` |
| Verifier Worker | - | `VS_WORKER_CONFIG_NAME` |
| PostgreSQL | 5432 | - |
| Redis | 6379 | - |
| MinIO | 9000 (API), 9090 (Console) | - |

## 5. Configure Extension for Local Development

The browser extension defaults to production verifier. You must change it to use localhost.

1. In the extension, find the version text at bottom (e.g., "VULTISIG EXTENSION V1.x.x")
2. **Click it 3 times** to open Developer Options modal
3. Change **Plugin Server URL** from `https://verifier.vultisig.com` to `http://localhost:8080`
4. Click Save

## Troubleshooting

### "unsupported protocol scheme" error
Worker config missing `vault_service.relay.server`. Ensure `worker-config.json` has the relay URL.

### Plugin installation stuck at "connecting to app store"
Extension is talking to production verifier instead of local. Configure Developer Options:
1. Click version text 3 times to open Developer Options
2. Set Plugin Server URL to `http://localhost:8080`

### Plugin installation stuck at "connecting to server"
The TSS session is registered but waiting for the other party. Check:
1. Extension/app is using same relay server (`https://api.vultisig.com/router`)
2. Session ID matches between worker logs and extension
3. Extension Developer Options pointing to `http://localhost:8080`

### Metrics port conflict
Non-critical - worker/server share default metrics port. Update `metrics.port` in configs to use different ports.

## Stop Services
```bash
docker compose down
pkill -f "go run cmd"
```
