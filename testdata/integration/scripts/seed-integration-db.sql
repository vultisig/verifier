-- Seed database for /plugin-signer/sign integration tests
-- This script seeds:
--   1. Plugin API keys (for PluginAuthMiddleware)
--   2. Plugin policies (for policy lookup + validation)
--
-- These records enable testing of the plugin execution flow WITHOUT requiring
-- valid cryptographic signatures for policy creation.

-- Load fixture data for consistent public key
-- Public key: 0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798
-- (from testdata/integration-vault/fixture.json)

\set VAULT_PUBKEY '0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798'

-- 1. Seed plugin API keys for each plugin in proposed.yaml
-- These allow plugins to authenticate via PluginAuthMiddleware

-- DCA Plugin
INSERT INTO plugin_apikey (id, plugin_id, apikey, status, expires_at)
VALUES (
  'dca-apikey-00000000-0000-0000-0000-000000000001'::uuid,
  'vultisig-dca-0000',
  'integration-test-apikey-vultisig-dca-0000',  -- Matches format from seed_integration_db.go
  1,  -- status = 1 (active)
  NOW() + INTERVAL '24 hours'
)
ON CONFLICT (apikey) DO UPDATE
SET
  plugin_id  = EXCLUDED.plugin_id,
  status     = 1,
  expires_at = EXCLUDED.expires_at;

-- Recurring Sends Plugin
INSERT INTO plugin_apikey (id, plugin_id, apikey, status, expires_at)
VALUES (
  'recurring-apikey-0000-0000-0000-000000000001'::uuid,
  'vultisig-recurring-sends-0000',
  'integration-test-apikey-vultisig-recurring-sends-0000',  -- Matches format from seed_integration_db.go
  1,  -- status = 1 (active)
  NOW() + INTERVAL '24 hours'
)
ON CONFLICT (apikey) DO UPDATE
SET
  plugin_id  = EXCLUDED.plugin_id,
  status     = 1,
  expires_at = EXCLUDED.expires_at;


-- 2. Seed plugin policies for testing
-- These policies will be looked up by /plugin-signer/sign endpoint
-- NOTE: Signatures are dummy values (all zeros) because we're seeding directly in DB,
-- bypassing the policy creation endpoint that requires cryptographic verification.

-- DCA Policy
INSERT INTO plugin_policies (id, public_key, plugin_id, plugin_version, policy_version, signature, recipe, active)
VALUES (
  '00000000-0000-0000-0000-000000000011'::uuid,
  :'VAULT_PUBKEY',
  'vultisig-dca-0000',
  '1.0.0',
  1,
  '0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000',  -- Dummy signature
  'ChZwZXJtaXNzaXZlLXRlc3QtcG9saWN5EhZQZXJtaXNzaXZlIFRlc3QgUG9saWN5GiNBbGxvd3MgYWxsIHRyYW5zYWN0aW9ucyBmb3IgdGVzdGluZyABKhBpbnRlZ3JhdGlvbi10ZXN0MlAKFWV0aGVyZXVtLmV0aC50cmFuc2ZlchABGhhBbGxvdyBFdGhlcmV1bSB0cmFuc2ZlcnMyG2FsbG93LWV0aGVyZXVtLWV0aC10cmFuc2ZlcjJRChdldGhlcmV1bS5lcmMyMC50cmFuc2ZlchABGhVBbGxvdyBFUkMyMCB0cmFuc2ZlcnMyHWFsbG93LWV0aGVyZXVtLWVyYzIwLXRyYW5zZmVyMk8KFmV0aGVyZXVtLmVyYzIwLmFwcHJvdmUQARoVQWxsb3cgRVJDMjAgYXBwcm92YWxzMhxhbGxvdy1ldGhlcmV1bS1lcmMyMC1hcHByb3Zl',  -- Permissive policy with EVM rules in base64
  true
)
ON CONFLICT (id) DO UPDATE
SET
  public_key      = EXCLUDED.public_key,
  plugin_id       = EXCLUDED.plugin_id,
  plugin_version  = EXCLUDED.plugin_version,
  policy_version  = EXCLUDED.policy_version,
  signature       = EXCLUDED.signature,
  recipe          = EXCLUDED.recipe,
  active          = EXCLUDED.active,
  updated_at      = NOW();

-- Recurring Sends Policy
INSERT INTO plugin_policies (id, public_key, plugin_id, plugin_version, policy_version, signature, recipe, active)
VALUES (
  '00000000-0000-0000-0000-000000000012'::uuid,
  :'VAULT_PUBKEY',
  'vultisig-recurring-sends-0000',
  '1.0.0',
  1,
  '0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000',  -- Dummy signature
  'ChZwZXJtaXNzaXZlLXRlc3QtcG9saWN5EhZQZXJtaXNzaXZlIFRlc3QgUG9saWN5GiNBbGxvd3MgYWxsIHRyYW5zYWN0aW9ucyBmb3IgdGVzdGluZyABKhBpbnRlZ3JhdGlvbi10ZXN0MlAKFWV0aGVyZXVtLmV0aC50cmFuc2ZlchABGhhBbGxvdyBFdGhlcmV1bSB0cmFuc2ZlcnMyG2FsbG93LWV0aGVyZXVtLWV0aC10cmFuc2ZlcjJRChdldGhlcmV1bS5lcmMyMC50cmFuc2ZlchABGhVBbGxvdyBFUkMyMCB0cmFuc2ZlcnMyHWFsbG93LWV0aGVyZXVtLWVyYzIwLXRyYW5zZmVyMk8KFmV0aGVyZXVtLmVyYzIwLmFwcHJvdmUQARoVQWxsb3cgRVJDMjAgYXBwcm92YWxzMhxhbGxvdy1ldGhlcmV1bS1lcmMyMC1hcHByb3Zl',  -- Permissive policy with EVM rules in base64
  true
)
ON CONFLICT (id) DO UPDATE
SET
  public_key      = EXCLUDED.public_key,
  plugin_id       = EXCLUDED.plugin_id,
  plugin_version  = EXCLUDED.plugin_version,
  policy_version  = EXCLUDED.policy_version,
  signature       = EXCLUDED.signature,
  recipe          = EXCLUDED.recipe,
  active          = EXCLUDED.active,
  updated_at      = NOW();


-- Summary
SELECT
  'Seeded Plugin API Keys' as summary,
  COUNT(*) as count
FROM plugin_apikey
WHERE apikey LIKE 'integration-test-apikey-%';

SELECT
  'Seeded Plugin Policies' as summary,
  COUNT(*) as count
FROM plugin_policies
WHERE public_key = :'VAULT_PUBKEY';
