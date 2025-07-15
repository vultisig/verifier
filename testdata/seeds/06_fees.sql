INSERT INTO plugin_policy_billing (id, asset, "type", frequency, start_date, amount, plugin_policy_id)
VALUES ('20000000-2000-2000-0000-000000000222', 'usdc', 'once', NULL, CURRENT_DATE, 0.05, '20000000-0000-0000-0000-000000000002') 
ON CONFLICT (id) DO NOTHING;

INSERT INTO fees (id, plugin_policy_billing_id, amount)
VALUES ('20000000-2000-2000-2222-000000000000', '20000000-2000-2000-0000-000000000222', 0.05)
ON CONFLICT (id) DO NOTHING;