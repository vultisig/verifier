GET http://localhost:8080/ping
HTTP 200

GET http://localhost:8080/plugins
HTTP 200
[Asserts]
jsonpath "$.plugins" exists
jsonpath "$.total_count" exists
jsonpath "$.plugins" count >= 1
jsonpath "$.plugins[0].id" matches "^[a-z]+-[a-z]+-[0-9a-f]{4}$"
jsonpath "$.plugins[0].title" exists
jsonpath "$.plugins[0].description" exists
jsonpath "$.plugins[0].server_endpoint" exists
jsonpath "$.plugins[0].pricing_id" exists
jsonpath "$.plugins[0].category_id" exists
jsonpath "$.plugins[0].created_at" exists
jsonpath "$.plugins[0].updated_at" exists

GET http://localhost:8080/plugins/vultisig-dca-0000
HTTP 200
Content-Type: application/json
[Asserts]
jsonpath "$.id" matches "vultisig-dca-0000"
jsonpath "$.title" exists
jsonpath "$.description" exists
jsonpath "$.server_endpoint" exists
jsonpath "$.pricing_id" exists
jsonpath "$.category_id" == "plugin"
jsonpath "$.created_at" exists
jsonpath "$.updated_at" exists

GET http://localhost:8080/categories
HTTP 200
Content-Type: application/json
# Go adds a newline so we need to use the raw string body format
```json
[{"id":"ai-agent","name":"AI Agent"},{"id":"plugin","name":"Plugin"}]
```

GET http://localhost:8080/tags
HTTP 200
Content-Type: application/json
[Asserts]
jsonpath "$" exists
jsonpath "$" count >= 2
jsonpath "$[0].id" matches "00000000-0000-0000-0000-000000000001"
jsonpath "$[0].name" == "Trading"
jsonpath "$[0].created_at" exists
jsonpath "$[1].id" matches "00000000-0000-0000-0000-000000000002"
jsonpath "$[1].name" == "Operations"
jsonpath "$[1].created_at" exists

GET http://localhost:8080/pricing/00000000-0000-0000-0000-000000000001
HTTP 200
Content-Type: application/json
[Asserts]
jsonpath "$.id" matches "00000000-0000-0000-0000-000000000001"
jsonpath "$.type" == "free"
jsonpath "$.amount" == 0.00
jsonpath "$.metric" == "fixed"
jsonpath "$.created_at" exists

GET http://localhost:8080/pricing/00000000-0000-0000-0000-000000000002
HTTP 200
Content-Type: application/json
[Asserts]
jsonpath "$.id" matches "00000000-0000-0000-0000-000000000002"
jsonpath "$.type" == "per-tx"
jsonpath "$.amount" == 100.00
jsonpath "$.metric" == "fixed"
jsonpath "$.created_at" exists