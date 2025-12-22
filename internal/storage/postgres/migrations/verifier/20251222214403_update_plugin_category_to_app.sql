-- +goose Up
-- +goose StatementBegin
UPDATE plugins SET category = 'app' WHERE category = 'plugin';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE plugins SET category = 'plugin' WHERE category = 'app';
-- +goose StatementEnd
