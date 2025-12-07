#!/usr/bin/env bash
set -euo pipefail

# Run from repo root even if script is called from inside testdata/smoke
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

echo "Generating plugin env files from proposed.yaml"
echo "Repo root: $(pwd)"

if ! command -v yq >/dev/null 2>&1; then
  echo "ERROR: yq not found in PATH"
  exit 1
fi

if [ ! -f "proposed.yaml" ]; then
  echo "ERROR: proposed.yaml not found in repo root"
  exit 1
fi

# All plugin IDs from proposed.yaml (yq v4)
PLUGIN_IDS=$(yq -r '.plugins[].id' proposed.yaml)

if [ -z "$PLUGIN_IDS" ]; then
  echo "No plugins found in proposed.yaml"
  exit 1
fi

mkdir -p testdata/smoke

for PLUGIN_ID in $PLUGIN_IDS; do
  SERVER_ENDPOINT=$(yq -r ".plugins[] | select(.id == \"$PLUGIN_ID\") | .server_endpoint" proposed.yaml)
  PLUGIN_TITLE=$(yq -r ".plugins[] | select(.id == \"$PLUGIN_ID\") | .title" proposed.yaml)

  ENV_FILE="testdata/smoke/${PLUGIN_ID}.env"

  {
    echo "plugin_id=$PLUGIN_ID"
    # title is handy for logs / future tests
    [ -n "${PLUGIN_TITLE:-}" ] && [ "$PLUGIN_TITLE" != "null" ] && \
      echo "plugin_title=$PLUGIN_TITLE"
    [ -n "${SERVER_ENDPOINT:-}" ] && [ "$SERVER_ENDPOINT" != "null" ] && \
      echo "plugin_server_endpoint=$SERVER_ENDPOINT"
  } > "$ENV_FILE"

  echo "  -> wrote $ENV_FILE"
done

echo "Done. Env files ready under testdata/smoke/*.env"
