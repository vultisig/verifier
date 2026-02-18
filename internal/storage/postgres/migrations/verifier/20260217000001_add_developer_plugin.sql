-- +goose Up
INSERT INTO plugins (id, title, description, server_endpoint, category)
VALUES
('vultisig-developer-0000',
 'Developer (Listing Fee)',
 'Plugin listing fee payment service',
 'https://plugin-developer.lb.services.1conf.com',
 'app')
ON CONFLICT (id) DO NOTHING;

-- +goose Down
