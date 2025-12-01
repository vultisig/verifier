-- +goose Up
-- +goose StatementBegin
BEGIN;

UPDATE plugins
SET
    logo_url = 'https://raw.githubusercontent.com/vultisig/verifier/main/assets/plugins/recurring_send/icon.jpg',
    thumbnail_url = 'https://raw.githubusercontent.com/vultisig/verifier/main/assets/plugins/recurring_send/banner.jpg'
WHERE id = 'vultisig-recurring-sends-0000';

UPDATE plugins
SET
    logo_url = 'https://raw.githubusercontent.com/vultisig/verifier/main/assets/plugins/recurring_swap/icon.jpg',
    thumbnail_url = 'https://raw.githubusercontent.com/vultisig/verifier/main/assets/plugins/recurring_swap/banner.jpg'
WHERE id = 'vultisig-dca-0000';

UPDATE plugins
SET
    logo_url = 'https://raw.githubusercontent.com/vultisig/verifier/main/assets/plugins/payment/icon.jpg'
    thumbnail_url = 'https://raw.githubusercontent.com/vultisig/verifier/main/assets/plugins/payment/banner.jpg'
WHERE id = 'vultisig-fees-feee';

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

UPDATE plugins
SET
    logo_url = '',
    thumbnail_url = ''
WHERE id IN ('vultisig-recurring-sends-0000', 'vultisig-dca-0000');

UPDATE plugins
SET
    logo_url = ''
WHERE id = 'vultisig-fees-feee';

COMMIT;
-- +goose StatementEnd
