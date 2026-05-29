#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
PREFIX="${PREFIX:-/usr/local/bin}"
LAUNCHD_DIR="/Library/LaunchDaemons"
PLIST="com.hdhealth.agent.plist"

echo "Building hd-health..."
(cd "$ROOT" && make build)

echo "Installing binaries to $PREFIX..."
install -m 755 "$ROOT/bin/hd-health" "$PREFIX/hd-health"
install -m 755 "$ROOT/bin/hd-health-agent" "$PREFIX/hd-health-agent"

if [[ "$(uname)" == "Darwin" ]]; then
  echo "Installing LaunchDaemon..."
  sudo install -m 644 "$ROOT/deploy/macos/$PLIST" "$LAUNCHD_DIR/$PLIST"
  sudo launchctl bootout system "$LAUNCHD_DIR/$PLIST" 2>/dev/null || true
  sudo launchctl bootstrap system "$LAUNCHD_DIR/$PLIST"
  echo "Agent installed. Logs: /var/log/hd-health-agent.log"
fi

echo "Done. Try: hd-health scan"
