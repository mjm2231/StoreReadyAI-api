#!/usr/bin/env bash
set -euo pipefail

# db_reset.sh
# Truncate (clear) all tables in a MySQL database, keeping schema.
#
# Usage:
#   bash scripts/db_reset.sh
#
# 常见示例：
#   # 1）交互式重置默认数据库
#   bash scripts/db_reset.sh
#
#   # 2）强制重置，不进行确认
#   FORCE=1 bash scripts/db_reset.sh
#
#   # 3）仅预演：只打印命令和目标表，不实际执行
#   DRY_RUN=1 bash scripts/db_reset.sh
#
#   # 4）仅重置指定表
#   ONLY_TABLES=users,subscriptions FORCE=1 bash scripts/db_reset.sh
#
#   # 5）重置时跳过某些表
#   SKIP_TABLES=schema_migrations,users FORCE=1 bash scripts/db_reset.sh
#
#   # 6）使用自定义连接参数
#   MYSQL_HOST=127.0.0.1 MYSQL_PORT=3306 MYSQL_USER=root MYSQL_PASSWORD=123456 MYSQL_DATABASE=storeready_ai FORCE=1 bash scripts/db_reset.sh
#
# Notes:
#   - The script only clears data and keeps table structures.
#   - By default, `schema_migrations` is skipped.
#   - If `.env` exists in the project root, it will be loaded automatically.
#
# Configure via env vars (defaults shown):
#   MYSQL_HOST=127.0.0.1
#   MYSQL_PORT=3306
#   MYSQL_USER=root
#   MYSQL_PASSWORD=
#   MYSQL_DATABASE=storeready_ai
#   MYSQL_CHARSET=utf8mb4
#   MYSQL_SOCKET=          # optional; if set, will be used instead of host/port
#   DRY_RUN=0              # set to 1 to print commands but not execute
#   FORCE=0                # set to 1 to skip interactive confirmation
#   ONLY_TABLES=           # optional comma-separated list, e.g. "users,subscriptions"
#   SKIP_TABLES=           # optional comma-separated list, e.g. "schema_migrations"

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
MYSQL_SOCKET="${MYSQL_SOCKET:-}"
DRY_RUN="${DRY_RUN:-0}"
FORCE="${FORCE:-0}"
ONLY_TABLES="${ONLY_TABLES:-}"
SKIP_TABLES="${SKIP_TABLES:-schema_migrations}"

#
# Discover mysql client binary (same as db_init.sh)
MYSQL_BIN_CANDIDATES=(
  "${ROOT_DIR}/.dev/bin/mysql/bin/mysql"
)

MYSQL_BIN=""
for c in "${MYSQL_BIN_CANDIDATES[@]}"; do
  if [[ -f "${c}" ]]; then
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
  echo "[db_reset] ERROR: mysql client not found. Install MySQL or run 'make dev-setup'." >&2
  exit 1
fi

# Build mysql args
MYSQL_ARGS=("-u" "${MYSQL_USER}" "--default-character-set=${MYSQL_CHARSET}")

if [[ -n "${MYSQL_SOCKET}" ]]; then
  MYSQL_ARGS+=("--socket=${MYSQL_SOCKET}")
else
  MYSQL_ARGS+=("-h" "${MYSQL_HOST}" "-P" "${MYSQL_PORT}")
fi

export MYSQL_PWD="${MYSQL_PASSWORD}"

run() {
  if [[ "${DRY_RUN}" == "1" ]]; then
    printf '+ '
    printf '%q ' "$@"
    printf '\n'
  else
    "$@"
  fi
}

mysql_exec() {
  local sql="$1"
  if [[ -n "${MYSQL_SOCKET}" ]]; then
    run "${MYSQL_BIN}" "${MYSQL_ARGS[@]}" -e "${sql}"
  else
    run "${MYSQL_BIN}" --protocol=TCP "${MYSQL_ARGS[@]}" -e "${sql}"
  fi
}

