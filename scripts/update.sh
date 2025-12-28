#!/bin/bash
# Update mcduck-wallet to latest version
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(dirname "$SCRIPT_DIR")"

source "$REPO_DIR/.env"

echo "Building and deploying update..."

cd "$REPO_DIR"

# Generate templates
templ generate ./...

# Build for Linux
mkdir -p bin
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC=x86_64-linux-musl-cc \
    go build -ldflags '-linkmode external -extldflags "-static" -s -w' \
    -o bin/mcduck-wallet cmd/mcduck-wallet/main.go

echo "Transferring binary..."
scp bin/mcduck-wallet "${REMOTE_USER}@${SERVER}:~/mcduck-wallet/bin/mcduck-wallet.new"

echo "Swapping binary and restarting..."
ssh "${REMOTE_USER}@${SERVER}" "cd ~/mcduck-wallet && mv bin/mcduck-wallet.new bin/mcduck-wallet && chmod +x bin/mcduck-wallet"
ssh "$SSH_HOST" "sudo systemctl restart mcduck-wallet"

sleep 2

echo "Verifying..."
ssh "$SSH_HOST" "systemctl is-active mcduck-wallet && echo 'Service running'"

echo "Update complete!"
