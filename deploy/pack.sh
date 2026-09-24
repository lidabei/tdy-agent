#!/usr/bin/env bash
# 在开发机或 CI 上打包 Linux 发布物
# 用法: bash deploy/pack.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/dist-release"
APP="$OUT/tdy-manager"

rm -rf "$OUT"
mkdir -p "$APP/static" "$APP/uploads"

echo "[1/3] build frontend..."
(cd "$ROOT/frontend" && npm install && npm run build)
cp -R "$ROOT/frontend/dist/." "$APP/static/"

echo "[2/3] build linux backend..."
(cd "$ROOT/backend" && \
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "$APP/tdy-server" .)

echo "[3/3] copy deploy files..."
cp "$ROOT/deploy/config.production.yaml.example" "$APP/config.yaml.example"
cp "$ROOT/deploy/tdy-manager.service" "$APP/"
cp "$ROOT/deploy/nginx.conf.example" "$APP/"
cp "$ROOT/deploy/install.sh" "$APP/"
chmod +x "$APP/tdy-server" "$APP/install.sh"

TAR="$OUT/tdy-manager-linux-amd64.tar.gz"
tar -C "$OUT" -czf "$TAR" tdy-manager

echo
echo "OK: $TAR"
echo "上传到服务器后:"
echo "  scp $TAR user@SERVER:/tmp/"
echo "  ssh user@SERVER 'cd /tmp && tar xzf tdy-manager-linux-amd64.tar.gz && sudo bash tdy-manager/install.sh'"
