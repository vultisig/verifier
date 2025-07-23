INSERT INTO plugins (id, title, description, server_endpoint, category) VALUES
-- Copytrader Plugin
(
    'vultisig-copytrader-0000',
    'Vultisig Copy Trading Plugin',
    'Automatic copy trading system. Repeats UniswapV2 swaps for the selected address.',
    'http://copytrader:8080',
    'plugin'
) ON CONFLICT (id) DO NOTHING;