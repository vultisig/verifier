INSERT INTO plugin_apikey (id, plugin_id, apikey, created_at, expires_at, status) VALUES
(
 gen_random_uuid(),
 'vultisig-payroll-0000',
 'localhost-apikey',
 now(),
 null,
 1
)
