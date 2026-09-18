#!/usr/bin/env bash
# Hostrix installer (Linux — Ubuntu/Debian)
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/HeatzyV2/hostrix/main/installer/install.sh | sudo bash
# Non-interactive:
#   curl -fsSL ... | sudo env HOSTRIX_NONINTERACTIVE=1 HOSTRIX_BOOTSTRAP_ADMIN_PASSWORD='…' bash

set -euo pipefail

HOSTRIX_VERSION="${HOSTRIX_VERSION:-main}"
HOSTRIX_REPO="${HOSTRIX_REPO:-HeatzyV2/hostrix}"
HOSTRIX_INSTALL_DIR="${HOSTRIX_INSTALL_DIR:-/opt/hostrix}"
HOSTRIX_NONINTERACTIVE="${HOSTRIX_NONINTERACTIVE:-0}"
HOSTRIX_DB_NAME="${HOSTRIX_DB_NAME:-hostrix}"
HOSTRIX_DB_USER="${HOSTRIX_DB_USER:-hostrix}"
HOSTRIX_HTTP_ADDR="${HOSTRIX_HTTP_ADDR:-:8080}"
HOSTRIX_PANEL_PORT="${HOSTRIX_PANEL_PORT:-3000}"

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

# curl|bash has no usable TTY on stdin — force non-interactive or read from /dev/tty
ensure_prompt_mode() {
  if [[ ! -t 0 ]]; then
    if [[ "${HOSTRIX_NONINTERACTIVE}" != "1" ]] && [[ ! -e /dev/tty ]]; then
      HOSTRIX_NONINTERACTIVE=1
      log "No TTY detected — switching to non-interactive defaults"
    elif [[ "${HOSTRIX_NONINTERACTIVE}" != "1" ]]; then
      log "Running via pipe — prompts will use /dev/tty"
    fi
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
  OS_CODENAME="${VERSION_CODENAME:-}"
  case "${OS_ID}" in
    ubuntu|debian) ;;
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
    if [[ -e /dev/tty ]]; then
      read -r -p "${msg} [${default}]: " input </dev/tty || true
    else
      read -r -p "${msg} [${default}]: " input || true
    fi
    printf -v "${var}" '%s' "${input:-$default}"
  else
    if [[ -e /dev/tty ]]; then
      read -r -p "${msg}: " input </dev/tty
    else
      read -r -p "${msg}: " input
    fi
    printf -v "${var}" '%s' "${input}"
  fi
}

go_bin() {
  if [[ -x /usr/local/go/bin/go ]]; then
    echo /usr/local/go/bin/go
  elif command -v go >/dev/null 2>&1; then
    command -v go
  else
    die "Go toolchain not found after install"
  fi
}

install_packages() {
  log "Installing system packages..."
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -y
  apt-get install -y curl ca-certificates gnupg lsb-release git build-essential \
    openssl mariadb-server mariadb-client fail2ban

  if ! command -v go >/dev/null 2>&1 && [[ ! -x /usr/local/go/bin/go ]]; then
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
  if [[ "${HOSTRIX_ADMIN_PASSWORD}" == "changeme" ]]; then
    log "WARNING: using default admin password 'changeme' — change it after login"
  fi

  prompt HOSTRIX_DB_PASSWORD_INPUT "MariaDB password for user ${HOSTRIX_DB_USER}" "${db_pass}"
  HOSTRIX_DB_PASSWORD="${HOSTRIX_DB_PASSWORD_INPUT}"

  mysql -e "CREATE DATABASE IF NOT EXISTS \`${HOSTRIX_DB_NAME}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
  mysql -e "CREATE USER IF NOT EXISTS '${HOSTRIX_DB_USER}'@'localhost' IDENTIFIED BY '${HOSTRIX_DB_PASSWORD}';"
  mysql -e "ALTER USER '${HOSTRIX_DB_USER}'@'localhost' IDENTIFIED BY '${HOSTRIX_DB_PASSWORD}';"
  mysql -e "GRANT ALL PRIVILEGES ON \`${HOSTRIX_DB_NAME}\`.* TO '${HOSTRIX_DB_USER}'@'localhost'; FLUSH PRIVILEGES;"
  ok "MariaDB database ${HOSTRIX_DB_NAME} ready"
}

install_incus() {
  if command -v incus >/dev/null 2>&1; then
    ok "Incus already installed"
  else
    log "Installing Incus (Zabbly)..."
    mkdir -p /etc/apt/keyrings
    if ! curl -fsSL https://pkgs.zabbly.com/key.asc -o /etc/apt/keyrings/zabbly.asc; then
      die "Failed to download Zabbly Incus signing key. Install Incus manually, then re-run."
    fi
    if [[ -z "${OS_CODENAME}" ]]; then
      die "Could not detect VERSION_CODENAME for Incus apt repo"
    fi
    echo "deb [signed-by=/etc/apt/keyrings/zabbly.asc] https://pkgs.zabbly.com/incus/stable ${OS_CODENAME} main" \
      >/etc/apt/sources.list.d/zabbly-incus-stable.list
    apt-get update -y
    apt-get install -y incus
  fi

  if ! command -v incus >/dev/null 2>&1; then
    die "Incus installation failed"
  fi

  if ! incus info >/dev/null 2>&1; then
    log "Initializing Incus (auto)..."
    incus admin init --auto || die "incus admin init failed"
  fi
  ok "Incus ready"
}

