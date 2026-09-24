#!/usr/bin/env bash
# 在 Linux 服务器上安装/升级 TDY Manager
# 用法: sudo bash install.sh [/opt/tdy-manager]
set -euo pipefail

SRC="$(cd "$(dirname "$0")" && pwd)"
DEST="${1:-/opt/tdy-manager}"

if [[ "$(id -u)" -ne 0 ]]; then
  echo "请使用 root 或 sudo 运行"
  exit 1
fi

if [[ ! -x "$SRC/tdy-server" ]]; then
  echo "未找到可执行文件: $SRC/tdy-server"
  exit 1
fi

echo "==> 安装目录: $DEST"
id tdy >/dev/null 2>&1 || useradd --system --home "$DEST" --shell /usr/sbin/nologin tdy

mkdir -p "$DEST/uploads" "$DEST/static"
install -m 755 "$SRC/tdy-server" "$DEST/tdy-server"

if [[ -d "$SRC/static" ]]; then
  rm -rf "$DEST/static"
  cp -a "$SRC/static" "$DEST/static"
fi

if [[ ! -f "$DEST/config.yaml" ]]; then
  if [[ -f "$SRC/config.yaml.example" ]]; then
    cp "$SRC/config.yaml.example" "$DEST/config.yaml"
  elif [[ -f "$SRC/config.yaml" ]]; then
    cp "$SRC/config.yaml" "$DEST/config.yaml"
  fi
  echo
  echo "!!! 请编辑配置后再启动:"
  echo "    nano $DEST/config.yaml"
  echo "    - database 连接信息"
  echo "    - jwt_secret（openssl rand -hex 32）"
  echo "    - cors_origins（你的域名/IP）"
  echo "    - mode: release"
  echo
else
  echo "保留已有配置: $DEST/config.yaml"
fi

chown -R tdy:tdy "$DEST"
chmod 640 "$DEST/config.yaml" 2>/dev/null || true

if [[ -f "$SRC/tdy-manager.service" ]]; then
  cp "$SRC/tdy-manager.service" /etc/systemd/system/tdy-manager.service
  systemctl daemon-reload
  systemctl enable tdy-manager
fi

echo
echo "数据库准备（如尚未创建）:"
echo "  sudo -u postgres psql -c \"CREATE USER tdy WITH PASSWORD '强密码';\""
echo "  sudo -u postgres psql -c \"CREATE DATABASE tdy_manager OWNER tdy;\""
echo
echo "启动:"
echo "  systemctl start tdy-manager"
echo "  systemctl status tdy-manager"
echo "  curl -s http://127.0.0.1:8080/api/health"
echo
echo "可选 Nginx: 参考 $DEST/nginx.conf.example 或 $SRC/nginx.conf.example"
echo "完成。"
