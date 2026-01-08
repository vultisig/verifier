#!/bin/bash
set -euo pipefail

# Error trap for better debugging
trap 'echo "❌ Error on line $LINENO: $BASH_COMMAND"; exit 1' ERR

# Integration test runner using Hurl with parallel execution
# Generates per-plugin wrapper .hurl files that include the main template
# Runs single hurl invocation with --jobs for parallelism
# Produces one combined HTML report + one JUnit XML

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
            yq) echo "   Install: brew install yq (macOS) or sudo snap install yq (Linux)" ;;
            jq) echo "   Install: brew install jq (macOS) or sudo apt install jq (Linux)" ;;
            hurl) echo "   Install: brew install hurl (macOS) or see https://hurl.dev" ;;
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

# Validate parsed values
MISSING_FIELDS=""
[ -z "$VAULT_PUBKEY" ] || [ "$VAULT_PUBKEY" = "null" ] && MISSING_FIELDS="$MISSING_FIELDS VAULT_PUBKEY"
[ -z "$VAULT_NAME" ] || [ "$VAULT_NAME" = "null" ] && MISSING_FIELDS="$MISSING_FIELDS VAULT_NAME"
[ -z "$RESHARE_SESSION_ID" ] || [ "$RESHARE_SESSION_ID" = "null" ] && MISSING_FIELDS="$MISSING_FIELDS RESHARE_SESSION_ID"
[ -z "$HEX_ENCRYPTION_KEY" ] || [ "$HEX_ENCRYPTION_KEY" = "null" ] && MISSING_FIELDS="$MISSING_FIELDS HEX_ENCRYPTION_KEY"
[ -z "$HEX_CHAIN_CODE" ] || [ "$HEX_CHAIN_CODE" = "null" ] && MISSING_FIELDS="$MISSING_FIELDS HEX_CHAIN_CODE"
[ -z "$LOCAL_PARTY_ID" ] || [ "$LOCAL_PARTY_ID" = "null" ] && MISSING_FIELDS="$MISSING_FIELDS LOCAL_PARTY_ID"
[ -z "$EMAIL" ] || [ "$EMAIL" = "null" ] && MISSING_FIELDS="$MISSING_FIELDS EMAIL"
if [ -n "$MISSING_FIELDS" ]; then
    echo "❌ Error: Failed to parse required fields from fixture file:$MISSING_FIELDS"
    exit 1
fi

# Policy test constants
POLICY_ID_CREATE="00000000-0000-0000-0000-000000000001"
POLICY_SIGNATURE="0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
POLICY_RECIPE="CgA="

# Generate JWT token for policy endpoints
JWT_SECRET="mysecret"
JWT_TOKEN=$("$ITUTIL" jwt --secret "$JWT_SECRET" --pubkey "$VAULT_PUBKEY")
if [ -z "$JWT_TOKEN" ]; then
    echo "❌ Error: Failed to generate JWT token"
    exit 1
fi

# Generate test EVM transaction for plugin-signer tests (using JSON output + jq)
EVM_FIXTURE_JSON=$("$ITUTIL" evm-fixture --output json)
TX_B64=$(echo "$EVM_FIXTURE_JSON" | jq -r '.tx_b64')
MSG_B64=$(echo "$EVM_FIXTURE_JSON" | jq -r '.msg_b64')
MSG_SHA256_B64=$(echo "$EVM_FIXTURE_JSON" | jq -r '.msg_sha256_b64')
if [ -z "$TX_B64" ] || [ -z "$MSG_B64" ] || [ -z "$MSG_SHA256_B64" ] || \
   [ "$TX_B64" = "null" ] || [ "$MSG_B64" = "null" ] || [ "$MSG_SHA256_B64" = "null" ]; then
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

