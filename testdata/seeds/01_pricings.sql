INSERT INTO pricings (id, type, frequency, amount, metric) VALUES 
-- Free pricing for basic plugins
('00000000-0000-0000-0000-000000000001', 'free', NULL, 0.00, 'fixed'),

-- Per-transaction pricing
('00000000-0000-0000-0000-000000000002', 'per-tx', NULL, 100.00, 'fixed'),

-- Monthly recurring pricing
('00000000-0000-0000-0000-000000000003', 'recurring', 'monthly', 1000.00, 'fixed'),

-- Weekly recurring pricing  
('00000000-0000-0000-0000-000000000004', 'recurring', 'weekly', 250.00, 'fixed');