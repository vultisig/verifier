#!/bin/bash

set -e

if [ -z "$SERVER" ] || [ -z "$USER" ] || [ -z "$DEPLOY_PATH" ]; then
    echo "Error: SERVER, USER, and DEPLOY_PATH environment variables must be set"
    exit 1
fi

echo "Deploying to $USER@$SERVER:$DEPLOY_PATH..."

echo "1. Syncing files to server..."
rsync -avz --delete \
    --exclude='.git' \
    --exclude='.devenv' \
    --exclude='.env' \
    --exclude='*.log' \
    --exclude='dist/' \
    --exclude='node_modules/' \
    --exclude='.github/' \
    ./ $USER@$SERVER:$DEPLOY_PATH/

echo "2. Building and deploying on server..."
ssh $USER@$SERVER << EOF
cd $DEPLOY_PATH

echo "Building Go binaries..."
go build -o verifier cmd/verifier/main.go
go build -o worker cmd/worker/main.go
go build -o txindexer cmd/tx_indexer/main.go

echo "Stopping services before binary replacement..."
sudo systemctl stop verifier || true
sudo systemctl stop txindexer || true
sudo systemctl stop worker || true

echo "Installing binaries to /usr/local/bin/..."
sudo cp verifier /usr/local/bin/
sudo cp txindexer /usr/local/bin/
sudo cp worker /usr/local/bin/
sudo chmod +x /usr/local/bin/verifier /usr/local/bin/txindexer /usr/local/bin/worker

# Verify binaries were installed
if [ ! -f "/usr/local/bin/verifier" ]; then
    echo "ERROR: verifier binary not found in /usr/local/bin/"
    exit 1
fi
if [ ! -f "/usr/local/bin/txindexer" ]; then
    echo "ERROR: tx_indexer binary not found in /usr/local/bin/"
    exit 1
fi
if [ ! -f "/usr/local/bin/worker" ]; then
    echo "ERROR: worker binary not found in /usr/local/bin/"
    exit 1
fi

echo "Creating application directory..."
sudo mkdir -p /var/lib/vultisig
sudo chown $USER:$USER /var/lib/vultisig

echo "Binary installation successful:"
ls -la /usr/local/bin/verifier /usr/local/bin/txindexer /usr/local/bin/worker

echo "Restarting systemd services..."
sudo systemctl restart verifier
sudo systemctl restart txindexer
sudo systemctl restart worker

echo "Checking service status..."
sudo systemctl status verifier --no-pager -l
sudo systemctl status txindexer --no-pager -l
sudo systemctl status worker --no-pager -l

echo "Deployment completed!"
EOF

echo "Deployment finished successfully!"