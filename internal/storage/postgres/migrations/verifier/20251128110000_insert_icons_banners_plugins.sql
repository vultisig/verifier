-- +goose Up
-- +goose StatementBegin
BEGIN;

UPDATE plugins
SET
    logo_url = 'https://raw.githubusercontent.com/vultisig/verifier/main/assets/plugins/recurring_send/icon.svg',
    thumbnail_url = 'https://raw.githubusercontent.com/vultisig/verifier/main/assets/plugins/recurring_send/banner.png'
WHERE id = 'vultisig-recurring-sends-0000';

UPDATE plugins
SET
    logo_url = 'https://raw.githubusercontent.com/vultisig/verifier/main/assets/plugins/recurring_swap/icon.svg',
    thumbnail_url = 'https://raw.githubusercontent.com/vultisig/verifier/main/assets/plugins/recurring_swap/banner.png'
WHERE id = 'vultisig-dca-0000';

UPDATE plugins
SET
    logo_url = 'https://raw.githubusercontent.com/vultisig/verifier/main/assets/plugins/payment/icon.svg'
WHERE id = 'vultisig-fees-feee';

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

UPDATE plugins
SET
    logo_url = '',
    thumbnail_url = '',
WHERE id IN ('vultisig-recurring-sends-0000', 'vultisig-dca-0000', 'vultisig-fees-feee');

COMMIT;
-- +goose StatementEnd
