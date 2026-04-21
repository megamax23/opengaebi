#!/usr/bin/env bash
# SessionStart hook — register this Claude Code session with the opengaebi bridge.
# Called automatically by Claude Code when a session starts.
#
# Required env vars (set in shell profile or .claude/settings.json env):
#   BRIDGE_BASE_URL     bridge server base URL (default: http://localhost:7777)
#   BRIDGE_API_KEY      API key for the bridge
#   BRIDGE_WORKSPACE    workspace name (default: default)
#   BRIDGE_SESSION_NAME unique name for this session (default: hostname+PID)

set -euo pipefail

BASE_URL="${BRIDGE_BASE_URL:-http://localhost:7777}"
API_KEY="${BRIDGE_API_KEY:-}"
WORKSPACE="${BRIDGE_WORKSPACE:-default}"
SESSION_NAME="${BRIDGE_SESSION_NAME:-$(hostname)-$$}"

if [[ -z "$API_KEY" ]]; then
  echo "[opengaebi] BRIDGE_API_KEY not set — skipping session registration" >&2
  exit 0
fi

PAYLOAD=$(printf '{"workspace":"%s","name":"%s","kind":"session","tags":["tool:claude-code"]}' \
  "$WORKSPACE" "$SESSION_NAME")

RESPONSE=$(curl -sf -X POST "${BASE_URL}/v1/agents" \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d "$PAYLOAD" 2>&1) || {
  echo "[opengaebi] bridge unreachable — skipping registration" >&2
  exit 0
}

echo "[opengaebi] session registered: workspace=${WORKSPACE} name=${SESSION_NAME}"