mysql_exec_db() {
  local db="$1"
  local sql="$2"
  if [[ -n "${MYSQL_SOCKET}" ]]; then
    run "${MYSQL_BIN}" "${MYSQL_ARGS[@]}" "${db}" -e "${sql}"
  else
    run "${MYSQL_BIN}" --protocol=TCP "${MYSQL_ARGS[@]}" "${db}" -e "${sql}"
  fi
}

echo "[db_reset] Using mysql: ${MYSQL_BIN}"
echo "[db_reset] Target database: ${MYSQL_DATABASE}"

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
  echo "[db_reset] ERROR: MySQL not reachable (host=${MYSQL_HOST} port=${MYSQL_PORT} socket=${MYSQL_SOCKET})." >&2
  exit 1
fi

# Confirm database exists
DB_EXISTS="$(mysql_exec "SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME='${MYSQL_DATABASE}'" 2>/dev/null | tail -n 1 || true)"
if [[ "${DB_EXISTS}" != "${MYSQL_DATABASE}" ]]; then
  echo "[db_reset] ERROR: database '${MYSQL_DATABASE}' does not exist." >&2
  exit 1
fi

# Parse ONLY_TABLES / SKIP_TABLES into sql IN lists
to_in_list() {
  local s="$1"
  s="$(echo "${s}" | tr -d '[:space:]')"
  [[ -z "${s}" ]] && { echo ""; return 0; }
  local out=""
  IFS=',' read -r -a arr <<< "${s}"
  for t in "${arr[@]}"; do
    [[ -z "${t}" ]] && continue
    # escape backticks by doubling (simple)
    out="${out}'${t}',"
  done
  out="${out%,}"
  echo "${out}"
}

ONLY_IN="$(to_in_list "${ONLY_TABLES}")"
SKIP_IN="$(to_in_list "${SKIP_TABLES}")"

# List tables
LIST_SQL="SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA='${MYSQL_DATABASE}' AND TABLE_TYPE='BASE TABLE'"
if [[ -n "${ONLY_IN}" ]]; then
  LIST_SQL="${LIST_SQL} AND TABLE_NAME IN (${ONLY_IN})"
fi
if [[ -n "${SKIP_IN}" ]]; then
  LIST_SQL="${LIST_SQL} AND TABLE_NAME NOT IN (${SKIP_IN})"
fi
LIST_SQL="${LIST_SQL} ORDER BY TABLE_NAME;"

TABLES_RAW="$(mysql_exec "${LIST_SQL}" 2>/dev/null || true)"
TABLES="$(echo "${TABLES_RAW}" | tail -n +2 | sed '/^$/d' || true)"

if [[ -z "${TABLES}" ]]; then
  echo "[db_reset] No tables to truncate (maybe excluded by ONLY_TABLES/SKIP_TABLES)."
  exit 0
fi

echo "[db_reset] Tables to truncate:"
echo "${TABLES}" | sed 's/^/  - /'

if [[ "${FORCE}" != "1" ]]; then
  echo
  echo "⚠️  This will TRUNCATE ALL DATA in database '${MYSQL_DATABASE}'. Schema will be kept."
  echo "Type the database name to continue: "
  read -r confirm
  if [[ "${confirm}" != "${MYSQL_DATABASE}" ]]; then
    echo "[db_reset] Abort."
    exit 1
  fi
fi

# Build one SQL batch: disable FK checks, truncate tables, enable FK checks.
# Use backticks to quote table names safely.
BATCH="SET FOREIGN_KEY_CHECKS=0;"
while IFS= read -r t; do
  [[ -z "${t}" ]] && continue
  BATCH="${BATCH} TRUNCATE TABLE \`${t}\`;"
done <<< "${TABLES}"
BATCH="${BATCH} SET FOREIGN_KEY_CHECKS=1;"

mysql_exec_db "${MYSQL_DATABASE}" "${BATCH}"

echo "[db_reset] Done. All data truncated."