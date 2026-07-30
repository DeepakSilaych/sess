#!/usr/bin/env bash
# test_sess.sh — Smoke tests for sess v2
# Non-interactive tests only (tmux attach requires a terminal)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SESS="$SCRIPT_DIR/../bin/sess"
export SESS_DIR="/tmp/sess-test-$$"

cleanup() { rm -rf "$SESS_DIR"; }
trap cleanup EXIT

echo "Running sess v2 tests..."

check() {
    local label="$1" needle="$2" out="$3"
    if printf '%s' "$out" | grep -q "$needle"; then
        echo "✓ $label: OK"
    else
        echo "✗ $label: ERROR"
        printf '%s\n' "$out"
        exit 1
    fi
}

# Syntax checks
bash -n "$SESS" && echo "✓ sess: syntax OK" || { echo "✗ sess: syntax ERROR"; exit 1; }
bash -c "source $SCRIPT_DIR/../etc/bash-completion/sess" && echo "✓ bash completion: OK" || { echo "✗ bash completion: ERROR"; exit 1; }

# Version
check "version" "sess" "$($SESS version)"

# Help
check "help" "DETACH" "$($SESS help)"

# Doctor
check "doctor" "tmux" "$($SESS doctor 2>&1)"

# Empty list
check "empty ls" "No sessions" "$($SESS ls)"

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
SESS_CWD=$REPO
SESS_CREATED=$(date -Iseconds)
STATE
echo "$(date -Iseconds)  created (branch: main)" > "$SESS_DIR/sessions/test-session/log"

# List should show our session
check "ls shows session" "test-session" "$($SESS ls)"

# Status
check "status" "test-session" "$($SESS status test-session)"

# Path
check "path" "sess-test-repo" "$($SESS path test-session)"

# Log
check "log" "created" "$($SESS log test-session)"

# Remove
$SESS rm test-session && echo "✓ rm: OK" || { echo "✗ rm: ERROR"; exit 1; }

# Verify removed
check "removed" "No sessions" "$($SESS ls)"

# Cleanup repo
rm -rf "$REPO"

echo ""
echo "All tests passed!"
