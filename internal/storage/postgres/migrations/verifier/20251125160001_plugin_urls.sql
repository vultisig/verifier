-- +goose Up
UPDATE plugins
SET
    server_endpoint = 'https://plugin-dca-swap.lb.services.1conf.com'
WHERE id = 'vultisig-dca-0000';

UPDATE plugins
SET
    server_endpoint = 'https://plugin-dca-send.lb.services.1conf.com'
WHERE id = 'vultisig-recurring-sends-0000';

-- +goose Down