install_fail2ban() {
  log "Configuring Fail2Ban..."
  mkdir -p /etc/fail2ban/jail.d
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
findtime = 10m
bantime  = 1h

[hostrix-api]
enabled  = true
port     = 8080
filter   = hostrix-auth
logpath  = /var/log/hostrix/auth-fail.log
maxretry = 10
findtime = 10m
bantime  = 1h
EOF

  mkdir -p /etc/fail2ban/filter.d
  cat >/etc/fail2ban/filter.d/hostrix-auth.conf <<'EOF'
[Definition]
failregex = ^.*hostrix.*(invalid credentials|unauthorized|too many login attempts).*$
            ^.*POST /api/v1/auth/login.* (401|429).*$
ignoreregex =
EOF

  mkdir -p /var/log/hostrix
  touch /var/log/hostrix/auth-fail.log
  chmod 640 /var/log/hostrix/auth-fail.log

  systemctl enable --now fail2ban
  systemctl restart fail2ban
  ok "Fail2Ban enabled (sshd + Hostrix auth jails)"
}

fetch_source() {
  log "Fetching Hostrix (${HOSTRIX_VERSION})..."
  mkdir -p "$(dirname "${HOSTRIX_INSTALL_DIR}")"
  if [[ -d "${HOSTRIX_INSTALL_DIR}/.git" ]]; then
    git -C "${HOSTRIX_INSTALL_DIR}" fetch --depth 1 origin "${HOSTRIX_VERSION}"
    git -C "${HOSTRIX_INSTALL_DIR}" checkout -f "FETCH_HEAD"
  else
    rm -rf "${HOSTRIX_INSTALL_DIR}"
    if ! git clone --depth 1 --branch "${HOSTRIX_VERSION}" \
      "https://github.com/${HOSTRIX_REPO}.git" "${HOSTRIX_INSTALL_DIR}"; then
      git clone --depth 1 "https://github.com/${HOSTRIX_REPO}.git" "${HOSTRIX_INSTALL_DIR}"
    fi
  fi
}

build_hostrix() {
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

detect_public_ip() {
  local ip
  ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
  echo "${ip:-127.0.0.1}"
}

write_env() {
  local ip panel_origin
  ip="$(detect_public_ip)"
  panel_origin="${HOSTRIX_PANEL_ORIGIN:-http://${ip}:${HOSTRIX_PANEL_PORT}}"

  mkdir -p /etc/hostrix
  cat >/etc/hostrix/hostrix.env <<EOF
HOSTRIX_HTTP_ADDR=${HOSTRIX_HTTP_ADDR}
HOSTRIX_PANEL_ORIGIN=${panel_origin}
HOSTRIX_DB_HOST=127.0.0.1
HOSTRIX_DB_PORT=3306
HOSTRIX_DB_USER=${HOSTRIX_DB_USER}
HOSTRIX_DB_PASSWORD=${HOSTRIX_DB_PASSWORD}
HOSTRIX_DB_NAME=${HOSTRIX_DB_NAME}
HOSTRIX_BOOTSTRAP_ADMIN_USERNAME=admin
HOSTRIX_BOOTSTRAP_ADMIN_EMAIL=admin@localhost
HOSTRIX_BOOTSTRAP_ADMIN_PASSWORD=${HOSTRIX_ADMIN_PASSWORD}
HOSTRIX_COOKIE_SECURE=false
HOSTRIX_TEMPLATES_DIR=${HOSTRIX_INSTALL_DIR}/templates
EOF
  chmod 600 /etc/hostrix/hostrix.env

  mkdir -p /etc/systemd/system/hostrix-panel.service.d
  cat >/etc/systemd/system/hostrix-panel.service.d/override.conf <<EOF
[Service]
Environment=HOSTRIX_API_URL=http://127.0.0.1:8080
Environment=HOSTRIX_API_INTERNAL_URL=http://127.0.0.1:8080
Environment=NEXT_PUBLIC_HOSTRIX_API_URL=http://${ip}:8080
Environment=PORT=${HOSTRIX_PANEL_PORT}
EOF
}

install_systemd() {
  log "Installing systemd units..."
  cp "${HOSTRIX_INSTALL_DIR}/installer/systemd/hostrix-api.service" /etc/systemd/system/
  cp "${HOSTRIX_INSTALL_DIR}/installer/systemd/hostrix-panel.service" /etc/systemd/system/
  cp "${HOSTRIX_INSTALL_DIR}/installer/systemd/hostrix-agent.service" /etc/systemd/system/
  systemctl daemon-reload
  systemctl enable --now hostrix-api
  systemctl enable --now hostrix-panel
}

print_summary() {
  local ip
  ip="$(detect_public_ip)"
  echo
  ok "Hostrix installed."
  echo "  Panel     : http://${ip}:${HOSTRIX_PANEL_PORT}"
  echo "  API       : http://${ip}:8080"
  echo "  Admin     : admin / ${HOSTRIX_ADMIN_PASSWORD}"
  echo "  Config    : /etc/hostrix/hostrix.env"
  echo "  Install   : ${HOSTRIX_INSTALL_DIR}"
  echo "  Fail2Ban  : sshd + hostrix-api/panel auth jails"
  echo
  echo "Next (mono-node):"
  echo "  1. Open the panel and create a Node (address 127.0.0.1, port 8081)"
  echo "  2. Save the one-time token to /etc/hostrix/agent.env — see .env.agent.example"
  echo "  3. systemctl enable --now hostrix-agent"
  echo
}

main() {
  require_root
  ensure_prompt_mode
  detect_os
  detect_arch
  install_packages
  setup_mariadb
  install_incus
  install_fail2ban
  fetch_source
  write_env
  build_hostrix
  install_systemd
  print_summary
}

main "$@"