# Create directories and clean up
mkdir -p "$GENERATED_DIR"
mkdir -p "$RESULTS_DIR"
rm -f "$GENERATED_DIR"/*.hurl
rm -f "$RESULTS_DIR/junit.xml"
rm -rf "$RESULTS_DIR/html"

# Check template exists
if [ ! -f "$TEMPLATE_FILE" ]; then
    echo "❌ Error: Template file not found: $TEMPLATE_FILE"
    exit 1
fi

# Read plugins from proposed.yaml and generate per-plugin wrapper .hurl files
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

    # Plugin-specific values
    PLUGIN_API_KEY="integration-test-apikey-$PLUGIN_ID"

    # Generate policy ID dynamically based on plugin index (matches seed-db logic: i+11)
    POLICY_INDEX=$((i + 11))
    POLICY_ID_SIGNER=$(printf "00000000-0000-0000-0000-0000000000%02d" "$POLICY_INDEX")

    # Generate wrapper .hurl file with [Options] variables, then include template
    OUTPUT_FILE="$GENERATED_DIR/${PLUGIN_ID}.hurl"
    cat > "$OUTPUT_FILE" <<EOF
# Auto-generated wrapper for $PLUGIN_ID
# Sets variables via [Options], then includes the main test template

# Dummy request to set variables for all subsequent requests
# This GET will succeed and set variables for the included template
GET ${VERIFIER_URL}/plugins/${PLUGIN_ID}
[Options]
variable: VERIFIER_URL="${VERIFIER_URL}"
variable: PLUGIN_ID="${PLUGIN_ID}"
variable: VAULT_PUBKEY="${VAULT_PUBKEY}"
variable: VAULT_NAME="${VAULT_NAME}"
variable: RESHARE_SESSION_ID="${RESHARE_SESSION_ID}"
variable: HEX_ENCRYPTION_KEY="${HEX_ENCRYPTION_KEY}"
variable: HEX_CHAIN_CODE="${HEX_CHAIN_CODE}"
variable: LOCAL_PARTY_ID="${LOCAL_PARTY_ID}"
variable: EMAIL="${EMAIL}"
variable: POLICY_ID_CREATE="${POLICY_ID_CREATE}"
variable: POLICY_SIGNATURE="${POLICY_SIGNATURE}"
variable: POLICY_RECIPE="${POLICY_RECIPE}"
variable: JWT_TOKEN="${JWT_TOKEN}"
variable: PLUGIN_API_KEY="${PLUGIN_API_KEY}"
variable: POLICY_ID_SIGNER="${POLICY_ID_SIGNER}"
variable: EVM_TX_B64="${TX_B64}"
variable: EVM_MSG_B64="${MSG_B64}"
variable: EVM_MSG_SHA256_B64="${MSG_SHA256_B64}"
HTTP 200
[Asserts]
jsonpath "\$.data.id" == "${PLUGIN_ID}"

# Include the main test template (all tests after this use the variables above)
EOF

    # Append the template content (skip the first test since we already did it as the variable setter)
    # Extract everything after "# Test 2:" to avoid duplicating Test 1
    sed -n '/^# Test 2:/,$p' "$TEMPLATE_FILE" >> "$OUTPUT_FILE"

    GENERATED_COUNT=$((GENERATED_COUNT + 1))
done

echo ""
echo "📂 Generated $GENERATED_COUNT wrapper files in $GENERATED_DIR"
echo ""

# Run ALL tests in a single hurl invocation with parallelism
echo "🚀 Running tests in parallel (jobs: $HURL_JOBS)..."
echo "========================================"
echo ""

# Disable ERR trap during hurl execution
trap - ERR

hurl \
    --test \
    --jobs "$HURL_JOBS" \
    --error-format long \
    --report-html "$RESULTS_DIR/html" \
    --report-junit "$RESULTS_DIR/junit.xml" \
    "$GENERATED_DIR"/*.hurl 2>&1 | tee "$RESULTS_DIR/hurl.log"

HURL_EXIT=${PIPESTATUS[0]}

# Re-enable ERR trap
trap 'echo "❌ Error on line $LINENO: $BASH_COMMAND"; exit 1' ERR

echo ""
echo "========================================"
echo "📊 Test Summary (from JUnit XML)"
echo "========================================"

# Parse JUnit XML for stable summary
if [ -f "$RESULTS_DIR/junit.xml" ]; then
    # Extract counts from JUnit XML using sed (portable across Linux/macOS)
    TOTAL_TESTS=$(sed -n 's/.*tests="\([0-9]*\)".*/\1/p' "$RESULTS_DIR/junit.xml" | head -1)
    TOTAL_FAILURES=$(sed -n 's/.*failures="\([0-9]*\)".*/\1/p' "$RESULTS_DIR/junit.xml" | head -1)
    TOTAL_ERRORS=$(sed -n 's/.*errors="\([0-9]*\)".*/\1/p' "$RESULTS_DIR/junit.xml" | head -1)
    TOTAL_TIME=$(sed -n 's/.*<testsuite.*time="\([0-9.]*\)".*/\1/p' "$RESULTS_DIR/junit.xml" | head -1)

    TOTAL_TESTS=${TOTAL_TESTS:-0}
    TOTAL_FAILURES=${TOTAL_FAILURES:-0}
    TOTAL_ERRORS=${TOTAL_ERRORS:-0}
    TOTAL_TIME=${TOTAL_TIME:-0}

    PASSED_COUNT=$((TOTAL_TESTS - TOTAL_FAILURES - TOTAL_ERRORS))
    FAILED_COUNT=$((TOTAL_FAILURES + TOTAL_ERRORS))

    echo ""
    echo "📊 Results: $PASSED_COUNT passed, $FAILED_COUNT failed (${TOTAL_TIME}s)"
    echo ""

    # List passed/failed test files
    # JUnit XML may have all testcases on one line, so split first
    # Self-closing <testcase ... /> means passed, <testcase>...<failure>...</testcase> means failed

    echo "✅ Passed:"
    # Split testcases onto separate lines, find self-closing ones, extract name
    PASSED_TESTS=$(sed 's/<testcase/\n<testcase/g' "$RESULTS_DIR/junit.xml" | \
        grep '<testcase.*\/>' | \
        sed -n 's/.*name="\([^"]*\)".*/\1/p' | \
        sed 's|.*/||; s|\.hurl$||')
    if [ -n "$PASSED_TESTS" ]; then
        echo "$PASSED_TESTS" | while read -r testcase; do
            echo "   $testcase"
        done
    else
        echo "   (none)"
    fi

    echo ""
    echo "❌ Failed:"
    # Check if any failures exist
    if grep -q '<failure' "$RESULTS_DIR/junit.xml" 2>/dev/null; then
        # Find testcases with failure children (non-self-closing)
        sed 's/<testcase/\n<testcase/g' "$RESULTS_DIR/junit.xml" | \
            grep -v '\/>' | grep '<testcase' | \
            sed -n 's/.*name="\([^"]*\)".*/\1/p' | \
            sed 's|.*/||; s|\.hurl$||' | while read -r testcase; do
            echo "   $testcase"
        done
    else
        echo "   (none)"
    fi

    # Show slowest tests - extract name and time, sort by time
    echo ""
    echo "🐢 Slowest tests:"
    # Split testcases, extract time and name
    sed 's/<testcase/\n<testcase/g' "$RESULTS_DIR/junit.xml" | \
        grep '<testcase' | \
        sed -n 's/.*time="\([0-9.]*\)".*name="\([^"]*\)".*/\1 \2/p; s/.*name="\([^"]*\)".*time="\([0-9.]*\)".*/\2 \1/p' | \
        sed 's| .*/| |; s|\.hurl$||' | \
        sort -rn | head -5 | \
        awk '{printf "   %6.2fs  %s\n", $1, $2}' 2>/dev/null || echo "   (no timing data)"
else
    echo "⚠️  JUnit XML not found, falling back to log parsing"

    # Fallback to hurl output parsing
    PASSED_COUNT=$(grep -cE "^Success\s+" "$RESULTS_DIR/hurl.log" 2>/dev/null || true)
    FAILED_COUNT=$(grep -cE "^Failure\s+" "$RESULTS_DIR/hurl.log" 2>/dev/null || true)
    PASSED_COUNT=${PASSED_COUNT:-0}
    FAILED_COUNT=${FAILED_COUNT:-0}

    echo ""
    echo "📊 Results: $PASSED_COUNT passed, $FAILED_COUNT failed"
fi

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
