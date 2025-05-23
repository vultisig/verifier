-- +goose Up
-- +goose StatementBegin
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL
);

ALTER TABLE plugins
    ADD COLUMN category_id UUID,
    ADD CONSTRAINT fk_plugins_category FOREIGN KEY (category_id) REFERENCES categories(id);

INSERT INTO categories (name)
    VALUES ('AI Agent'), ('Plugin');

UPDATE plugins
    SET category_id = (SELECT id FROM categories WHERE name = 'Plugin');

ALTER TABLE plugins
    ALTER COLUMN category_id SET NOT NULL;
    
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE plugins
    DROP CONSTRAINT fk_plugins_category,
    DROP COLUMN category_id;

DROP TABLE IF EXISTS categories;
-- +goose StatementEnd
