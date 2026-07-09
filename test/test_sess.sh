#!/usr/bin/env bash
# test_sess.sh — Smoke tests for sess v2
# Non-interactive tests only (dtach attach requires a terminal)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SESS="$SCRIPT_DIR/../bin/sess"
export SESS_DIR="/tmp/sess-test-$$"

cleanup() { rm -rf "$SESS_DIR"; }
trap cleanup EXIT

echo "Running sess v2 tests..."

# Syntax checks
bash -n "$SESS" && echo "✓ sess: syntax OK" || { echo "✗ sess: syntax ERROR"; exit 1; }
bash -c "source $SCRIPT_DIR/../etc/bash-completion/sess" && echo "✓ bash completion: OK" || { echo "✗ bash completion: ERROR"; exit 1; }

# Version
$SESS version | grep -q "sess" && echo "✓ version: OK" || { echo "✗ version: ERROR"; exit 1; }

# Help
$SESS help | grep -q "overlay" && echo "✓ help (overlay): OK" || { echo "✗ help: ERROR"; exit 1; }

# Doctor
$SESS doctor 2>&1 | grep -q "dtach" && echo "✓ doctor: OK" || { echo "✗ doctor: ERROR"; $SESS doctor 2>&1; exit 1; }

# Empty list
$SESS ls | grep -q "No sessions" && echo "✓ empty ls: OK" || { echo "✗ empty ls: ERROR"; exit 1; }

# Create a temp git repo
REPO="/tmp/sess-test-repo-$$"
mkdir -p "$REPO" && cd "$REPO"
git init && git config user.email "test@test.com" && git config user.name "Test"
echo "hello" > README.md && git add . && git commit -m "initial"

# Create a session (non-interactive — can't auto-attach, but can create state)
# We'll create the session directory manually to test non-attach commands
mkdir -p "$SESS_DIR/sessions/test-session"
cat > "$SESS_DIR/sessions/test-session/state" <<STATE
SESS_SESSION=test-session
SESS_BRANCH=main
SESS_OVERLAY=none
SESS_CWD=$REPO
SESS_CREATED=$(date -Iseconds)
STATE
mkdir -p "$SESS_DIR/sessions/test-session/overlays"
echo "$(date -Iseconds)  created (branch: main)" > "$SESS_DIR/sessions/test-session/log"

# List should show our session
$SESS ls | grep -q "test-session" && echo "✓ ls shows session: OK" || { echo "✗ ls: ERROR"; exit 1; }

# Status
$SESS status test-session | grep -q "test-session" && echo "✓ status: OK" || { echo "✗ status: ERROR"; exit 1; }

# Path (no overlay)
$SESS path test-session | grep -q "sess-test-repo" && echo "✓ path: OK" || { echo "✗ path: ERROR"; exit 1; }

# Log
$SESS log test-session | grep -q "created" && echo "✓ log: OK" || { echo "✗ log: ERROR"; exit 1; }

# Remove
$SESS rm test-session && echo "✓ rm: OK" || { echo "✗ rm: ERROR"; exit 1; }

# Verify removed
$SESS ls | grep -q "No sessions" && echo "✓ removed: OK" || { echo "✗ removed: ERROR"; exit 1; }

# Cleanup repo
rm -rf "$REPO"

echo ""
echo "All tests passed!"