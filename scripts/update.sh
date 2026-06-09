#!/usr/bin/env bash
set -euo pipefail

OWNER=""
REPO=""
ASSET="screenmate-bot-linux-amd64.tar.gz"

WORKDIR="/tmp/screenmate-bot-update"

rm -rf "$WORKDIR"
mkdir -p "$WORKDIR"
cd "$WORKDIR"

echo "Downloading latest release..."
wget -O "$ASSET" \
  "https://github.com/${OWNER}/${REPO}/releases/latest/download/${ASSET}"

echo "Unpacking..."
tar -xzf "$ASSET"

echo "Deploying..."
chmod +x scripts/deploy.sh
./scripts/deploy.sh

echo "Done."