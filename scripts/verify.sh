#!/bin/bash
# Verify mcduck-wallet deployment
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(dirname "$SCRIPT_DIR")"

source "$REPO_DIR/.env"

echo "Checking service status..."
ssh "$SSH_HOST" "sudo systemctl status mcduck-wallet --no-pager"

echo ""
echo "Recent logs:"
ssh "$SSH_HOST" "sudo journalctl -u mcduck-wallet -n 20 --no-pager"

echo ""
echo "Testing HTTP endpoint..."
ssh "$SSH_HOST" "curl -sI http://localhost:80 | head -5"
