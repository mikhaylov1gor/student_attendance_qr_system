#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

if [[ $# -lt 1 ]]; then
  echo "Использование: scripts/migrate-new.sh <name>" >&2
  echo "Пример:        scripts/migrate-new.sh add_refresh_tokens" >&2
  exit 2
fi

exec go run ./cmd/migrate create "$1"
