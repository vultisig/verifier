GET http://localhost:8080/ping
HTTP 200

GET http://localhost:8080/plugins
HTTP 200
[Asserts]
<<<<<<< HEAD
jsonpath "$.data.plugins" exists
jsonpath "$.data.total_count" exists
jsonpath "$.data.plugins" count >= 1
jsonpath "$.data.plugins[0].id" matches "^[a-z]+-[a-z]+-[0-9a-f]{4}$"
jsonpath "$.data.plugins[0].title" exists
jsonpath "$.data.plugins[0].description" exists
jsonpath "$.data.plugins[0].server_endpoint" exists
jsonpath "$.data.plugins[0].pricing" exists
jsonpath "$.data.plugins[0].category_id" exists
jsonpath "$.data.plugins[0].created_at" exists
jsonpath "$.data.plugins[0].updated_at" exists
=======
jsonpath "$.plugins" exists
jsonpath "$.total_count" exists
jsonpath "$.plugins" count >= 1
jsonpath "$.plugins[0].id" matches "^[a-z]+-[a-z]+-[0-9a-f]{4}$"
jsonpath "$.plugins[0].title" exists
jsonpath "$.plugins[0].description" exists
jsonpath "$.plugins[0].server_endpoint" exists
jsonpath "$.plugins[0].pricing" exists
jsonpath "$.plugins[0].category_id" exists
jsonpath "$.plugins[0].created_at" exists
jsonpath "$.plugins[0].updated_at" exists
>>>>>>> a768e80 (plugins yaml)

GET http://localhost:8080/plugins/vultisig-dca-0000
HTTP 200
Content-Type: application/json
[Asserts]
<<<<<<< HEAD
jsonpath "$.data.id" == "vultisig-dca-0000"
jsonpath "$.data.title" exists
jsonpath "$.data.description" exists
jsonpath "$.data.server_endpoint" exists
jsonpath "$.data.pricing" exists
jsonpath "$.data.category_id" == "plugin"
jsonpath "$.data.created_at" exists
jsonpath "$.data.updated_at" exists

GET http://localhost:8080/plugins/vultisig-dca-0000/reviews
HTTP 200
Content-Type: application/json
[Asserts]
jsonpath "$.data.reviews" exists
jsonpath "$.data.reviews" count >= 1
jsonpath "$.data.reviews[0].id" matches "{{uuid_regex}}"
jsonpath "$.data.reviews[0].address" matches "0x[0-9a-fA-F]{40}"
jsonpath "$.data.reviews[0].rating" == 5
jsonpath "$.data.reviews[0].comment" == "Hello world"
jsonpath "$.data.reviews[0].created_at" exists
=======
jsonpath "$.id" == "vultisig-dca-0000"
jsonpath "$.title" exists
jsonpath "$.description" exists
jsonpath "$.server_endpoint" exists
jsonpath "$.pricing" exists
jsonpath "$.category_id" == "plugin"
jsonpath "$.created_at" exists
jsonpath "$.updated_at" exists
>>>>>>> a768e80 (plugins yaml)

GET http://localhost:8080/plugins/vultisig-dca-0000/reviews
HTTP 200
Content-Type: application/json
[Asserts]
jsonpath "$.reviews" exists
jsonpath "$.reviews" count >= 1
jsonpath "$.reviews[0].id" matches "{{uuid_regex}}"
jsonpath "$.reviews[0].address" matches "0x[0-9a-fA-F]{40}"
jsonpath "$.reviews[0].rating" == 5
jsonpath "$.reviews[0].comment" == "Hello world"
jsonpath "$.reviews[0].created_at" exists

GET http://localhost:8080/categories
HTTP 200
Content-Type: application/json
<<<<<<< HEAD
[Asserts]
jsonpath "$.data" count == 2
jsonpath "$.data[0].id" == "ai-agent"
jsonpath "$.data[0].name" == "AI Agent"
jsonpath "$.data[1].id" == "plugin"
jsonpath "$.data[1].name" == "Plugin"

=======
# Go adds a newline so we need to use the raw string body format
```json
[{"id":"ai-agent","name":"AI Agent"},{"id":"plugin","name":"Plugin"}]
```
>>>>>>> a768e80 (plugins yaml)

GET http://localhost:8080/tags
HTTP 200
Content-Type: application/json
[Asserts]
jsonpath "$.data" exists
jsonpath "$.data" count == 2
jsonpath "$.data[0].id" matches "{{uuid_regex}}"
jsonpath "$.data[0].name" == "Trading"
jsonpath "$.data[0].created_at" exists
jsonpath "$.data[1].id" matches "{{uuid_regex}}"
jsonpath "$.data[1].name" == "Operations"
jsonpath "$.data[1].created_at" exists


GET http://localhost:8080/pricing/00000000-0000-0000-0000-000000000002
HTTP 200
Content-Type: application/json
[Asserts]
jsonpath "$.data.id" matches "{{uuid_regex}}"
jsonpath "$.data.type" == "per-tx"
jsonpath "$.data.amount" == 10000
jsonpath "$.data.metric" == "fixed"
jsonpath "$.data.created_at" exists

GET http://localhost:8080/pricing/00000000-0000-0000-0000-000000000003
HTTP 200
Content-Type: application/json
[Asserts]
jsonpath "$.data.id" matches "{{uuid_regex}}"
jsonpath "$.data.type" == "once"
jsonpath "$.data.amount" == 50000
jsonpath "$.data.metric" == "fixed"
jsonpath "$.data.created_at" exists

GET http://localhost:8080/pricing/00000000-0000-0000-0000-000000000004
HTTP 200
Content-Type: application/json
[Asserts]
jsonpath "$.data.id" matches "{{uuid_regex}}"
jsonpath "$.data.type" == "recurring"
jsonpath "$.data.frequency" == "monthly"
jsonpath "$.data.amount" == 30000
jsonpath "$.data.metric" == "fixed"
jsonpath "$.data.created_at" exists

GET http://localhost:8080/vault/get/vultisig-payroll-0000/{{vault_payroll}}
HTTP 200
[Asserts]
jsonpath "$.data.public_key_ecdsa" == {{vault_payroll}}
jsonpath "$.data.public_key_eddsa" == "d5bf32cef8fa68139b5efaaa398ca5449dcd8b4fa610ee8cd7431c09284b4610"
jsonpath "$.data.hex_chain_code" == "c572b1fce3cdb451621f9f8535c95a0272bfcc2b47dda6d136926658be38fe29"

GET http://localhost:8080/vault/exist/vultisig-payroll-0000/{{vault_payroll}}
HTTP 200
[Asserts]
jsonpath "$.data" == "ok"
