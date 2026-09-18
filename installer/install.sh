#!/usr/bin/env bash
# Hostrix installer — Phase 1 skeleton (Linux only)
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/OWNER/hostrix/main/installer/install.sh | bash
# Non-interactive:
#   HOSTRIX_NONINTERACTIVE=1 curl -fsSL ... | bash

set -euo pipefail

HOSTRIX_VERSION="${HOSTRIX_VERSION:-main}"
HOSTRIX_REPO="${HOSTRIX_REPO:-HeatzyV2/hostrix}"
HOSTRIX_INSTALL_DIR="${HOSTRIX_INSTALL_DIR:-/opt/hostrix}"
HOSTRIX_NONINTERACTIVE="${HOSTRIX_NONINTERACTIVE:-0}"
HOSTRIX_DB_NAME="${HOSTRIX_DB_NAME:-hostrix}"
HOSTRIX_DB_USER="${HOSTRIX_DB_USER:-hostrix}"
HOSTRIX_HTTP_ADDR="${HOSTRIX_HTTP_ADDR:-:8080}"
HOSTRIX_PANEL_ORIGIN="${HOSTRIX_PANEL_ORIGIN:-http://localhost:3000}"

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

log()  { echo -e "${CYAN}[hostrix]${NC} $*"; }
ok()   { echo -e "${GREEN}[hostrix]${NC} $*"; }
die()  { echo -e "${RED}[hostrix]${NC} $*" >&2; exit 1; }

require_root() {
  if [[ "${EUID}" -ne 0 ]]; then
    die "Please run as root (sudo)."
  fi
}

detect_os() {
  if [[ ! -f /etc/os-release ]]; then
    die "Unsupported OS: /etc/os-release missing"
  fi
  # shellcheck disable=SC1091
  source /etc/os-release
  OS_ID="${ID:-unknown}"
  OS_VERSION="${VERSION_ID:-unknown}"
  case "${OS_ID}" in
    ubuntu|debian|linuxmint) ;;
    *) die "Unsupported distribution: ${OS_ID}. Supported: Ubuntu/Debian." ;;
  esac
  log "Detected ${PRETTY_NAME:-$OS_ID $OS_VERSION}"
}

detect_arch() {
  ARCH="$(uname -m)"
  case "${ARCH}" in
    x86_64|amd64) ARCH=amd64 ;;
    aarch64|arm64) ARCH=arm64 ;;
    *) die "Unsupported CPU architecture: ${ARCH}" ;;
  esac
  log "Architecture: ${ARCH}"
}

prompt() {
  local var="$1" msg="$2" default="${3:-}"
  if [[ "${HOSTRIX_NONINTERACTIVE}" == "1" ]]; then
    printf -v "${var}" '%s' "${default}"
    return
  fi
  local input
  if [[ -n "${default}" ]]; then
    read -r -p "${msg} [${default}]: " input || true
    printf -v "${var}" '%s' "${input:-$default}"
  else
    read -r -p "${msg}: " input
    printf -v "${var}" '%s' "${input}"
  fi
}

install_packages() {
  log "Installing system packages..."
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -y
  apt-get install -y curl ca-certificates gnupg lsb-release git build-essential \
    mariadb-server mariadb-client

  if ! command -v go >/dev/null 2>&1; then
    log "Installing Go toolchain..."
    local go_ver="1.24.2"
    curl -fsSL "https://go.dev/dl/go${go_ver}.linux-${ARCH}.tar.gz" -o /tmp/go.tgz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf /tmp/go.tgz
    ln -sfn /usr/local/go/bin/go /usr/local/bin/go
  fi

  if ! command -v node >/dev/null 2>&1; then
    log "Installing Node.js 22.x..."
    curl -fsSL https://deb.nodesource.com/setup_22.x | bash -
    apt-get install -y nodejs
  fi
}

setup_mariadb() {
  log "Configuring MariaDB..."
  systemctl enable --now mariadb

  local db_pass
  if [[ -n "${HOSTRIX_DB_PASSWORD:-}" ]]; then
    db_pass="${HOSTRIX_DB_PASSWORD}"
  else
    db_pass="$(openssl rand -hex 16)"
  fi

  if [[ -n "${HOSTRIX_BOOTSTRAP_ADMIN_PASSWORD:-}" ]]; then
    HOSTRIX_ADMIN_PASSWORD="${HOSTRIX_BOOTSTRAP_ADMIN_PASSWORD}"
  else
    prompt HOSTRIX_ADMIN_PASSWORD "Admin panel password" "changeme"
  fi
  if [[ -z "${HOSTRIX_ADMIN_PASSWORD:-}" ]]; then
    HOSTRIX_ADMIN_PASSWORD="changeme"
  fi
  prompt HOSTRIX_DB_PASSWORD_INPUT "MariaDB password for user ${HOSTRIX_DB_USER}" "${db_pass}"
  HOSTRIX_DB_PASSWORD="${HOSTRIX_DB_PASSWORD_INPUT}"

  mysql -e "CREATE DATABASE IF NOT EXISTS \`${HOSTRIX_DB_NAME}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
  mysql -e "CREATE USER IF NOT EXISTS '${HOSTRIX_DB_USER}'@'localhost' IDENTIFIED BY '${HOSTRIX_DB_PASSWORD}';"
  mysql -e "ALTER USER '${HOSTRIX_DB_USER}'@'localhost' IDENTIFIED BY '${HOSTRIX_DB_PASSWORD}';"
  mysql -e "GRANT ALL PRIVILEGES ON \`${HOSTRIX_DB_NAME}\`.* TO '${HOSTRIX_DB_USER}'@'localhost'; FLUSH PRIVILEGES;"
  ok "MariaDB database ${HOSTRIX_DB_NAME} ready"
}

