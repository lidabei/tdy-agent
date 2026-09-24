#!/usr/bin/env bash
set -e
cd "$(dirname "$0")"
ROOT="$(pwd)"
LOG_DIR="$ROOT/.logs"
PID_DIR="$ROOT/.pids"
mkdir -p "$LOG_DIR" "$PID_DIR"

echo "========================================"
echo "  TDY Manager - Build and Start"
echo "========================================"

[[ -f backend/config.yaml ]] || { cp backend/config.example.yaml backend/config.yaml; echo "Edit backend/config.yaml"; exit 1; }

echo "[1/4] build frontend..."
(cd frontend && npm install && npm run build)

echo "[2/4] build backend..."
(cd backend && go build -o tdy-server .)

if [[ -f "$PID_DIR/server.pid" ]]; then
  bash "$ROOT/stop.sh" || true
fi

echo "[3/4] start server..."
(
  cd backend
  nohup ./tdy-server >"$LOG_DIR/server.log" 2>&1 &
  echo $! >"$PID_DIR/server.pid"
)

echo "[4/4] wait health..."
for i in $(seq 1 60); do
  if curl -fsS "http://127.0.0.1:8080/api/health" >/dev/null 2>&1; then
    echo "Ready: http://127.0.0.1:8080/"
    echo "Login: admin / admin123"
    exit 0
  fi
  sleep 2
done
echo "health timeout, check $LOG_DIR/server.log"
exit 1
