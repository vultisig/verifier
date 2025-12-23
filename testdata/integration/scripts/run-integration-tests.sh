#!/bin/bash
set -euo pipefail

# Error trap for better debugging
trap 'echo "❌ Error on line $LINENO: $BASH_COMMAND"; exit 1' ERR

# Integration test runner using Hurl with vault fixture support
# Generates tests from proposed.yaml and runs them in parallel through verifier API
# Requires: auth enabled, vault token seeded in database, JWT bearer token

VERIFIER_URL="${VERIFIER_URL:-http://localhost:8080}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INTEGRATION_DIR="$(dirname "$SCRIPT_DIR")"
REPO_ROOT="$(cd "$INTEGRATION_DIR/../.." && pwd)"
TEMPLATE_FILE="$INTEGRATION_DIR/hurl/plugin-integration.hurl"
FIXTURE_FILE="$INTEGRATION_DIR/fixture.json"
GENERATED_DIR="$INTEGRATION_DIR/hurl/generated"
RESULTS_DIR="$INTEGRATION_DIR/test-results"
BIN_DIR="$INTEGRATION_DIR/bin"
ITUTIL="$BIN_DIR/itutil"

# Parallel jobs (default 4, override with HURL_JOBS env var)
HURL_JOBS="${HURL_JOBS:-4}"

echo "🧪 Integration Tests with Vault Fixture"
echo "========================================"
echo "Verifier URL: $VERIFIER_URL"
echo "Fixture File: $FIXTURE_FILE"
echo "Parallel Jobs: $HURL_JOBS"
echo ""

# Check dependencies
for cmd in yq jq hurl; do
    if ! command -v $cmd &> /dev/null; then
        echo "❌ Error: $cmd is not installed"
        case $cmd in
            yq) echo "   Install with: brew install yq" ;;
            jq) echo "   Install with: brew install jq" ;;
            hurl) echo "   Install with: brew install hurl" ;;
        esac
        exit 1
    fi
done

# Build itutil if needed
if [ ! -f "$ITUTIL" ] || [ "$INTEGRATION_DIR/cmd/itutil/main.go" -nt "$ITUTIL" ]; then
    echo "🔨 Building itutil..."
    mkdir -p "$BIN_DIR"
    (cd "$REPO_ROOT" && go build -o "$ITUTIL" ./testdata/integration/cmd/itutil)
    echo "   ✅ Built $ITUTIL"
    echo ""
fi

# Read fixture metadata
if [ ! -f "$FIXTURE_FILE" ]; then
    echo "❌ Error: Fixture file not found: $FIXTURE_FILE"
    exit 1
fi

# Parse fixture JSON
VAULT_PUBKEY=$(jq -r '.vault.public_key' "$FIXTURE_FILE")
VAULT_NAME=$(jq -r '.vault.name' "$FIXTURE_FILE")
RESHARE_SESSION_ID=$(jq -r '.reshare.session_id' "$FIXTURE_FILE")
HEX_ENCRYPTION_KEY=$(jq -r '.reshare.hex_encryption_key' "$FIXTURE_FILE")
HEX_CHAIN_CODE=$(jq -r '.reshare.hex_chain_code' "$FIXTURE_FILE")
LOCAL_PARTY_ID=$(jq -r '.reshare.local_party_id' "$FIXTURE_FILE")
EMAIL=$(jq -r '.reshare.email' "$FIXTURE_FILE")
# Note: OLD_PARTIES is hardcoded in template since Hurl can't render JSON arrays

# Policy test constants (using test values from smoke tests)
POLICY_ID_CREATE="00000000-0000-0000-0000-000000000001"
POLICY_SIGNATURE="0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
POLICY_RECIPE="CgA="

# Generate JWT token for policy endpoints
JWT_SECRET="mysecret"
JWT_TOKEN=$("$ITUTIL" jwt --secret "$JWT_SECRET" --pubkey "$VAULT_PUBKEY")
if [ $? -ne 0 ] || [ -z "$JWT_TOKEN" ]; then
    echo "❌ Error: Failed to generate JWT token"
    exit 1
