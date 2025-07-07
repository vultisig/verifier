INSERT INTO pricings (id, type, frequency, amount, metric) VALUES 
-- Free pricing for basic plugins
('00000000-0000-0000-0000-000000000001', 'free', NULL, 0.00, 'fixed'),

-- Per-transaction pricing
('00000000-0000-0000-0000-000000000002', 'per-tx', NULL, 0.05, 'fixed'),

-- Monthly recurring pricing
('00000000-0000-0000-0000-000000000003', 'recurring', 'monthly', 0.50, 'fixed'),

-- Weekly recurring pricing  
('00000000-0000-0000-0000-000000000004', 'recurring', 'weekly', 0.20, 'fixed')
ON CONFLICT (id) DO NOTHING;