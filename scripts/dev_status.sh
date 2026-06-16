#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATA="${ROOT}/.data"

check() {
  local name="$1" f="$2"
  if [[ -f "${f}" ]] && kill -0 "$(cat "${f}")" >/dev/null 2>&1; then
    echo "[OK] ${name} pid=$(cat "${f}")"
  else
    echo "[DOWN] ${name}"
  fi
}

check api   "${DATA}/api.pid"
check redis "${DATA}/redis/redis.pid"
check mysql "${DATA}/mysql/mysql.pid"