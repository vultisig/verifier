INSERT INTO tags (id, name, created_at) VALUES 
('00000000-0000-0000-0000-000000000001', 'Trading', NOW()),
('00000000-0000-0000-0000-000000000002', 'Operations', NOW()) ON CONFLICT (id) DO NOTHING;

INSERT INTO plugin_tags (plugin_id, tag_id) VALUES 
('vultisig-dca-0000', '00000000-0000-0000-0000-000000000001'),
('vultisig-payroll-0000', '00000000-0000-0000-0000-000000000002') ON CONFLICT (plugin_id, tag_id) DO NOTHING;