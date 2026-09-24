#!/usr/bin/env bash
cd "$(dirname "$0")"
PID_DIR="$(pwd)/.pids"
if [[ -f "$PID_DIR/server.pid" ]]; then
  pid=$(cat "$PID_DIR/server.pid")
  echo "stop pid=$pid"
  kill "$pid" 2>/dev/null || true
  pkill -P "$pid" 2>/dev/null || true
  rm -f "$PID_DIR/server.pid"
fi
if command -v fuser >/dev/null 2>&1; then
  fuser -k 8080/tcp 2>/dev/null || true
fi
echo "stopped"
