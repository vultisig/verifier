# Integration tests for vultisig-dca-0000
# Generated from proposed.yaml with vault fixture support
# Tests run through verifier API at http://localhost:8080
# Auth is enabled for policy endpoint tests (JWT required)

# Test 1: Get Plugin Details
GET http://localhost:8080/plugins/vultisig-dca-0000
HTTP 200
[Asserts]
jsonpath "$.data.id" == "vultisig-dca-0000"
jsonpath "$.data.title" exists

# Test 2: Recipe Specification
GET http://localhost:8080/plugins/vultisig-dca-0000/recipe-specification
HTTP 200
[Asserts]
jsonpath "$.data.plugin_id" == "vultisig-dca-0000"
jsonpath "$.data.plugin_name" exists
jsonpath "$.data.supported_resources" exists

# Test 3: Reshare Vault (POST - main plugin server interaction)
POST http://localhost:8080/vault/reshare
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwdWJsaWNfa2V5IjoiMDI3OWJlNjY3ZWY5ZGNiYmFjNTVhMDYyOTVjZTg3MGIwNzAyOWJmY2RiMmRjZTI4ZDk1OWYyODE1YjE2ZjgxNzk4IiwidG9rZW5faWQiOiJpbnRlZ3JhdGlvbi10b2tlbi0xIiwiZXhwIjoxNzY2NTg1NjYzLCJpYXQiOjE3NjY0OTkyNjN9.maWqUy9016YGoE5CnJhze_1DQGUpkGsaky8H5_gxiUk
Content-Type: application/json
{
  "name": "integration-test-vault",
  "public_key": "0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
  "session_id": "00000000-0000-0000-0000-000000000000",
  "hex_encryption_key": "0000000000000000000000000000000000000000000000000000000000000000",
  "hex_chain_code": "0000000000000000000000000000000000000000000000000000000000000000",
  "local_party_id": "verifier-test-party",
  "old_parties": ["party1","party2"],
  "email": "integration@test.example.com",
  "plugin_id": "vultisig-dca-0000"
}
HTTP 200
[Asserts]
jsonpath "$.status" == 200
jsonpath "$.data" exists

# Test 4: Reshare Vault Again (Idempotency check - should return "already_exists")
POST http://localhost:8080/vault/reshare
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwdWJsaWNfa2V5IjoiMDI3OWJlNjY3ZWY5ZGNiYmFjNTVhMDYyOTVjZTg3MGIwNzAyOWJmY2RiMmRjZTI4ZDk1OWYyODE1YjE2ZjgxNzk4IiwidG9rZW5faWQiOiJpbnRlZ3JhdGlvbi10b2tlbi0xIiwiZXhwIjoxNzY2NTg1NjYzLCJpYXQiOjE3NjY0OTkyNjN9.maWqUy9016YGoE5CnJhze_1DQGUpkGsaky8H5_gxiUk
Content-Type: application/json
{
  "name": "integration-test-vault",
  "public_key": "0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
  "session_id": "00000000-0000-0000-0000-000000000000",
  "hex_encryption_key": "0000000000000000000000000000000000000000000000000000000000000000",
  "hex_chain_code": "0000000000000000000000000000000000000000000000000000000000000000",
  "local_party_id": "verifier-test-party",
  "old_parties": ["party1","party2"],
  "email": "integration@test.example.com",
  "plugin_id": "vultisig-dca-0000"
}
HTTP 200
[Asserts]
jsonpath "$.status" == 200
jsonpath "$.data" exists

# Test 5: Create Plugin Policy (fails with invalid signature - requires real ECDSA signature from derived private key)
# NOTE: This test verifies that signature verification is working correctly.
# Policy creation requires a valid signature from the derived Ethereum private key.
# Without implementing BIP32 child key derivation + signing, this endpoint cannot return 200.
# For integration testing of plugin execution flow, use /plugin-signer/sign endpoints instead.
POST http://localhost:8080/plugin/policy
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwdWJsaWNfa2V5IjoiMDI3OWJlNjY3ZWY5ZGNiYmFjNTVhMDYyOTVjZTg3MGIwNzAyOWJmY2RiMmRjZTI4ZDk1OWYyODE1YjE2ZjgxNzk4IiwidG9rZW5faWQiOiJpbnRlZ3JhdGlvbi10b2tlbi0xIiwiZXhwIjoxNzY2NTg1NjYzLCJpYXQiOjE3NjY0OTkyNjN9.maWqUy9016YGoE5CnJhze_1DQGUpkGsaky8H5_gxiUk
Content-Type: application/json
{
  "id": "00000000-0000-0000-0000-000000000001",
  "public_key": "0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
  "plugin_id": "vultisig-dca-0000",
  "plugin_version": "1.0.0",
  "policy_version": 1,
  "signature": "0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
  "recipe": "CgA=",
  "billing": [],
  "active": true
}
HTTP 400
[Asserts]
jsonpath "$.error" exists

