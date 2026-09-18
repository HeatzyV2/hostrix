#!/usr/bin/env bash
# Optional in-container helper: install or update WordPress via WP-CLI.
# Intended to run INSIDE a Hostrix WordPress LXC (not on the Hostrix host).
# Usage:
#   sudo bash /opt/hostrix-templates/wordpress/setup.sh install
#   sudo bash /opt/hostrix-templates/wordpress/setup.sh update

set -euo pipefail

DOCROOT="${DOCROOT:-/var/www/html}"
PHP_VERSION="${PHP_VERSION:-8.3}"
ACTION="${1:-install}"

die() { echo "wordpress-setup: $*" >&2; exit 1; }

require_root() {
  [[ "${EUID}" -eq 0 ]] || die "run as root"
}

install_packages() {
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -y
  apt-get install -y nginx "php${PHP_VERSION}-fpm" "php${PHP_VERSION}-mysql" \
    "php${PHP_VERSION}-xml" "php${PHP_VERSION}-curl" "php${PHP_VERSION}-gd" \
    "php${PHP_VERSION}-mbstring" "php${PHP_VERSION}-zip" curl unzip mysql-client
}

install_wpcli() {
  if command -v wp >/dev/null 2>&1; then
    return
  fi
  curl -fsSL https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar -o /usr/local/bin/wp
  chmod +x /usr/local/bin/wp
}

configure_nginx() {
  cat >/etc/nginx/sites-available/wordpress <<EOF
server {
    listen ${SERVER_PORT:-80} default_server;
    root ${DOCROOT};
    index index.php index.html;
    client_max_body_size 64m;

    location / {
        try_files \$uri \$uri/ /index.php?\$args;
    }

    location ~ \.php\$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:/run/php/php${PHP_VERSION}-fpm.sock;
    }
}
EOF
  ln -sfn /etc/nginx/sites-available/wordpress /etc/nginx/sites-enabled/wordpress
  rm -f /etc/nginx/sites-enabled/default
}

install_wordpress() {
  mkdir -p "${DOCROOT}"
  if [[ ! -f "${DOCROOT}/wp-config.php" ]] && [[ ! -f "${DOCROOT}/wp-settings.php" ]]; then
    wp core download --path="${DOCROOT}" --allow-root
  fi
  chown -R www-data:www-data "${DOCROOT}"
}

update_wordpress() {
  [[ -f "${DOCROOT}/wp-settings.php" ]] || die "WordPress not found in ${DOCROOT}"
  wp core update --path="${DOCROOT}" --allow-root
  wp plugin update --all --path="${DOCROOT}" --allow-root || true
  wp theme update --all --path="${DOCROOT}" --allow-root || true
  wp core update-db --path="${DOCROOT}" --allow-root || true
  chown -R www-data:www-data "${DOCROOT}"
  echo "WordPress updated."
}

main() {
  require_root
  case "${ACTION}" in
    install)
      install_packages
      install_wpcli
      configure_nginx
      install_wordpress
      echo "WordPress files ready in ${DOCROOT}. Configure DB via wp config create / panel files."
      ;;
    update)
      install_wpcli
      update_wordpress
      ;;
    *)
      die "usage: $0 {install|update}"
      ;;
  esac
}

main "$@"
