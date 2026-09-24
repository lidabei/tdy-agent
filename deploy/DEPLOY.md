# 服务器部署

前后端一体：一个 `tdy-server` 进程托管 API + 前端静态页，依赖 PostgreSQL。

## 快速流程（Windows 打包 → Linux 安装）

### 1. 本机打包

双击或在项目根目录执行：

```bat
deploy\pack.bat
```

得到目录：`dist-release\tdy-manager\`（含 `tdy-server`、`static/`、配置示例、安装脚本）。

### 2. 上传到服务器

```powershell
scp -r dist-release\tdy-manager user@你的服务器IP:/tmp/
```

### 3. 服务器安装

```bash
ssh user@你的服务器IP
sudo bash /tmp/tdy-manager/install.sh
# 默认安装到 /opt/tdy-manager
```

### 4. 改生产配置

```bash
sudo nano /opt/tdy-manager/config.yaml
```

必改项：

| 项 | 说明 |
|----|------|
| `database.*` | PostgreSQL 地址/库名/账号密码 |
| `server.mode` | 必须是 `release` |
| `server.jwt_secret` | `openssl rand -hex 32` 生成 |
| `server.cors_origins` | 你的访问地址，如 `http://IP` 或 `https://域名` |
| `server.static_dir` | 保持 `static`（相对 `/opt/tdy-manager`） |

### 5. 准备数据库

```bash
sudo -u postgres psql <<'SQL'
CREATE USER tdy WITH PASSWORD '换成强密码';
CREATE DATABASE tdy_manager OWNER tdy;
SQL
```

把上面密码写进 `config.yaml`。

### 6. 启动

```bash
sudo systemctl start tdy-manager
sudo systemctl status tdy-manager
curl -s http://127.0.0.1:8080/api/health
```

浏览器访问：`http://服务器IP:8080/`

首次启动会自动迁移表结构并（空库时）写入演示账号。**上线后立刻改掉 `admin/admin123` 密码。**

---

## 可选：Nginx 反代（80/443）

```bash
sudo cp /opt/tdy-manager/nginx.conf.example /etc/nginx/sites-available/tdy-manager
# 编辑 server_name
sudo ln -sf /etc/nginx/sites-available/tdy-manager /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

防火墙放行：

```bash
# firewalld 示例
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https
# 若不用 Nginx、直接暴露 8080：
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload
```

HTTPS 可用 certbot：`sudo certbot --nginx -d your-domain.com`

---

## 升级

本机重新 `deploy\pack.bat` → 上传覆盖 →：

```bash
sudo systemctl stop tdy-manager
sudo cp /tmp/tdy-manager/tdy-server /opt/tdy-manager/
sudo rm -rf /opt/tdy-manager/static && sudo cp -a /tmp/tdy-manager/static /opt/tdy-manager/
sudo chown -R tdy:tdy /opt/tdy-manager
sudo systemctl start tdy-manager
```

`config.yaml` 与 `uploads/` 不要覆盖。

---

## 常用运维

```bash
sudo journalctl -u tdy-manager -f     # 日志
sudo systemctl restart tdy-manager    # 重启
sudo systemctl stop tdy-manager       # 停止
```

健康检查：`GET /api/health`

---

## 架构说明

```
浏览器 → (可选 Nginx:80/443) → tdy-server:8080 → PostgreSQL
                              └─ 静态页(static/) + /api/*
```

服务器最低建议：1 核 1G、已装 PostgreSQL 16+（或 Docker 跑库）。
