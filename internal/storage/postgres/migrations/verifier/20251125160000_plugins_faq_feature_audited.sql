-- +goose Up
ALTER TABLE plugins
    ADD COLUMN faqs JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN features JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN audited BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE plugins
    DROP COLUMN faqs,
    DROP COLUMN features,
    DROP COLUMN audited;
