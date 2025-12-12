# Plugin Smoke Test Tool

Simple Go-based tool to test plugin HTTP endpoints without requiring access to plugin source code.

## Scope

This is a **lightweight API contract smoke test** that validates:

* Plugin endpoints exist and are reachable
* Endpoints return expected HTTP status codes (`200`, `400`, `401`, `404`)
* Responses are valid JSON when a JSON body is returned
* Required fields are present in key responses (e.g. `plugin_id`, `plugin_name`, etc.)

This test **does NOT** validate:

* Actual plugin installation or real keyshare generation (requires verifier + DB + seeded data)
* Business logic correctness
* End-to-end vault / policy flows

For full E2E integration tests (including real vault reshare and keyshare validation), use the complete verifier test suite.

## Usage

### Test all plugins from `proposed.yaml`

```bash
./plugin-smoke-test proposed.yaml
```

### Test a single plugin

```bash
./plugin-smoke-test <plugin-url> [plugin-id] [plugin-title]
```

Example:

```bash
./plugin-smoke-test https://plugin-dca-swap.lb.services.1conf.com vultisig-dca-0000 "Recurring Swaps"
```

## What It Tests

### Required Endpoints (8 tests total)

1. `GET /plugin/recipe-specification`
   Validates plugin metadata and schema:

   * `plugin_id`
   * `plugin_name`
   * `requirements`
   * `configuration`
   * `configuration_example`
   * `supported_resources`
   * `permissions`

2. `GET /vault/exist/:pluginId/:publicKey` – vault existence check

3. `GET /vault/get/:pluginId/:publicKey` – retrieve vault metadata

4. `DELETE /vault/:pluginId/:publicKey` – delete vault

5. `POST /vault/reshare` – create/reshare vault

6. `POST /plugin/policy` – create policy (usually requires auth)

7. `PUT /plugin/policy` – update policy (usually requires auth)

8. `DELETE /plugin/policy/:policyId` – delete policy (usually requires auth)

For these endpoints, the smoke test expects:

* **Acceptable status codes:** `200`, `400`, `401`, `404`

  * `200 OK` – request handled successfully
  * `400 Bad Request` – validation error (fine for smoke test)
  * `401 Unauthorized` – missing auth / signature (fine for smoke test)
  * `404 Not Found` – resource not found but endpoint exists

* **Unacceptable status codes (test fails):** `5xx` or anything outside `{200, 400, 401, 404}`

  * e.g. `500`, `502`, `503`, `504` → treated as server / deployment issues

For `200` responses, the tool additionally validates that the response body is **valid JSON** and non-empty where applicable.

## Output Format

The tool uses structured logging for clear, machine-parseable output:

```text
level=info msg="Testing 2 plugins from proposed.yaml"
level=info msg="Testing plugin" id=vultisig-dca-0000 name="Recurring Swaps" url="https://plugin-dca-swap.lb.services.1conf.com"
level=info msg=Success test=recipe-specification plugin_id=vultisig-dca-0000 plugin_name="Recurring Swaps" url="https://plugin-dca-swap.lb.services.1conf.com" id=vultisig-dca-0000
level=info msg=Success test=vault-exist url="https://plugin-dca-swap.lb.services.1conf.com" id=vultisig-dca-0000
level=info msg=Success test=vault-get url="https://plugin-dca-swap.lb.services.1conf.com" id=vultisig-dca-0000
level=info msg=Success test=vault-delete url="https://plugin-dca-swap.lb.services.1conf.com" id=vultisig-dca-0000
level=info msg=Success test=vault-reshare url="https://plugin-dca-swap.lb.services.1conf.com" id=vultisig-dca-0000
level=info msg=Success test=create-policy url="https://plugin-dca-swap.lb.services.1conf.com" id=vultisig-dca-0000
level=info msg=Success test=update-policy url="https://plugin-dca-swap.lb.services.1conf.com" id=vultisig-dca-0000
level=info msg=Success test=delete-policy url="https://plugin-dca-swap.lb.services.1conf.com" id=vultisig-dca-0000
level=info msg="All tests passed" url="https://plugin-dca-swap.lb.services.1conf.com" id=vultisig-dca-0000
level=info msg="Summary: 2 passed, 0 failed"
```

## Building

```bash
go build -o plugin-smoke-test testdata/plugin-smoke-test/main.go
```

## Exit Codes

* `0` – All tests passed
* `1` – One or more tests failed

## CI Integration

This tool is used in `.github/workflows/plugin-smoke-tests.yaml` to automatically test all plugins whenever:

* `proposed.yaml` changes, or
* the smoke test tool / workflow itself changes.

## For Plugin Developers

Your plugin server **must** implement all of the following endpoints for the smoke test to pass.

### 1. Recipe Specification

```http
GET /plugin/recipe-specification
```

Example response:

```json
{
  "plugin_id": "your-plugin-id",
  "plugin_name": "Your Plugin Name",
  "requirements": { },
  "configuration": { },
  "configuration_example": { },
  "supported_resources": [ ],
  "permissions": [ ]
}
```

**Required fields:**

* `plugin_id` (string) – Unique identifier for your plugin
* `plugin_name` (string) – Display name for your plugin
* `requirements` (object) – Plugin requirements specification
* `configuration` (object) – JSON schema for plugin configuration
* `configuration_example` (object or array) – Example configuration matching the schema
* `supported_resources` (array) – List of supported resource types
* `permissions` (array) – List of required permissions

### 2. Vault Endpoints

* `GET /vault/exist/:pluginId/:publicKey`
* `GET /vault/get/:pluginId/:publicKey`
* `POST /vault/reshare`
* `DELETE /vault/:pluginId/:publicKey`

For smoke tests, these endpoints must:

* Be routable
* Return one of `200`, `400`, `401`, `404`
* Return valid JSON bodies when they include a response body

### 3. Policy Endpoints

* `POST /plugin/policy`
* `PUT /plugin/policy`
* `DELETE /plugin/policy/:policyId`

For smoke tests, these endpoints may reject the test payload with `400` / `401`, but must:

* Be routable
* Not crash (no `5xx`)
* Return valid JSON when returning a body
