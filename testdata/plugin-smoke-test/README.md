# Plugin Smoke Test Tool

Simple Go-based tool to test plugin HTTP endpoints without requiring access to plugin source code.

## Usage

### Test all plugins from proposed.yaml

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
1. ✅ `GET /plugin/recipe-specification` - Validates plugin metadata, requirements, configuration schema, example config, supported resources, and permissions
2. ✅ `GET /vault/exist/:pluginId/:publicKey` - Check vault existence
3. ✅ `GET /vault/get/:pluginId/:publicKey` - Retrieve vault metadata
4. ✅ `DELETE /vault/:pluginId/:publicKey` - Delete vault
5. ✅ `POST /vault/reshare` - Create/reshare vault
6. ✅ `POST /plugin/policy` - Create policy (requires auth)
7. ✅ `PUT /plugin/policy` - Update policy (requires auth)
8. ✅ `DELETE /plugin/policy/:policyId` - Delete policy (requires auth)

All endpoints must respond with valid HTTP status codes (200, 400, 401, 404) to indicate the endpoint exists. They may fail with validation errors or authentication errors, but must not return 500 or other unexpected status codes.

## Output Format

```
🧪 Testing 2 plugins from proposed.yaml

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Plugin 1/2
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  URL:   https://plugin-dca-swap.lb.services.1conf.com
  ID:    vultisig-dca-0000
  Title: DCA Plugin

  [1/8] GET /plugin/recipe-specification ... ✅
        Plugin ID: vultisig-dca-0000
        Plugin Name: Recurring Swaps
  [2/8] GET /vault/exist/:pluginId/:publicKey ... ✅
  [3/8] GET /vault/get/:pluginId/:publicKey ... ✅
  [4/8] DELETE /vault/:pluginId/:publicKey ... ✅
  [5/8] POST /vault/reshare ... ✅
  [6/8] POST /plugin/policy ... ✅
  [7/8] PUT /plugin/policy ... ✅
  [8/8] DELETE /plugin/policy/:policyId ... ✅

  ✅ All tests passed!

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Summary: 2 passed, 0 failed
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Building

```bash
go build -o plugin-smoke-test testdata/plugin-smoke-test/main.go
```

## Exit Codes

- `0` - All tests passed
- `1` - One or more tests failed

## CI Integration

This tool is used in `.github/workflows/plugin-smoke-tests.yaml` to automatically test all plugins when `proposed.yaml` is modified.

## For Plugin Developers

Your plugin server must implement all of the following endpoints:

### 1. Recipe Specification (Required)
```
GET /plugin/recipe-specification
```

Returns:
```json
{
  "plugin_id": "your-plugin-id",
  "plugin_name": "Your Plugin Name",
  "requirements": { ... },
  "configuration": { ... },
  "configuration_example": { ... },
  "supported_resources": [ ... ],
  "permissions": [ ... ]
}
```

**Required fields:**
- `plugin_id` (string) - Unique identifier for your plugin
- `plugin_name` (string) - Display name for your plugin
- `requirements` (object) - Plugin requirements specification
- `configuration` (object) - JSON schema for plugin configuration
- `configuration_example` (object or array) - Example configuration instance matching the configuration schema
- `supported_resources` (array) - List of supported resource types
- `permissions` (array) - List of required permissions

### 2. Vault Endpoints (Required)
- `GET /vault/exist/:pluginId/:publicKey` - Check if vault exists
- `GET /vault/get/:pluginId/:publicKey` - Retrieve vault metadata
- `POST /vault/reshare` - Create or reshare vault
- `DELETE /vault/:pluginId/:publicKey` - Delete vault

### 3. Policy Endpoints (Required)
- `POST /plugin/policy` - Create a new policy
- `PUT /plugin/policy` - Update an existing policy
- `DELETE /plugin/policy/:policyId` - Delete a policy

All endpoints must return valid HTTP status codes (200, 400, 401, 404) to indicate proper implementation. Authentication errors (401) and validation errors (400) are acceptable, but unexpected errors (500+) will fail the test.
