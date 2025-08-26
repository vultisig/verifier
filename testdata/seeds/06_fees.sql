INSERT INTO plugin_policy_billing (id, asset, "type", frequency, start_date, amount, plugin_policy_id)
VALUES ('20000000-2000-2000-0000-000000000222', 'usdc', 'once', NULL, CURRENT_DATE, 50000, '20000000-0000-0000-0000-000000000002') 
ON CONFLICT DO NOTHING;

INSERT INTO plugin_policy_billing (id, asset, "type", frequency, start_date, amount, plugin_policy_id)
VALUES ('20000000-2000-2000-0000-000000000223', 'usdc', 'recurring', 'monthly', '2025-01-01', 30000, '20000000-0000-0000-0000-000000000002') 
ON CONFLICT DO NOTHING;

-- Fee debits with required fields
INSERT INTO fee_debits (id, subtype, plugin_policy_billing_id, public_key, amount)
VALUES ('20000000-2000-2000-2222-000000000000', 'fee', '20000000-2000-2000-0000-000000000222', '027e897b35aa9f9fff223b6c826ff42da37e8169fae7be57cbd38be86938a746c6', 50000)
ON CONFLICT DO NOTHING;

INSERT INTO fee_debits (id, subtype, plugin_policy_billing_id, public_key, amount, charged_at)
VALUES ('00000000-0000-0000-0000-000000000001', 'fee', '20000000-2000-2000-0000-000000000223', '027e897b35aa9f9fff223b6c826ff42da37e8169fae7be57cbd38be86938a746c6', 30000, '2025-02-01')
ON CONFLICT DO NOTHING;
