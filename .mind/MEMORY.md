
## 2026-01-14 05:33:18
VERIFIER REPOSITORY OVERVIEW (jp/cli-testing-tool branch):

MAIN ENTRY POINTS:
1. cmd/verifier/main.go - Verifier API server (Echo framework, Postgres, Redis)
2. cmd/worker/main.go - Async task worker (Asynq queue processor)
3. cmd/devctl/main.go - CLI testing tool (Cobra-based CLI)
4. cmd/portal/main.go - Developer portal
5. cmd/tx_indexer/main.go - Transaction indexer

KEY VULTISIG DEPENDENCIES:
- github.com/vultisig/commondata - Protocol buffers for vault structures
- github.com/vultisig/go-wrappers - CGO bindings to TSS/crypto libraries
- github.com/vultisig/mobile-tss-lib - TSS library
- github.com/vultisig/recipes - Plugin recipe definitions
- github.com/vultisig/vultiserver - Relay/authentication services
- github.com/vultisig/vultisig-go - Core utilities (address, relay, types)

CORE FRAMEWORKS:
- Echo/v4 - HTTP server framework
- Cobra - CLI framework
- Asynq - Task queue
- PostgreSQL - Primary database
- Redis - Caching and queue backend
- MinIO - S3-compatible storage

DEVCTL CLI TESTING TOOL:
Location: cmd/devctl/
Purpose: CLI for local plugin development without browser extension
Features:
- vault import/export/generate/reshare
- plugin install/list/info/spec
- policy create/list/update
- tss operations (keygen/keysign/reshare)
- auth/verify/status/report commands
- Integration with Fast Vault Server for 2-of-2 initial keygen
- 4-party reshare (CLI + Fast Vault + Verifier + Plugin)

JP/CLI-TESTING-TOOL BRANCH ADDITIONS:
- Complete devctl CLI tool (~3000+ lines of command handlers)
- devenv/ directory with Docker Compose and configurations
- Integration tests in testdata/integration/gotest/
- Enhanced plugin API endpoints
- Local development documentation and examples
