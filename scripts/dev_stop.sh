#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATA="${ROOT}/.data"

kill_if() {
  local f="$1"
  [[ -f "${f}" ]] || return 0

  local pid
  pid="$(cat "${f}" 2>/dev/null || true)"
  if [[ -z "${pid}" ]]; then
    rm -f "${f}"
    return 0
  fi

  # If process does not exist, remove stale pid file.
  if ! kill -0 "${pid}" >/dev/null 2>&1; then
    rm -f "${f}"
    return 0
  fi

  # Try graceful stop.
  kill "${pid}" >/dev/null 2>&1 || true

  # Wait up to 3s for exit.
  local i
  for i in {1..30}; do
    if ! kill -0 "${pid}" >/dev/null 2>&1; then
      rm -f "${f}"
      return 0
    fi
    sleep 0.1
  done

  # Force kill.
  kill -9 "${pid}" >/dev/null 2>&1 || true

  # Wait up to 2s for exit.
  for i in {1..20}; do
    if ! kill -0 "${pid}" >/dev/null 2>&1; then
      rm -f "${f}"
      return 0
    fi
    sleep 0.1
  done

  echo "[warn] failed to kill pid=${pid} from ${f}" >&2
  # Do NOT remove pidfile so user can inspect.
}

kill_if "${DATA}/api.pid"
kill_if "${DATA}/redis/redis.pid"
kill_if "${DATA}/mysql/mysql.pid"

kill_listen_port() {
  local port="$1"
  # Find listener PIDs (macOS lsof)
  local pids
  pids="$(lsof -nP -t -iTCP:${port} -sTCP:LISTEN 2>/dev/null || true)"
  [[ -z "${pids}" ]] && return 0

  echo "[db_stop] found listener(s) on :${port}: ${pids}" >&2

  # Try TERM first.
  kill ${pids} >/dev/null 2>&1 || true

  # Wait a bit, then force kill if still listening.
  sleep 0.3

  # Repeat a few rounds because some dev runners may respawn the process quickly.
  local i
  for i in {1..30}; do
    pids="$(lsof -nP -t -iTCP:${port} -sTCP:LISTEN 2>/dev/null || true)"
    [[ -z "${pids}" ]] && return 0
    kill -9 ${pids} >/dev/null 2>&1 || true
    sleep 0.1
  done

  # Still listening
  pids="$(lsof -nP -t -iTCP:${port} -sTCP:LISTEN 2>/dev/null || true)"
  if [[ -n "${pids}" ]]; then
    echo "[warn] port :${port} still has listener(s): ${pids}" >&2
  fi
}

# Fallback: in case pid files were stale/missing, ensure common dev ports are freed.
kill_listen_port 8080
kill_listen_port 6379
kill_listen_port 3306

# Extra fallback: kill common dev binaries by name

kill_by_name() {
  local name="$1"
  # macOS pkill: try TERM then KILL
  if pgrep -x "${name}" >/dev/null 2>&1; then
    echo "[db_stop] pkill -TERM ${name}" >&2
    pkill -TERM -x "${name}" >/dev/null 2>&1 || true
    sleep 0.3
    if pgrep -x "${name}" >/dev/null 2>&1; then
      echo "[db_stop] pkill -KILL ${name}" >&2
      pkill -KILL -x "${name}" >/dev/null 2>&1 || true
    fi
  fi
}

kill_by_name api
kill_by_name redis-server
kill_by_name mysqld

echo "==> stopped"