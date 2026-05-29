#!/bin/bash
# Jamf Extension Attribute: HD Health storage report (JSON one-liner)
# Install hd-health to /usr/local/bin first.

PATH="/usr/local/bin:/opt/homebrew/bin:$PATH"

if ! command -v hd-health >/dev/null 2>&1; then
  echo "<result>hd-health not installed</result>"
  exit 0
fi

OUT=$(hd-health export --format json 2>/dev/null) || OUT='{"error":"report failed"}'
# Jamf expects XML result; escape minimal
ESC=$(echo "$OUT" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g')
echo "<result>${ESC}</result>"
