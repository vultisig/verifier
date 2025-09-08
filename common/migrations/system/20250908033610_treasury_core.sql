-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS plugin_developer (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    public_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE plugin_policies ADD COLUMN developer_id UUID REFERENCES plugin_developer(id) ON DELETE CASCADE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE plugins DROP COLUMN IF EXISTS developer_id;
DROP TABLE IF EXISTS plugin_developer;

-- +goose StatementEnd
