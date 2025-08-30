INSERT INTO plugins (id, title, description, server_endpoint, category) VALUES 
-- DCA Plugin with per-transaction pricing
(
    'vultisig-dca-0000', 
    'Vultisig DCA Plugin', 
    'Dollar Cost Averaging automation for cryptocurrency investments. Automatically execute recurring buy orders based on predefined schedules and strategies.',
    'http://dca-server:8080',
    'plugin'
),

-- Payroll Plugin with monthly pricing
(
    'vultisig-payroll-0000',
    'Vultisig Payroll Plugin', 
    'Automated payroll system for cryptocurrency payments. Handle employee payments, tax calculations, and compliance reporting.',
    'http://payroll-server:8080',
    'plugin'
),

-- Copytrader Plugin with monthly pricing
(
    'vultisig-copytrader-0000',
    'Vultisig Copytrader Plugin',
    'Copytrader',
    'http://copytrader-server:8080',
    'plugin'
),

-- Fee Management Plugin with free pricing
(
    'vultisig-fees-feee',
    'Vultisig Fee Management Plugin',
    'Fee collection and management system. Track, calculate, and distribute fees across different protocols and services.',
    'http://fee-server:8080',
    'plugin'
),

-- NBits Labs Merkle Plugin with free pricing
(
    'nbits-labs-merkle-e93d',
    'NBits Labs Merkle Plugin',
    'Merkle tree implementation for efficient data storage and retrieval.',
    'http://localhost:8089',
    'plugin'
) ON CONFLICT (id) DO NOTHING;