#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

exec go test -race -count=1 ./...
