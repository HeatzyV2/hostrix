#!/usr/bin/env bash
set -euo pipefail

if [[ "${EUID}" -ne 0 ]]; then
  echo "Please run as root" >&2
  exit 1
fi

systemctl stop hostrix-api 2>/dev/null || true
systemctl stop hostrix-panel 2>/dev/null || true
systemctl stop hostrix-agent 2>/dev/null || true
systemctl disable hostrix-api 2>/dev/null || true
systemctl disable hostrix-panel 2>/dev/null || true
systemctl disable hostrix-agent 2>/dev/null || true
rm -f /etc/systemd/system/hostrix-api.service
rm -f /etc/systemd/system/hostrix-panel.service
rm -f /etc/systemd/system/hostrix-agent.service
systemctl daemon-reload

rm -rf /opt/hostrix
# Keep /etc/hostrix by default (contains secrets). Remove explicitly if desired:
# rm -rf /etc/hostrix

echo "Hostrix uninstalled (MariaDB data left intact)."
