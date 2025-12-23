# Integration Tests

Plugin integration tests using Hurl with parallel execution.

## Quick Start

```bash
make test-integration
```

## Structure

```
testdata/integration/
├── cmd/itutil/main.go     # Unified CLI helper tool (itutil)
├── bin/itutil             # Compiled binary (auto-built)
├── scripts/
│   ├── run-integration-tests.sh  # Main test runner
│   └── seed-integration-db.sql   # Database seed SQL
├── hurl/
│   ├── plugin-integration.hurl   # Test template
│   └── generated/                # Generated test files per plugin
├── fixture.json           # Test vault fixture
├── test-results/          # Test output (HTML, JUnit, logs)
└── README.md
```

## itutil CLI

A unified CLI tool for integration test helpers. Built automatically on first run.

### Commands

```bash
# Generate JWT token
itutil jwt --secret mysecret --pubkey 0x1234... [--token-id ID] [--expire-hours 24]

# Generate EVM transaction fixtures
itutil evm-fixture [--chain-id 1] [--to ADDR] [--value WEI] [--output shell|json|plain]

# Generate permissive policy (base64)
itutil policy-b64 --allow-all [--target ADDR]

# Seed database with plugins
itutil seed-db [--proposed path/to/proposed.yaml] [--dsn DSN]
```

### Manual Build

```bash
go build -o testdata/integration/bin/itutil ./testdata/integration/cmd/itutil
```

## Test Runner Features

- **Parallel execution**: Tests run in parallel using Hurl's `--jobs` flag
- **HTML reports**: Detailed HTML report at `test-results/html/index.html`
- **JUnit XML**: CI-compatible output at `test-results/junit.xml`
- **Slowest tests**: Summary shows timing for performance debugging

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `VERIFIER_URL` | `http://localhost:8080` | Verifier API endpoint |
| `HURL_JOBS` | `4` | Parallel test jobs |

## Adding New Plugins

1. Add plugin to `proposed.yaml` in repo root
2. Run `make seed-integration-db` to seed the database
3. Tests are automatically generated for new plugins

## Test Template

The template at `hurl/plugin-integration.hurl` defines 14 tests per plugin:
1. GET /plugins/{id} (200)
2. GET /plugins/{id}/recipe-specification (200)
3. POST /vault/reshare (200)
4. POST /vault/reshare duplicate (200)
5. POST /plugin/policy with valid JWT (400 - signature validation)
6. POST /plugin/policy without auth (401)
7. GET /plugin/policy/{id} without auth (401)
8. GET /plugin/policy/{id} with bad ID (400)
9. POST /plugin-signer/sign without auth (401)
10. POST /plugin-signer/sign with bad API key (401)
11. POST /plugin-signer/sign with empty messages (400)
12. POST /plugin-signer/sign with valid request (200)
13. GET /plugin-signer/status/{task_id} (200)
14. GET /plugin-signer/status/invalid (404)
