#!/usr/bin/env bash
# Optional in-container helper: install or update Dotclear.
# Run INSIDE a Hostrix Dotclear LXC (not on the Hostrix host).
# Usage:
#   sudo bash setup.sh install
#   sudo bash setup.sh update

set -euo pipefail

DOCROOT="${DOCROOT:-/var/www/dotclear}"
PHP_VERSION="${PHP_VERSION:-8.3}"
ACTION="${1:-install}"
DOTCLEAR_ZIP_URL="${DOTCLEAR_ZIP_URL:-https://download.dotclear.org/latest.zip}"

die() { echo "dotclear-setup: $*" >&2; exit 1; }

require_root() {
  [[ "${EUID}" -eq 0 ]] || die "run as root"
}

install_packages() {
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -y
  apt-get install -y nginx curl unzip \
    "php${PHP_VERSION}-fpm" \
    "php${PHP_VERSION}-mysql" \
    "php${PHP_VERSION}-mbstring" \
    "php${PHP_VERSION}-xml" \
    "php${PHP_VERSION}-intl" \
    "php${PHP_VERSION}-gd" \
    "php${PHP_VERSION}-curl" \
    "php${PHP_VERSION}-zip"
}

configure_nginx() {
  cat >/etc/nginx/sites-available/dotclear <<EOF
server {
    listen ${SERVER_PORT:-80} default_server;
    root ${DOCROOT};
    index index.php index.html;
    client_max_body_size 64m;

    location / {
        try_files \$uri \$uri/ /index.php?\$args;
    }

    location ~ [^/]\\.php(/|\$) {
        fastcgi_split_path_info ^(.+?\\.php)(/.*)\$;
        try_files \$fastcgi_script_name =404;
        set \$path_info \$fastcgi_path_info;
        fastcgi_param PATH_INFO \$path_info;
        fastcgi_index index.php;
        include fastcgi.conf;
        fastcgi_pass unix:/run/php/php${PHP_VERSION}-fpm.sock;
    }

    location ~* ^/(inc|cache|db|plugins|themes|var)/ {
        deny all;
        return 404;
    }
}
EOF
  ln -sfn /etc/nginx/sites-available/dotclear /etc/nginx/sites-enabled/dotclear
  rm -f /etc/nginx/sites-enabled/default
}

install_dotclear() {
  mkdir -p "${DOCROOT}"
  if [[ ! -f "${DOCROOT}/index.php" ]]; then
    local tmp
    tmp="$(mktemp -d)"
    curl -fsSL "${DOTCLEAR_ZIP_URL}" -o "${tmp}/dotclear.zip"
    unzip -q "${tmp}/dotclear.zip" -d "${tmp}/extract"
    # Archive usually contains a top-level "dotclear/" directory
    if [[ -d "${tmp}/extract/dotclear" ]]; then
      cp -a "${tmp}/extract/dotclear/." "${DOCROOT}/"
    else
      cp -a "${tmp}/extract/." "${DOCROOT}/"
    fi
    rm -rf "${tmp}"
  fi
  mkdir -p "${DOCROOT}/cache" "${DOCROOT}/public"
  chown -R www-data:www-data "${DOCROOT}"
  chmod -R u+rwX,g+rwX "${DOCROOT}/cache" "${DOCROOT}/public" || true
}

update_dotclear() {
  [[ -f "${DOCROOT}/index.php" ]] || die "Dotclear not found in ${DOCROOT}"
  local tmp backup
  tmp="$(mktemp -d)"
  backup="$(mktemp -d)"
  cp -a "${DOCROOT}/." "${backup}/"
  curl -fsSL "${DOTCLEAR_ZIP_URL}" -o "${tmp}/dotclear.zip"
  unzip -q "${tmp}/dotclear.zip" -d "${tmp}/extract"
  if [[ -d "${tmp}/extract/dotclear" ]]; then
    cp -a "${tmp}/extract/dotclear/." "${DOCROOT}/"
  else
    cp -a "${tmp}/extract/." "${DOCROOT}/"
  fi
  # Preserve local config / content if present
  for keep in inc/config.php db public; do
    if [[ -e "${backup}/${keep}" ]]; then
      rm -rf "${DOCROOT}/${keep}"
      cp -a "${backup}/${keep}" "${DOCROOT}/${keep}"
    fi
  done
  chown -R www-data:www-data "${DOCROOT}"
  rm -rf "${tmp}" "${backup}"
  echo "Dotclear updated. Finish any DB migration from the admin UI if prompted."
}

main() {
  require_root
  case "${ACTION}" in
    install)
      install_packages
      configure_nginx
      install_dotclear
      echo "Dotclear ready in ${DOCROOT}. Open /admin/install/ to finish the wizard."
      ;;
    update)
      update_dotclear
      ;;
    *)
      die "usage: $0 {install|update}"
      ;;
  esac
}

main "$@"
