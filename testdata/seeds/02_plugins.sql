INSERT INTO plugins (id, title, description, server_endpoint, pricing_id, category) VALUES 
-- DCA Plugin with per-transaction pricing
(
    'vultisig-dca-0000', 
    'Vultisig DCA Plugin', 
    'Dollar Cost Averaging automation for cryptocurrency investments. Automatically execute recurring buy orders based on predefined schedules and strategies.',
    'http://dca-server:8081',
    '00000000-0000-0000-0000-000000000002',
    'plugin'
),

-- Payroll Plugin with monthly pricing
(
    'vultisig-payroll-0000',
    'Vultisig Payroll Plugin', 
    'Automated payroll system for cryptocurrency payments. Handle employee payments, tax calculations, and compliance reporting.',
    'http://payroll-server:8080',
    '00000000-0000-0000-0000-000000000003', 
    'plugin'
),

-- Fee Management Plugin with free pricing
(
    'vultisig-fees-feee',
    'Vultisig Fee Management Plugin',
    'Fee collection and management system. Track, calculate, and distribute fees across different protocols and services.',
    'http://fee-server:8080',
    '00000000-0000-0000-0000-000000000001',
    'plugin'
) ON CONFLICT (id) DO NOTHING;