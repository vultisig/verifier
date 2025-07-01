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