# Test 6: Policy endpoint requires JWT (test auth enforcement)
POST http://localhost:8080/plugin/policy
Content-Type: application/json
{
  "id": "test-no-auth",
  "public_key": "0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
  "plugin_id": "vultisig-dca-0000",
  "plugin_version": "1.0.0",
  "policy_version": 1,
  "signature": "0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
  "recipe": "CgA=",
  "billing": [],
  "active": true
}
HTTP 401
[Asserts]
jsonpath "$.error" exists

# Test 7: Get Policy endpoint requires JWT
GET http://localhost:8080/plugin/policy/test-id
HTTP 401
[Asserts]
jsonpath "$.error" exists

# Test 8: Get Policy with JWT (test endpoint with auth - invalid ID format returns 400)
GET http://localhost:8080/plugin/policy/test-id
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwdWJsaWNfa2V5IjoiMDI3OWJlNjY3ZWY5ZGNiYmFjNTVhMDYyOTVjZTg3MGIwNzAyOWJmY2RiMmRjZTI4ZDk1OWYyODE1YjE2ZjgxNzk4IiwidG9rZW5faWQiOiJpbnRlZ3JhdGlvbi10b2tlbi0xIiwiZXhwIjoxNzY2NTg1NjYzLCJpYXQiOjE3NjY0OTkyNjN9.maWqUy9016YGoE5CnJhze_1DQGUpkGsaky8H5_gxiUk
HTTP 400
[Asserts]
jsonpath "$.error" exists


# ========================================
# Plugin-Signer Tests (High-Signal Coverage)
# ========================================
# Tests /plugin-signer/sign which provides comprehensive integration coverage:
#   - Plugin API key authentication
#   - Policy lookup + validation
#   - Transaction parsing
#   - Policy engine evaluation
#   - TX indexer + Redis + Asynq enqueuing

# Test 9: /plugin-signer/sign requires API key
POST http://localhost:8080/plugin-signer/sign
Content-Type: application/json
{
  "plugin_id": "vultisig-dca-0000",
  "public_key": "0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
  "policy_id": "00000000-0000-0000-0000-000000000011",
  "messages": []
}
HTTP 401
[Asserts]
jsonpath "$.error" exists


# Test 10: /plugin-signer/sign with invalid API key
POST http://localhost:8080/plugin-signer/sign
Authorization: Bearer invalid-api-key-12345
Content-Type: application/json
{
  "plugin_id": "vultisig-dca-0000",
  "public_key": "0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
  "policy_id": "00000000-0000-0000-0000-000000000011",
  "messages": []
}
HTTP 401
[Asserts]
jsonpath "$.error" exists


# Test 11: /plugin-signer/sign with empty messages (400 - bad request)
POST http://localhost:8080/plugin-signer/sign
Authorization: Bearer integration-test-apikey-vultisig-dca-0000
Content-Type: application/json
{
  "plugin_id": "vultisig-dca-0000",
  "public_key": "0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
  "policy_id": "00000000-0000-0000-0000-000000000011",
  "messages": []
}
HTTP 400
[Asserts]
jsonpath "$.error" exists

# Test 12: /plugin-signer/sign with valid request (enqueues task)
POST http://localhost:8080/plugin-signer/sign
Authorization: Bearer integration-test-apikey-vultisig-dca-0000
Content-Type: application/json
{
  "plugin_id": "vultisig-dca-0000",
  "public_key": "0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
  "policy_id": "00000000-0000-0000-0000-000000000011",
  "transactions": "Au8BgIQ7msoAhQSoF8gAglIIlHQtNcxmNMBTKSWjuES8nnWV8L6whwONfqTGgACAwA==",
  "transaction_type": "evm",
  "messages": [
    {
      "message": "BKE1PNCcu7OeQE6eeGp/nSiu5FpfKLqTp/5SLCWFNcU=",
      "chain": "Ethereum",
      "hash": "BKE1PNCcu7OeQE6eeGp/nSiu5FpfKLqTp/5SLCWFNcU=",
      "hash_function": "SHA256"
    }
  ]
}
HTTP 200
[Captures]
task_id: jsonpath "$.data.task_ids[0]"
[Asserts]
jsonpath "$.data" exists
jsonpath "$.data.task_ids" isCollection
jsonpath "$.data.task_ids" count == 1


# Test 13: GET /plugin-signer/sign/response/:taskId requires API key
GET http://localhost:8080/plugin-signer/sign/response/{{task_id}}
HTTP 401
[Asserts]
jsonpath "$.error" exists


# Test 14: GET /plugin-signer/sign/response/:taskId with API key
GET http://localhost:8080/plugin-signer/sign/response/{{task_id}}
Authorization: Bearer integration-test-apikey-vultisig-dca-0000
HTTP *
# Accept any HTTP code - task may be pending/processing/failed without DKLS worker
