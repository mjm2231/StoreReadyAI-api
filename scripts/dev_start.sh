#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

BIN="${ROOT}/.dev/bin"
ETC="${ROOT}/.dev/etc"
DATA="${ROOT}/.data"
LOGS="${DATA}/logs"

mkdir -p "${DATA}/"{mysql,redis,logs}

print_log_tail() {
  local title="$1"
  local file="$2"

  if [[ -f "${file}" ]]; then
    echo "==> ${title}: ${file}" >&2
    tail -n 80 "${file}" >&2 || true
  else
    echo "==> ${title}: ${file} not found" >&2
  fi
}

read_conf_port() {
  local file="$1"
  local default_port="$2"

  if [[ -f "${file}" ]]; then
    awk '
      $1 == "port" && $2 ~ /^[0-9]+$/ { print $2; found=1; exit }
      END { if (!found) print "'"${default_port}"'" }
    ' "${file}"
  else
    echo "${default_port}"
  fi
}

wait_for_tcp() {
  local name="$1"
  local host="$2"
  local port="$3"
  local max_attempts="${4:-30}"
  local pid_file="${5:-}"
  local err_log="${6:-}"
  local out_log="${7:-}"

  for ((i=1; i<=max_attempts; i++)); do
    if nc -z "${host}" "${port}" >/dev/null 2>&1; then
      echo "==> ${name} ready (${host}:${port})"
      return 0
    fi

    if [[ -n "${pid_file}" && -f "${pid_file}" ]]; then
      local pid
      pid="$(cat "${pid_file}" || true)"
      if [[ -n "${pid}" ]] && ! kill -0 "${pid}" >/dev/null 2>&1; then
        echo "ERROR: ${name} process exited before ready, pid=${pid}" >&2
        [[ -n "${err_log}" ]] && print_log_tail "${name} stderr" "${err_log}"
        [[ -n "${out_log}" ]] && print_log_tail "${name} stdout" "${out_log}"
        return 1
      fi
    fi

    sleep 1
  done

  echo "ERROR: ${name} not ready after ${max_attempts}s (${host}:${port})" >&2
  [[ -n "${err_log}" ]] && print_log_tail "${name} stderr" "${err_log}"
  [[ -n "${out_log}" ]] && print_log_tail "${name} stdout" "${out_log}"
  return 1
}

stop_if_running() {
  local name="$1"
  local pid_file="$2"

  if [[ -f "${pid_file}" ]]; then
    local pid
    pid="$(cat "${pid_file}" || true)"
    if [[ -n "${pid}" ]] && kill -0 "${pid}" >/dev/null 2>&1; then
      echo "==> Stop existing ${name} pid=${pid}"
      kill "${pid}" >/dev/null 2>&1 || true
      sleep 1
    fi
    rm -f "${pid_file}"
  fi
}

stop_if_running "API" "${DATA}/api.pid"
stop_if_running "Redis" "${DATA}/redis/redis.pid"
stop_if_running "MySQL" "${DATA}/mysql/mysql.pid"

# 清理 MySQL 可能残留的 pid，避免异常退出后无法启动。
rm -f "${DATA}/mysql/"*.pid
rm -f "${DATA}/redis/"*.pid

REDIS_PORT="$(read_conf_port "${ETC}/redis.conf" "6379")"

if [[ ! -x "${BIN}/redis/redis-server" ]]; then
  echo "ERROR: redis-server not found or not executable: ${BIN}/redis/redis-server" >&2
  exit 1
fi

echo "==> Start Redis"
"${BIN}/redis/redis-server" "${ETC}/redis.conf" >"${LOGS}/redis.out.log" 2>"${LOGS}/redis.err.log" &
echo $! > "${DATA}/redis/redis.pid"
wait_for_tcp "Redis" "127.0.0.1" "${REDIS_PORT}" 15 "${DATA}/redis/redis.pid" "${LOGS}/redis.err.log" "${LOGS}/redis.out.log"

echo "==> Start MySQL"
if [[ ! -d "${DATA}/mysql/mysql" ]]; then
  echo "==> MySQL init datadir"
  "${BIN}/mysql/bin/mysqld" --initialize-insecure --datadir="${DATA}/mysql" >"${LOGS}/mysql-init.out.log" 2>"${LOGS}/mysql-init.err.log"
fi
"${BIN}/mysql/bin/mysqld" --defaults-file="${ETC}/my.cnf" >"${LOGS}/mysql.out.log" 2>"${LOGS}/mysql.err.log" &
echo $! > "${DATA}/mysql/mysql.pid"
wait_for_tcp "MySQL" "127.0.0.1" "3306" 30 "${DATA}/mysql/mysql.pid" "${LOGS}/mysql.err.log" "${LOGS}/mysql.out.log"

echo "==> Ensure database"
"${BIN}/mysql/bin/mysql" -h127.0.0.1 -uroot -e "CREATE DATABASE IF NOT EXISTS storeready_ai DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;"

echo "==> Start API"
(
  cd "${ROOT}"
  go run ./cmd/api -c configs/dev.yaml
) >"${LOGS}/api.out.log" 2>"${LOGS}/api.err.log" &
echo $! > "${DATA}/api.pid"

echo "==> Done"
echo "==> Logs: ${LOGS}"