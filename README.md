# Vultisig Verifier

A service providing threshold signature scheme (TSS) operations for vaults. Works with [Vultisig Plugins](https://github.com/vultisig/plugin) to process and sign blockchain transactions.

## Features

- Vault management (create, retrieve, manage)
- TSS operations with DKLS
- Transaction processing
- Policy enforcement
- Asynchronous processing

## Architecture

- **Core Components**: API server, Worker service, PostgreSQL/Redis storage
- **Integration**: Receives transactions from Plugins, processes them securely
- **Security Model**: Plugins handle business logic; Verifier manages cryptographic operations

## Setup

### Prerequisites
- Go 1.24.2+, PostgreSQL 14+, Redis 6+, Docker

### Quick Start
```bash
# create a shared network for docker containers
docker network create shared_network
# start verifier and worker in docker compose
make up

# seed the postgres database with initial data
make seed-db

# run frontend marketplace locally
make run-frontend
```

How to install dev version of Chrome extension from desired branch:

- Clone [vultisig-windows](https://github.com/vultisig/vultisig-windows) repo and checkout to desired branch;
- `yarn install` to install deps;
- `yarn run build-extension` to build Chrome extension files;
- Open Chrome and load extension there:
  - Open `chrome://extensions`;
  - Click `Developer mode` toggle at top-right;
  - Click `Load unpacked`
  - Select `vultisig-windows/clients/extension/dist` folder;
  - Now you'll see Vultisig Extension at the extensions list;

To switch branch or update it: rebuild code and click `Reload` button near Vultisig Extension at the extensions list — Chrome would reload files from the same path;

![extension.png](readme-static/extension.png)

### Configuration
Edit `config.yaml` with appropriate settings for:
- Server (port, host)
- Database connection
- Redis connection
- Storage options (S3 or local)
- TSS parameters

## API Endpoints

**Authentication:** `/auth` (POST), `/auth/refresh` (POST)

**Vault Management:**
- Create: `/vault/create` (POST)
- Reshare: `/vault/reshare` (POST)
- Get: `/vault/get/:pubKey` (GET)
- Check: `/vault/exist/:pubKey` (GET)

**Signing:**
- Sign: `/vault/sign` (POST)
- Get results: `/vault/sign/response/:id` (GET)

**Transactions:**
- Create: `/sync/transaction` (POST)
- Update: `/sync/transaction` (PUT)

## Development

**Key directories:**
- `/cmd` - Entry points (verifier, worker)
- `/internal` - Core implementation (API, services, storage)
- `/types` - Data structures
- `/vault` - Vault storage implementations

**Testing:** Run `make test` or `go test ./[path]/...`

## License

See LICENSE file for terms.
