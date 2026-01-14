# Vultisig CLI Testing - Scratchpad

## Vault: FastPlugin1
- **Password:** `Password123`
- **Public Key ECDSA:** `022bb79475f896618cb26905a7b609d1a3f6b0a845878ea3dd301607c47a37a06a`
- **Public Key EdDSA:** `fd94522c03ff2bb571c3f120f2ac4f3732c86995fd827a309a2a8648b59e5d93`
- **ETH Address:** `0x2d63088Dacce3a87b0966982D52141AEe53be224`
- **Local Party:** `dev's MacBook Air-AC8`
- **Server Party:** `Server-08017`
- **Source File:** `vultisig-cluster/local/FastPlugin1-a06a-share2of2.vult`
- **Imported To:** `~/.vultisig/vaults/022bb79475f89661.json`

## Working Config
- **Encryption Secret:** `dev-encryption-secret-32b`
- **Relay Server:** `https://api.vultisig.com/router`
- **Fast Vault Server:** `https://api.vultisig.com`
- **JWT Secret:** `devsecret`

## Local Services (devctl start)
| Service | Port | Status |
|---------|------|--------|
| Verifier API | 8080 | Working |
| Verifier Worker | 8089 | Working |
| DCA Server | 8082 | Working |
| DCA Worker | 8183 | Working |
| PostgreSQL | 5432 | Working |
| Redis | 6379 | Working |
| MinIO | 9000 | Working |

## Verified Working
- `curl /vault/exist/{pubkey}` → 200 (vault exists on Fast Vault Server)
- `curl /vault/sign` with Password123 → Returns task ID (keysign request accepted)
- `devctl start` → All 6 services start successfully
- `devctl vault info` → Shows vault details
- `devctl report` → Shows full status
- Fast Vault reshare request accepted (plugin install gets past Fast Vault step)

## Config Files (Fixed)
- `devenv/config/verifier.json` - encryption_secret updated
- `devenv/config/worker.json` - encryption_secret + relay server updated
- `devenv/config/dca-worker.env` - Added RPC_ZKSYNC_URL, RPC_CRONOS_URL, RPC_TRON_URL
- `devenv/dca-policy.json` - Added vault ETH address

## Commands
```bash
# Set library path
export DYLD_LIBRARY_PATH=/Users/dev/dev/vultisig/go-wrappers/includes/darwin:$DYLD_LIBRARY_PATH

# Start services
go run ./cmd/devctl start

# Check status
go run ./cmd/devctl report

# Plugin install
go run ./cmd/devctl plugin install vultisig-dca-0000 -p "Password123"
```

## Current Blocker
- Auth login fails with 404 from nginx (keysign endpoint)
- Plugin install passes Fast Vault but fails at verifier 401 (needs fresh auth token)
