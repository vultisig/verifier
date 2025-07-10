
INSERT INTO pricings (id, plugin_id, asset, type, frequency, amount, metric) VALUES 

-- Per-transaction pricing 1c
('00000000-0000-0000-0000-000000000002', 'vultisig-dca-0000', 'usdc', 'per-tx', NULL, 1e4, 'fixed'),-
-- Single pricing for payroll plugin 5c
('20000000-0000-0000-0000-000000000003', 'vultisig-payroll-0000', 'usdc', 'once', NULL, 5e4, 'fixed'), 
-- Monthly recurring pricing 3c
('00000000-0000-0000-0000-000000000003', 'vultisig-payroll-0000', 'usdc', 'recurring', 'monthly', 3e4, 'fixed'); 