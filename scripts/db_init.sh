

#!/usr/bin/env bash
set -euo pipefail

# db_init.sh
# Initialize MySQL database and import schema/seed SQL files.
#
# Supports both local dev binaries (./.dev/bin/mysql) and system mysql.
# Configure via environment variables (defaults shown):
#   MYSQL_HOST=127.0.0.1
#   MYSQL_PORT=3306
#   MYSQL_USER=root
#   MYSQL_PASSWORD=
#   MYSQL_DATABASE=storeready_ai
#   MYSQL_CHARSET=utf8mb4
#   MYSQL_COLLATION=utf8mb4_unicode_ci
#   MYSQL_SOCKET=          # optional; if set, will be used instead of host/port
#   SQL_DIR=               # optional; if set, will be used as the only SQL directory
#   DRY_RUN=0              # set to 1 to print commands but not execute

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Load .env if present (without overriding already-exported vars)
if [[ -f "${ROOT_DIR}/.env" ]]; then
  # shellcheck disable=SC1090
  set -a
  source "${ROOT_DIR}/.env"
  set +a
fi

MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-root}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-}"
MYSQL_DATABASE="${MYSQL_DATABASE:-storeready_ai}"
MYSQL_CHARSET="${MYSQL_CHARSET:-utf8mb4}"
MYSQL_COLLATION="${MYSQL_COLLATION:-utf8mb4_unicode_ci}"
MYSQL_SOCKET="${MYSQL_SOCKET:-}"
SQL_DIR="${SQL_DIR:-}"
DRY_RUN="${DRY_RUN:-0}"

#
# Discover mysql client binary.
# Our dev-setup may install a full MySQL distribution under .dev/bin/{bin,lib,share}
# where the actual client is .dev/bin/bin/mysql.
MYSQL_BIN_CANDIDATES=(
  # Layout B: full MySQL distribution extracted to .dev/bin/mysql/{bin,lib,share}
  "${ROOT_DIR}/.dev/bin/mysql/bin/mysql"
)

MYSQL_BIN=""
for c in "${MYSQL_BIN_CANDIDATES[@]}"; do
  if [[ -f "${c}" ]]; then
    # Ensure it's executable (sometimes tar extraction loses +x)
    chmod +x "${c}" 2>/dev/null || true
  fi
  if [[ -x "${c}" ]]; then
    MYSQL_BIN="${c}"
    break
  fi
done

if [[ -z "${MYSQL_BIN}" ]]; then
  MYSQL_BIN="$(command -v mysql || true)"
fi

if [[ -z "${MYSQL_BIN}" || -d "${MYSQL_BIN}" ]]; then
  echo "[db_init] ERROR: mysql client not found. Install MySQL or run 'make dev-setup'." >&2
  exit 1
fi

# Build mysql args
MYSQL_ARGS=("-u" "${MYSQL_USER}" "--default-character-set=${MYSQL_CHARSET}")

# Prefer socket if provided
if [[ -n "${MYSQL_SOCKET}" ]]; then
  MYSQL_ARGS+=("--socket=${MYSQL_SOCKET}")
else
  MYSQL_ARGS+=("-h" "${MYSQL_HOST}" "-P" "${MYSQL_PORT}")
fi

# Password handling: don't echo password; pass via env var for mysql client
# mysql supports MYSQL_PWD, which avoids exposing password in process args.
export MYSQL_PWD="${MYSQL_PASSWORD}"

run() {
  if [[ "${DRY_RUN}" == "1" ]]; then
    # Print argv safely
    printf '+ '
    printf '%q ' "$@"
    printf '\n'
  else
    "$@"
  fi
}

mysql_exec() {
  local sql="$1"
  # Use --protocol=TCP when host/port are used; keeps behavior consistent on macOS.
  if [[ -n "${MYSQL_SOCKET}" ]]; then
    run "${MYSQL_BIN}" "${MYSQL_ARGS[@]}" -e "${sql}"
  else
    run "${MYSQL_BIN}" --protocol=TCP "${MYSQL_ARGS[@]}" -e "${sql}"
  fi
}

mysql_import_file() {
  local db="$1"
  local file="$2"
  echo "[db_init] Import: ${file}"
  if [[ -n "${MYSQL_SOCKET}" ]]; then
    run "${MYSQL_BIN}" "${MYSQL_ARGS[@]}" "${db}" --default-character-set="${MYSQL_CHARSET}" -e "SOURCE ${file};"
  else
    run "${MYSQL_BIN}" --protocol=TCP "${MYSQL_ARGS[@]}" "${db}" --default-character-set="${MYSQL_CHARSET}" -e "SOURCE ${file};"
  fi
}

echo "[db_init] Using mysql: ${MYSQL_BIN}"
echo "[db_init] Target database: ${MYSQL_DATABASE}"

# Wait for MySQL to be ready (up to 20s)
READY=0
for _ in $(seq 1 40); do
  if mysql_exec "SELECT 1" >/dev/null 2>&1; then
    READY=1
    break
  fi
  sleep 0.5
done

if [[ "${READY}" != "1" ]]; then
  echo "[db_init] ERROR: MySQL not reachable (host=${MYSQL_HOST} port=${MYSQL_PORT} socket=${MYSQL_SOCKET})." >&2
  exit 1
fi

# Create database if missing
mysql_exec "CREATE DATABASE IF NOT EXISTS \`${MYSQL_DATABASE}\` CHARACTER SET ${MYSQL_CHARSET} COLLATE ${MYSQL_COLLATION};"

# Determine SQL directories
SQL_DIR_CANDIDATES=()
if [[ -n "${SQL_DIR}" ]]; then
  SQL_DIR_CANDIDATES+=("${SQL_DIR}")
else
  # This repo convention: ./sqls/00_xxx.sql
  SQL_DIR_CANDIDATES+=("${ROOT_DIR}/sqls")
fi

FOUND_SQL_DIRS=()
for d in "${SQL_DIR_CANDIDATES[@]}"; do
  if [[ -d "${d}" ]]; then
    shopt -s nullglob
    files=("${d}"/*.sql)
    shopt -u nullglob
    if (( ${#files[@]} > 0 )); then
      FOUND_SQL_DIRS+=("${d}")
    fi
  fi
done

if (( ${#FOUND_SQL_DIRS[@]} == 0 )); then
  echo "[db_init] WARN: No .sql files found. Searched:" >&2
  for d in "${SQL_DIR_CANDIDATES[@]}"; do
    echo "  - ${d}" >&2
  done
  echo "[db_init] Database created (if needed). Nothing else to do."
  exit 0
fi

# Import SQL files in deterministic order
# Rule: lexical order within each directory (00_xxx.sql, 01_xxx.sql ...); directories imported in the order discovered.
for d in "${FOUND_SQL_DIRS[@]}"; do
  echo "[db_init] Importing SQL from: ${d}"
  shopt -s nullglob
  tmp_list="$(mktemp)"
  ls -1 "${d}"/*.sql 2>/dev/null | sort > "${tmp_list}" || true
  shopt -u nullglob

  while IFS= read -r f; do
    [[ -z "${f}" ]] && continue
    mysql_import_file "${MYSQL_DATABASE}" "${f}"
  done < "${tmp_list}"

  rm -f "${tmp_list}"
done

echo "[db_init] Done."