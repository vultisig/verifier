#!/bin/bash
set -euo pipefail

# Main entry point for integration tests
# Can be run from anywhere in the repository

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$(dirname "$SCRIPT_DIR")/../../" # Back to repository root

echo "🧪 Running Integration Tests"
echo "============================="
echo ""

# Run the actual test script
bash "$SCRIPT_DIR/scripts/run-integration-tests.sh"
