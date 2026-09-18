#!/usr/bin/env bash
set -euo pipefail

HOSTRIX_INSTALL_DIR="${HOSTRIX_INSTALL_DIR:-/opt/hostrix}"
HOSTRIX_REPO="${HOSTRIX_REPO:-HeatzyV2/hostrix}"
HOSTRIX_VERSION="${HOSTRIX_VERSION:-main}"

if [[ "${EUID}" -ne 0 ]]; then
  echo "Please run as root" >&2
  exit 1
fi

cd "${HOSTRIX_INSTALL_DIR}"
git fetch --depth 1 origin "${HOSTRIX_VERSION}"
git checkout -f FETCH_HEAD

cd "${HOSTRIX_INSTALL_DIR}/api"
go build -o "${HOSTRIX_INSTALL_DIR}/bin/hostrix-api" ./cmd/hostrix-api

cd "${HOSTRIX_INSTALL_DIR}/agent"
go build -o "${HOSTRIX_INSTALL_DIR}/bin/hostrix-agent" ./cmd/hostrix-agent

cd "${HOSTRIX_INSTALL_DIR}/panel"
npm ci || npm install
npm run build

systemctl restart hostrix-api
systemctl restart hostrix-panel
systemctl try-restart hostrix-agent
echo "Hostrix upgraded."
