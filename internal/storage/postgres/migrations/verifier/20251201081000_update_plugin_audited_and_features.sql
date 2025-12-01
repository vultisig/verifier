-- +goose Up
-- +goose StatementBegin
BEGIN;

UPDATE plugins
SET
    features = '["Automated Recurring Sends","Works Across All Supported Assets"," Fully Flexible Scheduling","Automatic Execution"]',
    audited = true 
WHERE id = 'vultisig-recurring-sends-0000';

UPDATE plugins
SET
    features = '["Automated Dollar Cost Averaging processing","Recurring Swaps from any Vultisig supported Asset to any Asset"," Fully Flexible Scheduling","Automatic Execution"]',
    audited = true
WHERE id = 'vultisig-dca-0000';

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

UPDATE plugins
SET
    features = '',
    audited = false
WHERE id IN ('vultisig-recurring-sends-0000', 'vultisig-dca-0000');

COMMIT;
-- +goose StatementEnd
