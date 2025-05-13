-- +goose Up
-- +goose StatementBegin
ALTER TABLE plugin_policies
  DROP COLUMN IF EXISTS chain_code_hex;
ALTER TABLE plugin_policies
  DROP COLUMN IF EXISTS is_ecdsa;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE plugin_policies
  ADD COLUMN IF NOT EXISTS chain_code_hex TEXT NOT NULL;
ALTER TABLE plugin_policies
  ADD COLUMN IF NOT EXISTS is_ecdsa BOOLEAN DEFAULT TRUE;
-- +goose StatementEnd