fi

# Generate test EVM transaction for plugin-signer tests (base64-encoded)
eval "$("$ITUTIL" evm-fixture --output shell)"
if [ -z "$TX_B64" ] || [ -z "$MSG_B64" ]; then
    echo "❌ Error: Failed to generate test EVM transaction"
    exit 1
fi

echo "📋 Fixture Details:"
echo "   Vault Public Key: $VAULT_PUBKEY"
echo "   Vault Name: $VAULT_NAME"
echo "   Session ID: $RESHARE_SESSION_ID"
echo "   Local Party ID: $LOCAL_PARTY_ID"
echo ""

# Check if verifier is running
echo "🔍 Checking if verifier is running..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$VERIFIER_URL/plugins" || echo "000")
if [ "$HTTP_CODE" != "200" ]; then
    echo "❌ Error: Verifier is not healthy at $VERIFIER_URL (HTTP $HTTP_CODE)"
    echo "   Start with: make run-server-integration"
    exit 1
fi
echo "   ✅ Verifier is healthy"
echo ""

# Create directories
mkdir -p "$GENERATED_DIR"
mkdir -p "$RESULTS_DIR"
rm -f "$GENERATED_DIR"/*.vars
rm -f "$RESULTS_DIR/hurl.log"

# Read plugins from proposed.yaml and generate tests
echo "📝 Generating tests from proposed.yaml..."
PROPOSED_FILE="$REPO_ROOT/proposed.yaml"
if [ ! -f "$PROPOSED_FILE" ]; then
    echo "❌ Error: proposed.yaml not found at $PROPOSED_FILE"
    exit 1
fi
PLUGIN_COUNT=$(yq eval '.plugins | length' "$PROPOSED_FILE")
echo "   Found $PLUGIN_COUNT plugins"
echo ""

GENERATED_COUNT=0

for i in $(seq 0 $((PLUGIN_COUNT - 1))); do
    PLUGIN_ID=$(yq eval ".plugins[$i].id" "$PROPOSED_FILE")

    if [ "$PLUGIN_ID" == "null" ] || [ -z "$PLUGIN_ID" ]; then
        continue
    fi

    PLUGIN_TITLE=$(yq eval ".plugins[$i].title" "$PROPOSED_FILE")

    echo "🔧 Generating: $PLUGIN_TITLE ($PLUGIN_ID)"

    # Determine plugin-specific API key and policy ID
    # API keys follow format: integration-test-apikey-<plugin_id> (from seed_integration_db.go)
    PLUGIN_API_KEY="integration-test-apikey-$PLUGIN_ID"

    case "$PLUGIN_ID" in
        "vultisig-dca-0000")
            POLICY_ID_SIGNER="00000000-0000-0000-0000-000000000011"
            ;;
        "vultisig-recurring-sends-0000")
            POLICY_ID_SIGNER="00000000-0000-0000-0000-000000000012"
            ;;
        *)
            # Generate a generic policy ID for any other plugins
            POLICY_ID_SIGNER="00000000-0000-0000-0000-000000000099"
            ;;
    esac

    # Generate variables file for this plugin (Hurl native approach - no sed escaping issues)
    # Note: OLD_PARTIES is hardcoded in template since Hurl can't render JSON arrays from variables
    # Note: Values must be quoted to prevent Hurl from interpreting hex strings as numbers
    VARS_FILE="$GENERATED_DIR/${PLUGIN_ID}.vars"
    cat > "$VARS_FILE" <<EOF
PLUGIN_ID="$PLUGIN_ID"
VAULT_PUBKEY="$VAULT_PUBKEY"
VAULT_NAME="$VAULT_NAME"
RESHARE_SESSION_ID="$RESHARE_SESSION_ID"
HEX_ENCRYPTION_KEY="$HEX_ENCRYPTION_KEY"
HEX_CHAIN_CODE="$HEX_CHAIN_CODE"
LOCAL_PARTY_ID="$LOCAL_PARTY_ID"
EMAIL="$EMAIL"
POLICY_ID_CREATE="$POLICY_ID_CREATE"
POLICY_SIGNATURE="$POLICY_SIGNATURE"
POLICY_RECIPE="$POLICY_RECIPE"
JWT_TOKEN="$JWT_TOKEN"
PLUGIN_API_KEY="$PLUGIN_API_KEY"
POLICY_ID_SIGNER="$POLICY_ID_SIGNER"
EVM_TX_B64="$TX_B64"
EVM_MSG_B64="$MSG_B64"
EOF

    ((GENERATED_COUNT++))
done

echo ""
echo "📂 Generated $GENERATED_COUNT variable files in $GENERATED_DIR"
echo ""

# Check template exists
if [ ! -f "$TEMPLATE_FILE" ]; then
    echo "❌ Error: Template file not found: $TEMPLATE_FILE"
    exit 1
fi

# Run tests using Hurl's native variables file approach
echo "🚀 Running tests in parallel (jobs: $HURL_JOBS)..."
echo "========================================"
echo ""

set +e
HURL_EXIT=0

# Run hurl for each plugin with its variables file
PASSED_PLUGINS=""
FAILED_PLUGINS=""

for VARS_FILE in "$GENERATED_DIR"/*.vars; do
    PLUGIN_NAME=$(basename "$VARS_FILE" .vars)
    echo "▶ Testing: $PLUGIN_NAME"

    hurl \
        --test \
        --error-format long \
        --variables-file "$VARS_FILE" \
        --report-html "$RESULTS_DIR/html" \
        --report-junit "$RESULTS_DIR/junit.xml" \
        "$TEMPLATE_FILE" 2>&1 | tee -a "$RESULTS_DIR/hurl.log"

    THIS_EXIT=${PIPESTATUS[0]}
    if [ "$THIS_EXIT" -ne 0 ]; then
        HURL_EXIT=$THIS_EXIT
        FAILED_PLUGINS="$FAILED_PLUGINS$PLUGIN_NAME\n"
    else
        PASSED_PLUGINS="$PASSED_PLUGINS$PLUGIN_NAME\n"
    fi
done
set -e

echo ""
echo "========================================"
echo "📊 Detailed Summary"
echo "========================================"

# Show results by plugin name
echo ""
echo "✅ Passed:"
if [ -n "$PASSED_PLUGINS" ]; then
    echo -e "$PASSED_PLUGINS" | grep -v '^$' | while read plugin; do echo "   $plugin"; done
else
    echo "   (none)"
fi

echo ""
echo "❌ Failed:"
if [ -n "$FAILED_PLUGINS" ]; then
    echo -e "$FAILED_PLUGINS" | grep -v '^$' | while read plugin; do echo "   $plugin"; done
else
    echo "   (none)"
fi

echo ""
echo "🐢 Slowest tests:"
grep -E "^(Success|Failure)\s+.* in [0-9]+ ms\)" "$RESULTS_DIR/hurl.log" 2>/dev/null \
    | awk '{
        # Extract duration from "(N request(s) in M ms)"
        for(i=1; i<=NF; i++) {
            if($(i+1) == "ms)") { ms=$i; break }
        }
        print ms
      }' \
    | sort -nr \
    | head -5 \
    | awk '{printf "   %6sms\n",$1}' || echo "   (no timing data)"

echo ""
echo "📁 Reports saved to: $RESULTS_DIR"
echo "   HTML Report: $RESULTS_DIR/html/index.html"
echo "   JUnit XML:   $RESULTS_DIR/junit.xml"
echo "   Raw Log:     $RESULTS_DIR/hurl.log"
echo ""

if [ "$HURL_EXIT" -ne 0 ]; then
    echo "❌ Integration tests failed (exit code: $HURL_EXIT)"
    exit "$HURL_EXIT"
fi

echo "✅ All integration tests passed!"
exit 0
