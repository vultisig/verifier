# plugin_smoke.hurl
# Smoke tests for plugin endpoints that verifier calls via ServerEndpoint.
#
# Expected vars (from env files):
#   plugin_id               - e.g. vultisig-dca-0000
#   plugin_server_endpoint  - e.g. https://plugin-dca-swap.lb.services.1conf.com


GET {{plugin_server_endpoint}}/plugin/recipe-specification

HTTP 200
Content-Type: application/json
[Asserts]
jsonpath "$.plugin_id" == {{plugin_id}}
jsonpath "$.plugin_name" == {{plugin_title}}
jsonpath "$.configuration.definitions" exists
jsonpath "$.configuration.properties" exists
jsonpath "$.configuration_example" exists
jsonpath "$.supported_resources" exists
