INSERT INTO plugin_policy_billing (id, asset, "type", frequency, start_date, amount, plugin_policy_id)
VALUES ('20000000-2000-2000-0000-000000000222', 'usdc', 'once', NULL, CURRENT_DATE, 50000, '20000000-0000-0000-0000-000000000002') 
ON CONFLICT (id) DO NOTHING;

INSERT INTO plugin_policy_billing (id, asset, "type", frequency, start_date, amount, plugin_policy_id)
VALUES ('20000000-2000-2000-0000-000000000223', 'usdc', 'recurring', 'monthly', '2025-01-01', 30000, '20000000-0000-0000-0000-000000000002') 
ON CONFLICT (id) DO NOTHING;

-- Fee debits with required fields
INSERT INTO fee_debits (id, subtype, plugin_policy_billing_id, public_key, amount)
VALUES ('20000000-2000-2000-2222-000000000000', 'fee', '20000000-2000-2000-0000-000000000222', '027e897b35aa9f9fff223b6c826ff42da37e8169fae7be57cbd38be86938a746c6', 50000)
ON CONFLICT (id) DO NOTHING;

INSERT INTO fee_debits (id, subtype, plugin_policy_billing_id, public_key, amount, charged_at)
VALUES ('00000000-0000-0000-0000-000000000001', 'fee', '20000000-2000-2000-0000-000000000223', '027e897b35aa9f9fff223b6c826ff42da37e8169fae7be57cbd38be86938a746c6', 30000, '2025-02-01')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fee_debits (id, subtype, plugin_policy_billing_id, public_key, amount, charged_at)
VALUES ('00000000-0000-0000-0000-000000000004', 'fee', '20000000-2000-2000-0000-000000000223', '027e897b35aa9f9fff223b6c826ff42da37e8169fae7be57cbd38be86938a746c6', 30000, '2025-03-01')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fee_debits (id, subtype, plugin_policy_billing_id, public_key, amount, charged_at)
VALUES ('00000000-0000-0000-0000-000000000006', 'fee', '20000000-2000-2000-0000-000000000223', '027e897b35aa9f9fff223b6c826ff42da37e8169fae7be57cbd38be86938a746c6', 30000, '2025-04-01')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fee_debits (id, subtype, plugin_policy_billing_id, public_key, amount, charged_at)
VALUES ('00000000-0000-0000-0000-000000000008', 'fee', '20000000-2000-2000-0000-000000000223', '027e897b35aa9f9fff223b6c826ff42da37e8169fae7be57cbd38be86938a746c6', 30000, '2025-05-01')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fee_debits (id, subtype, plugin_policy_billing_id, public_key, amount, charged_at)
VALUES ('00000000-0000-0000-0000-000000000010', 'fee', '20000000-2000-2000-0000-000000000223', '027e897b35aa9f9fff223b6c826ff42da37e8169fae7be57cbd38be86938a746c6', 30000, '2025-06-01')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fee_debits (id, subtype, plugin_policy_billing_id, public_key, amount, charged_at)
VALUES ('00000000-0000-0000-0000-000000000012', 'fee', '20000000-2000-2000-0000-000000000223', '027e897b35aa9f9fff223b6c826ff42da37e8169fae7be57cbd38be86938a746c6', 30000, '2025-07-01')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fee_debits (id, subtype, plugin_policy_billing_id, public_key, amount, charged_at)
VALUES ('00000000-0000-0000-0000-000000000014', 'fee', '20000000-2000-2000-0000-000000000223', '027e897b35aa9f9fff223b6c826ff42da37e8169fae7be57cbd38be86938a746c6', 30000, '2025-08-01')
ON CONFLICT (id) DO NOTHING;

-- Fee credits with required fields (negative amounts)
INSERT INTO fee_credits (id, subtype, public_key, amount)
VALUES ('00000000-0000-0000-0000-000000000003', 'fee_transacted', '027e897b35aa9f9fff223b6c826ff42da37e8169fae7be57cbd38be86938a746c6', 30000)
ON CONFLICT (id) DO NOTHING;

INSERT INTO fee_credits (id, subtype, public_key, amount)
VALUES ('00000000-0000-0000-0000-000000000005', 'fee_transacted', '027e897b35aa9f9fff223b6c826ff42da37e8169fae7be57cbd38be86938a746c6', 30000)
ON CONFLICT (id) DO NOTHING;

INSERT INTO fee_credits (id, subtype, public_key, amount)
VALUES ('00000000-0000-0000-0000-000000000007', 'fee_transacted', '027e897b35aa9f9fff223b6c826ff42da37e8169fae7be57cbd38be86938a746c6', 30000)
ON CONFLICT (id) DO NOTHING;

INSERT INTO fee_credits (id, subtype, public_key, amount)
VALUES ('00000000-0000-0000-0000-000000000009', 'fee_transacted', '027e897b35aa9f9fff223b6c826ff42da37e8169fae7be57cbd38be86938a746c6', 30000)
ON CONFLICT (id) DO NOTHING;

INSERT INTO fee_credits (id, subtype, public_key, amount)
VALUES ('00000000-0000-0000-0000-000000000011', 'fee_transacted', '027e897b35aa9f9fff223b6c826ff42da37e8169fae7be57cbd38be86938a746c6', 30000)
ON CONFLICT (id) DO NOTHING;

INSERT INTO fee_credits (id, subtype, public_key, amount)
VALUES ('00000000-0000-0000-0000-000000000013', 'fee_transacted', '027e897b35aa9f9fff223b6c826ff42da37e8169fae7be57cbd38be86938a746c6', 30000)
ON CONFLICT (id) DO NOTHING;

INSERT INTO fee_credits (id, subtype, public_key, amount)
VALUES ('00000000-0000-0000-0000-000000000015', 'fee_transacted', '027e897b35aa9f9fff223b6c826ff42da37e8169fae7be57cbd38be86938a746c6', 30000)
ON CONFLICT (id) DO NOTHING;