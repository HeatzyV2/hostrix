#!/usr/bin/env bash
# Alias: upgrade.sh → update.sh (kept for older docs)
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec bash "${SCRIPT_DIR}/update.sh" "$@"
