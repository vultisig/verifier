# Integration Tests

Go-based integration tests for local development with vcli.

## Quick Start

Start vcli, then run:

```bash
make test-integration
```

Or manually:

```bash
DATABASE_DSN="postgres://vultisig:vultisig@localhost:5432/vultisig-verifier?sslmode=disable" \
ENCRYPTION_SECRET="dev-encryption-secret-32b" \
JWT_SECRET="devsecret" \
go test -v -count=1 ./testdata/integration/gotest/...
```

## Structure

```
testdata/integration/
├── gotest/                   # Go integration tests
│   ├── integration_test.go   # TestMain setup
│   ├── client.go             # HTTP client helpers
│   ├── fixtures.go           # Fixture loaders and test plugins
│   ├── jwt.go                # JWT generation
│   ├── evm.go                # EVM fixture generation
│   ├── seeder.go             # DB/S3 seeding
│   ├── plugin_test.go        # Plugin endpoint tests
│   ├── vault_test.go         # Vault endpoint tests
│   ├── policy_test.go        # Policy endpoint tests
│   └── signer_test.go        # Signer endpoint tests
├── fixture.json              # Test vault fixture
└── README.md
```

## Environment Variables

| Variable | vcli Default | Description |
|----------|--------------|-------------|
| `VERIFIER_URL` | `http://localhost:8080` | Verifier API endpoint |
| `DATABASE_DSN` | `postgres://vultisig:vultisig@localhost:5432/vultisig-verifier?sslmode=disable` | PostgreSQL connection |
| `ENCRYPTION_SECRET` | `dev-encryption-secret-32b` | Vault encryption secret |
| `JWT_SECRET` | `devsecret` | JWT signing secret |
| `S3_ENDPOINT` | `http://localhost:9000` | MinIO/S3 endpoint |
| `S3_ACCESS_KEY` | `minioadmin` | S3 access key |
| `S3_SECRET_KEY` | `minioadmin` | S3 secret key |
| `S3_BUCKET` | `vultisig-verifier` | S3 bucket name |
| `DCA_PLUGIN_URL` | `http://localhost:8082` | DCA plugin server endpoint |

## Test Flow

1. **TestMain** seeds database and S3 with test fixtures
2. **Reshare** is initiated once for all plugins
3. **WaitForVault** polls until vaults are created
4. **Tests** run against seeded data

## Tests

16 tests per plugin covering:

**Plugin endpoints:**
1. GET /plugins/{id} (200)
2. GET /plugins/{id}/recipe-specification (200)

**Vault endpoints:**
3. GET /vault/exist/{plugin}/{pubkey} (200)
4. GET /vault/get/{plugin}/{pubkey} (200)

**Policy endpoints:**
5. GET /plugin/policy/{id} (200) - happy path
6. GET /plugin/policies/{pluginId} (200) - happy path
7. POST /plugin/policy with valid JWT (400 - signature validation)
8. POST /plugin/policy without auth (401)
9. GET /plugin/policy/{id} without auth (401)
10. GET /plugin/policy/{id} with bad ID (400)

**Plugin-signer endpoints:**
11. POST /plugin-signer/sign without auth (401)
12. POST /plugin-signer/sign with bad API key (401)
13. POST /plugin-signer/sign with empty messages (400)
14. POST /plugin-signer/sign with valid request (200)
15. GET /plugin-signer/sign/response/{task_id} without auth (401)
16. GET /plugin-signer/sign/response/{task_id} with API key (any)
