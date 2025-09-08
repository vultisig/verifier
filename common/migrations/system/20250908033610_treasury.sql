-- +goose Up
-- +goose StatementBegin

CREATE TABLE plugin_developer (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    public_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)

ALTER TABLE plugins ADD COLUMN developer_id UUID REFERENCES plugin_developer(id) ON DELETE CASCADE;


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
