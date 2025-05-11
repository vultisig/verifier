-- +goose Up
-- +goose StatementBegin
CREATE TYPE plugin_id AS ENUM (
    'vultisig-dca-0000',
    'vultisig-payroll-0000'
);

ALTER TABLE plugin_policies
    DROP COLUMN plugin_id;

ALTER TABLE plugin_policies
    ADD COLUMN plugin_id plugin_id;

UPDATE plugin_policies
    SET plugin_id = 'vultisig-dca-0000'
    WHERE plugin_type = 'dca';

UPDATE plugin_policies
    SET plugin_id = 'vultisig-payroll-0000'
    WHERE plugin_type = 'payroll';

ALTER TABLE plugin_policies
    ALTER COLUMN plugin_id SET NOT NULL;

ALTER TABLE plugin_policies
    DROP COLUMN plugin_type;

DROP TYPE plugin_type;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TYPE plugin_type AS ENUM (
    'dca',
    'payroll'
);

ALTER TABLE plugin_policies
    ADD COLUMN plugin_type plugin_type;

UPDATE plugin_policies
    SET plugin_type = 'dca'
    WHERE plugin_id = 'vultisig-dca-0000';

UPDATE plugin_policies
    SET plugin_type = 'payroll'
    WHERE plugin_id = 'vultisig-payroll-0000';

ALTER TABLE plugin_policies
    DROP COLUMN plugin_id;

ALTER TABLE plugin_policies
    ADD COLUMN plugin_id TEXT;

UPDATE plugin_policies
    SET plugin_id = 'vultisig-dca-0000'
    WHERE plugin_type = 'dca';

UPDATE plugin_policies
    SET plugin_id = 'vultisig-payroll-0000'
    WHERE plugin_type = 'payroll';

ALTER TABLE plugin_policies
    ALTER COLUMN plugin_id SET NOT NULL;

DROP TYPE plugin_id;
-- +goose StatementEnd
