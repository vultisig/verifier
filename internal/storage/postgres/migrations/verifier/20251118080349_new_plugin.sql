-- +goose Up
INSERT INTO plugins (id, title, description, server_endpoint, category)
VALUES
-- DCA Plugin with per-transaction pricing
('vultisig-dca-0001',
 'Vultisig Recurring Sends',
 'Recurring Sends. Automatically execute recurring transfer orders based on predefined schedules and strategies.',
 'http://dca-send.lb-1.plugins-cluster.nbitslabs.com',
 'plugin'),
ON CONFLICT (id) DO NOTHING;

INSERT INTO pricings (id, plugin_id, asset, type, frequency, amount, metric)
VALUES

-- Per-transaction pricing 1c
('00000000-0000-0000-0000-000000000006', 'vultisig-dca-0001', 'usdc', 'per-tx', NULL, 1e4,
 'fixed') ON CONFLICT (id) DO NOTHING;

INSERT INTO plugin_tags (plugin_id, tag_id)
VALUES ('vultisig-dca-0001', '00000000-0000-0000-0000-000000000001') ON CONFLICT (plugin_id, tag_id) DO NOTHING;

-- +goose Down
DELETE
FROM pricings
WHERE id = '00000000-0000-0000-0000-000000000006';

DELETE
FROM plugin_tags
WHERE plugin_id = 'vultisig-dca-0001';

DELETE
FROM plugins
WHERE id = 'vultisig-dca-0001';