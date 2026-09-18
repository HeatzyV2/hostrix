#!/usr/bin/env bash
# Hostrix update / upgrade script
# Usage (on the Hostrix host):
#   curl -fsSL https://raw.githubusercontent.com/HeatzyV2/hostrix/main/installer/update.sh | sudo bash
# Or locally after install:
#   sudo bash /opt/hostrix/installer/update.sh
#
# Env:
#   HOSTRIX_INSTALL_DIR  default /opt/hostrix
#   HOSTRIX_REPO         default HeatzyV2/hostrix
#   HOSTRIX_VERSION      default main

set -euo pipefail

HOSTRIX_INSTALL_DIR="${HOSTRIX_INSTALL_DIR:-/opt/hostrix}"
HOSTRIX_REPO="${HOSTRIX_REPO:-HeatzyV2/hostrix}"
HOSTRIX_VERSION="${HOSTRIX_VERSION:-main}"

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

log() { echo -e "${CYAN}[hostrix-update]${NC} $*"; }
ok()  { echo -e "${GREEN}[hostrix-update]${NC} $*"; }
die() { echo -e "${RED}[hostrix-update]${NC} $*" >&2; exit 1; }

require_root() {
  [[ "${EUID}" -eq 0 ]] || die "Please run as root (sudo)."
}

go_bin() {
  if [[ -x /usr/local/go/bin/go ]]; then
    echo /usr/local/go/bin/go
  elif command -v go >/dev/null 2>&1; then
    command -v go
  else
    die "Go not found. Re-run the installer or install Go."
  fi
}

fetch_source() {
  [[ -d "${HOSTRIX_INSTALL_DIR}/.git" ]] || die "Hostrix not found at ${HOSTRIX_INSTALL_DIR}"
  log "Updating source to ${HOSTRIX_VERSION}..."
  git -C "${HOSTRIX_INSTALL_DIR}" remote set-url origin "https://github.com/${HOSTRIX_REPO}.git" 2>/dev/null || true
  git -C "${HOSTRIX_INSTALL_DIR}" fetch --depth 1 origin "${HOSTRIX_VERSION}"
  git -C "${HOSTRIX_INSTALL_DIR}" checkout -f FETCH_HEAD
}

ensure_fail2ban() {
  export DEBIAN_FRONTEND=noninteractive
  if ! command -v fail2ban-client >/dev/null 2>&1; then
    log "Installing Fail2Ban..."
    apt-get update -y
    apt-get install -y fail2ban
  fi
  if [[ -f "${HOSTRIX_INSTALL_DIR}/installer/install.sh" ]]; then
    # Refresh jail config by re-running the embedded function body from a small helper file if present
    :
  fi
  mkdir -p /etc/fail2ban/jail.d /etc/fail2ban/filter.d /var/log/hostrix
  cat >/etc/fail2ban/jail.d/hostrix.conf <<'EOF'
[DEFAULT]
bantime  = 1h
findtime = 10m
maxretry = 5
backend  = systemd

[sshd]
enabled = true
port    = ssh
mode    = aggressive

[hostrix-panel]
enabled  = true
port     = 3000
filter   = hostrix-auth
logpath  = /var/log/hostrix/auth-fail.log
maxretry = 8

[hostrix-api]
enabled  = true
port     = 8080
filter   = hostrix-auth
logpath  = /var/log/hostrix/auth-fail.log
maxretry = 10
EOF
  cat >/etc/fail2ban/filter.d/hostrix-auth.conf <<'EOF'
[Definition]
failregex = ^.*hostrix.*(invalid credentials|unauthorized|too many login attempts).*$
            ^.*auth_fail ip=.*$
ignoreregex =
EOF
  touch /var/log/hostrix/auth-fail.log
  chmod 640 /var/log/hostrix/auth-fail.log
  systemctl enable --now fail2ban
  systemctl restart fail2ban || true
  ok "Fail2Ban ensured"
}

build_all() {
  local go
  go="$(go_bin)"
  mkdir -p "${HOSTRIX_INSTALL_DIR}/bin"

  log "Building API..."
  (cd "${HOSTRIX_INSTALL_DIR}/api" && "${go}" build -o "${HOSTRIX_INSTALL_DIR}/bin/hostrix-api" ./cmd/hostrix-api)

  log "Building Agent..."
  (cd "${HOSTRIX_INSTALL_DIR}/agent" && "${go}" build -o "${HOSTRIX_INSTALL_DIR}/bin/hostrix-agent" ./cmd/hostrix-agent)

  log "Building Panel..."
  (cd "${HOSTRIX_INSTALL_DIR}/panel" && { npm ci || npm install; } && npm run build)
}

refresh_units() {
  log "Refreshing systemd units..."
  cp "${HOSTRIX_INSTALL_DIR}/installer/systemd/hostrix-api.service" /etc/systemd/system/
  cp "${HOSTRIX_INSTALL_DIR}/installer/systemd/hostrix-panel.service" /etc/systemd/system/
  cp "${HOSTRIX_INSTALL_DIR}/installer/systemd/hostrix-agent.service" /etc/systemd/system/
  systemctl daemon-reload
}

restart_services() {
  log "Restarting services..."
  systemctl restart hostrix-api
  systemctl restart hostrix-panel
  systemctl try-restart hostrix-agent || true
}

print_summary() {
  local rev
  rev="$(git -C "${HOSTRIX_INSTALL_DIR}" rev-parse --short HEAD 2>/dev/null || echo unknown)"
  echo
  ok "Hostrix updated to ${rev} (${HOSTRIX_VERSION})."
  echo "  Templates on disk were refreshed; API reseeds MariaDB on next start."
  echo "  If the agent was running: systemctl status hostrix-agent"
  echo
}

main() {
  require_root
  fetch_source
  ensure_fail2ban
  build_all
  refresh_units
  restart_services
  print_summary
}

main "$@"
