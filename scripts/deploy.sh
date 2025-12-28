#!/bin/bash
# McDuck Wallet Deployment Script
# Deploys mcduck-wallet to a fresh server with proper service setup
# Idempotent - safe to run multiple times

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(dirname "$SCRIPT_DIR")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${BLUE}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[OK]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

# Load .env
load_env() {
    if [[ -f "$REPO_DIR/.env" ]]; then
        source "$REPO_DIR/.env"
        log "Loaded .env"
    else
        error "No .env found. Copy .env.example and configure it."
    fi
}

# Create user on remote server
setup_user() {
    log "Setting up user $REMOTE_USER on server..."

    ssh "$SSH_HOST" bash << REMOTE
set -e

# Create user if not exists
if ! id "$REMOTE_USER" &>/dev/null; then
    sudo useradd -m -s /bin/bash "$REMOTE_USER"
    echo "User $REMOTE_USER created"
else
    echo "User $REMOTE_USER already exists"
fi

# Setup SSH directory
sudo mkdir -p /home/$REMOTE_USER/.ssh
sudo chmod 700 /home/$REMOTE_USER/.ssh

# Add SSH key (idempotent)
echo "$SSH_PUBLIC_KEY" | sudo tee /home/$REMOTE_USER/.ssh/authorized_keys > /dev/null
sudo chmod 600 /home/$REMOTE_USER/.ssh/authorized_keys
sudo chown -R $REMOTE_USER:$REMOTE_USER /home/$REMOTE_USER/.ssh

echo "SSH key configured"
REMOTE

    success "User setup complete"
}

# Configure firewall
setup_firewall() {
    log "Configuring firewall..."

    ssh "$SSH_HOST" bash << 'REMOTE'
set -e

# Allow port 80 if not already allowed
if ! sudo ufw status | grep -q "80/tcp.*ALLOW"; then
    sudo ufw allow 80/tcp
    echo "Port 80 opened"
else
    echo "Port 80 already open"
fi
REMOTE

    success "Firewall configured"
}

# Create application directory
setup_directories() {
    log "Setting up directories..."

    ssh "$SSH_HOST" bash << REMOTE
set -e
sudo mkdir -p /home/$REMOTE_USER/mcduck-wallet/bin
sudo chown -R $REMOTE_USER:$REMOTE_USER /home/$REMOTE_USER/mcduck-wallet
REMOTE

    success "Directories ready"
}

# Build and transfer binary
deploy_binary() {
    log "Building binary..."

    cd "$REPO_DIR"

    # Generate templates
    templ generate ./...

    # Build for Linux
    mkdir -p bin
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC=x86_64-linux-musl-cc \
        go build -ldflags '-linkmode external -extldflags "-static" -s -w' \
        -o bin/mcduck-wallet cmd/mcduck-wallet/main.go

    success "Binary built"

    log "Transferring binary..."
    scp bin/mcduck-wallet "${REMOTE_USER}@${SERVER}:/home/$REMOTE_USER/mcduck-wallet/bin/mcduck-wallet.new"

    success "Binary transferred"
}

# Setup systemd service
setup_service() {
    log "Setting up systemd service..."

    ssh "$SSH_HOST" bash << REMOTE
set -e

# Create systemd service file
sudo tee /etc/systemd/system/mcduck-wallet.service > /dev/null << EOF
[Unit]
Description=McDuck Wallet Telegram Bot
After=network.target

[Service]
Type=simple
User=$REMOTE_USER
Group=$REMOTE_USER
WorkingDirectory=/home/$REMOTE_USER/mcduck-wallet
ExecStart=/home/$REMOTE_USER/mcduck-wallet/bin/mcduck-wallet
Restart=always
RestartSec=5
Environment=TELEGRAM_BOT_TOKEN=$TELEGRAM_BOT_TOKEN

# Allow binding to port 80
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE

# Security hardening
ProtectSystem=strict
ProtectHome=read-only
ReadWritePaths=/home/$REMOTE_USER/mcduck-wallet
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable mcduck-wallet

echo "Systemd service configured"
REMOTE

    success "Service setup complete"
}

# Activate new binary and restart
activate_and_restart() {
    log "Activating new binary and restarting service..."

    ssh "${REMOTE_USER}@${SERVER}" bash << 'REMOTE'
set -e
cd ~/mcduck-wallet

# Swap binaries
if [[ -f bin/mcduck-wallet.new ]]; then
    mv bin/mcduck-wallet.new bin/mcduck-wallet
    chmod +x bin/mcduck-wallet
fi
REMOTE

    # Restart via sudo user
    ssh "$SSH_HOST" "sudo systemctl restart mcduck-wallet"

    sleep 3

    # Check status
    if ssh "$SSH_HOST" "systemctl is-active --quiet mcduck-wallet"; then
        success "Service running"
    else
        warn "Service may not be running, check with: ssh $SSH_HOST 'sudo journalctl -u mcduck-wallet -n 50'"
    fi
}

# Verify deployment
verify() {
    log "Verifying deployment..."

    local status
    status=$(ssh "$SSH_HOST" "systemctl is-active mcduck-wallet || true")

    if [[ "$status" == "active" ]]; then
        success "mcduck-wallet is running"
        ssh "$SSH_HOST" "sudo systemctl status mcduck-wallet --no-pager | head -15"
    else
        error "Service not running. Status: $status"
    fi
}

# Main deployment
main() {
    log "McDuck Wallet Deployment"
    echo "═══════════════════════════════════════════════════════"

    cd "$REPO_DIR"
    load_env

    # Check SSH connectivity
    log "Testing SSH connection to $SSH_HOST..."
    ssh -o ConnectTimeout=10 "$SSH_HOST" "echo 'SSH OK'" || error "Cannot connect to $SSH_HOST"
    success "SSH connection OK"

    setup_user
    setup_firewall
    setup_directories
    deploy_binary
    setup_service
    activate_and_restart
    verify

    echo ""
    echo "═══════════════════════════════════════════════════════"
    success "Deployment complete!"
}

main "$@"
