-- +goose Up
-- +goose StatementBegin
-- Add new sends plugin ID to the enum (keeping vultisig-dca-0000 for swaps backward compatibility)
ALTER TYPE plugin_id ADD VALUE IF NOT EXISTS 'vultisig-recurring-sends-0000';
-- +goose StatementEnd

-- +goose Down
