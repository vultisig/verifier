-- +goose Up
-- Add new recurring sends plugin
INSERT INTO plugins (id, title, description, server_endpoint, category)
VALUES (
    'vultisig-recurring-sends-0000',
    'Recurring Sends',
    'Schedule recurring cryptocurrency transfers',
    'https://dca-send.lb-1.plugins-cluster.nbitslabs.com',
    'automation'
);

-- Update existing DCA plugin to point to swap endpoint and update metadata
UPDATE plugins
SET
    title = 'Recurring Swaps',
    description = 'Schedule recurring cryptocurrency swaps across chains and assets',
    server_endpoint = 'https://dca-swap.lb-1.plugins-cluster.nbitslabs.com'
WHERE id = 'vultisig-dca-0000';

-- +goose Down
