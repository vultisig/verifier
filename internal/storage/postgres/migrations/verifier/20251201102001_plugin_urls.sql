-- +goose Up
UPDATE plugins
SET
    server_endpoint = 'https://plugin-dca-swap.prod.plugins.vultisig.com'
WHERE id = 'vultisig-dca-0000';

UPDATE plugins
SET
    server_endpoint = 'https://plugin-dca-send.prod.plugins.vultisig.com'
WHERE id = 'vultisig-recurring-sends-0000';

UPDATE plugins
SET
    server_endpoint = 'https://plugin-fee-server.prod.plugins.vultisig.com'
WHERE id = 'vultisig-fees-feee';

-- +goose Down