install_incus_hint() {
  log "Incus will be required in Phase 2. Skipping Incus install for Phase 1."
}

fetch_source() {
  log "Fetching Hostrix (${HOSTRIX_VERSION})..."
  mkdir -p "${HOSTRIX_INSTALL_DIR}"
  if [[ -d "${HOSTRIX_INSTALL_DIR}/.git" ]]; then
    git -C "${HOSTRIX_INSTALL_DIR}" fetch --depth 1 origin "${HOSTRIX_VERSION}"
    git -C "${HOSTRIX_INSTALL_DIR}" checkout -f "FETCH_HEAD"
  else
    rm -rf "${HOSTRIX_INSTALL_DIR}"
    git clone --depth 1 --branch "${HOSTRIX_VERSION}" \
      "https://github.com/${HOSTRIX_REPO}.git" "${HOSTRIX_INSTALL_DIR}" \
      || git clone --depth 1 "https://github.com/${HOSTRIX_REPO}.git" "${HOSTRIX_INSTALL_DIR}"
  fi
}

build_hostrix() {
  log "Building API..."
  cd "${HOSTRIX_INSTALL_DIR}/api"
  /usr/local/go/bin/go build -o "${HOSTRIX_INSTALL_DIR}/bin/hostrix-api" ./cmd/hostrix-api

  log "Building Panel..."
  cd "${HOSTRIX_INSTALL_DIR}/panel"
  npm ci || npm install
  npm run build
}

write_env() {
  mkdir -p /etc/hostrix
  cat >/etc/hostrix/hostrix.env <<EOF
HOSTRIX_HTTP_ADDR=${HOSTRIX_HTTP_ADDR}
HOSTRIX_PANEL_ORIGIN=${HOSTRIX_PANEL_ORIGIN}
HOSTRIX_DB_HOST=127.0.0.1
HOSTRIX_DB_PORT=3306
HOSTRIX_DB_USER=${HOSTRIX_DB_USER}
HOSTRIX_DB_PASSWORD=${HOSTRIX_DB_PASSWORD}
HOSTRIX_DB_NAME=${HOSTRIX_DB_NAME}
HOSTRIX_BOOTSTRAP_ADMIN_USERNAME=admin
HOSTRIX_BOOTSTRAP_ADMIN_EMAIL=admin@localhost
HOSTRIX_BOOTSTRAP_ADMIN_PASSWORD=${HOSTRIX_ADMIN_PASSWORD}
HOSTRIX_COOKIE_SECURE=false
EOF
  chmod 600 /etc/hostrix/hostrix.env
}

install_systemd() {
  log "Installing systemd units..."
  cp "${HOSTRIX_INSTALL_DIR}/installer/systemd/hostrix-api.service" /etc/systemd/system/
  cp "${HOSTRIX_INSTALL_DIR}/installer/systemd/hostrix-panel.service" /etc/systemd/system/
  systemctl daemon-reload
  systemctl enable --now hostrix-api
  systemctl enable --now hostrix-panel
}

print_summary() {
  local ip
  ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
  echo
  ok "Hostrix Phase 1 installed."
  echo "  Panel     : http://${ip:-127.0.0.1}:3000"
  echo "  API       : http://${ip:-127.0.0.1}:8080"
  echo "  Admin     : admin / ${HOSTRIX_ADMIN_PASSWORD}"
  echo "  Config    : /etc/hostrix/hostrix.env"
  echo "  Install   : ${HOSTRIX_INSTALL_DIR}"
  echo
  echo "Note: Incus/Agent features arrive in Phase 2."
}

main() {
  require_root
  detect_os
  detect_arch
  install_packages
  setup_mariadb
  install_incus_hint
  fetch_source
  mkdir -p "${HOSTRIX_INSTALL_DIR}/bin"
  write_env
  build_hostrix
  install_systemd
  print_summary
}

main "$@"
