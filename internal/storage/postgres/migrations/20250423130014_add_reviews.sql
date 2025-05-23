-- +goose Up
-- +goose StatementBegin
CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    address TEXT NOT NULL,
    rating INT CHECK (rating BETWEEN 1 AND 5),
    comment TEXT NOT NULL CHECK (length(comment) <= 1000),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    plugin_id UUID NOT NULL,
    CONSTRAINT fk_plugin FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE,
    CONSTRAINT uq_review_per_address_plugin UNIQUE (address, plugin_id)
);
CREATE INDEX idx_reviews_plugin_id ON reviews(plugin_id);
CREATE INDEX idx_reviews_address ON reviews(address);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS reviews;
-- +goose StatementEnd