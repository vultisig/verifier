-- +goose Up

CREATE TABLE plugin_reports (
    plugin_id           plugin_id NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    reporter_public_key TEXT NOT NULL,
    reason              TEXT NOT NULL,
    created_at          TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    last_reported_at    TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    report_count        INTEGER DEFAULT 1 NOT NULL,

    PRIMARY KEY (plugin_id, reporter_public_key)
);

CREATE INDEX idx_plugin_reports_window
    ON plugin_reports (plugin_id, last_reported_at DESC);

CREATE TABLE plugin_pause_history (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_id           plugin_id NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    action              TEXT NOT NULL,
    report_count_window INTEGER,
    active_users        INTEGER,
    threshold_rate      NUMERIC(5,4),
    reason              TEXT,
    triggered_by        TEXT,
    created_at          TIMESTAMPTZ DEFAULT NOW() NOT NULL
);

CREATE INDEX idx_plugin_pause_history_plugin
    ON plugin_pause_history (plugin_id, created_at DESC);

-- +goose Down

DROP TABLE IF EXISTS plugin_pause_history;
DROP TABLE IF EXISTS plugin_reports;
