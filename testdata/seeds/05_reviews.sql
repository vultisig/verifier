INSERT INTO reviews (id, plugin_id, public_key, rating, comment, created_at, updated_at) VALUES 
 ('86e45f44-ec0f-427c-8c90-5f3f2ef2e101',
  'vultisig-dca-0000',
  '0x000000000000000000000000000000000000dEaD',
  5,
  'Hello world',
  NOW(),
  NOW()
) ON CONFLICT (id) DO NOTHING;