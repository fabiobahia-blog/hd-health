#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
PREFIX="${PREFIX:-/usr/local/bin}"

echo "Building hd-health..."
(cd "$ROOT" && make build)

echo "Installing binaries to $PREFIX..."
sudo install -m 755 "$ROOT/bin/hd-health" "$PREFIX/hd-health"
sudo install -m 755 "$ROOT/bin/hd-health-agent" "$PREFIX/hd-health-agent"

if [[ -d /etc/systemd/system ]]; then
  echo "Installing systemd unit..."
  sudo install -m 644 "$ROOT/deploy/fedora/hd-health-agent.service" /etc/systemd/system/
  sudo systemctl daemon-reload
  sudo systemctl enable --now hd-health-agent.service
  echo "Agent enabled. Status: systemctl status hd-health-agent"
fi

echo "Done. Try: hd-health scan"
