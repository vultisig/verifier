INSERT INTO plugin_policy_billing (id, asset, "type", frequency, start_date, amount, plugin_policy_id)
VALUES ('20000000-2000-2000-0000-000000000222', 'usdc', 'once', NULL, CURRENT_DATE, 50000, '20000000-0000-0000-0000-000000000002') 
ON CONFLICT DO NOTHING;

INSERT INTO plugin_policy_billing (id, asset, "type", frequency, start_date, amount, plugin_policy_id)
VALUES ('20000000-2000-2000-0000-000000000223', 'usdc', 'recurring', 'monthly', '2025-01-01', 30000, '20000000-0000-0000-0000-000000000002') 
ON CONFLICT DO NOTHING;

INSERT INTO plugin_policy_billing (id, asset, "type", frequency, start_date, amount, plugin_policy_id)
VALUES ('00000000-0000-0000-0000-000000000999', 'usdc', 'recurring', 'monthly', '2025-01-01', 30000, '42940b2a-214b-487f-9195-aa33dc44ae5c') 
ON CONFLICT DO NOTHING;