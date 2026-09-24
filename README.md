# TDY 派单管理

管理员发单指定用户 → 接单提交 → 验收入账 → 钱包/提现/对账。

## 技术栈

Go + Gin + GORM + PostgreSQL · Vue3 + Element Plus

## 一键启动（推荐）

Windows：双击 `start.bat`（会编译前端+后端，健康检查通过后打开浏览器）

Linux：

```bash
chmod +x start.sh stop.sh
./start.sh
```

访问：**http://127.0.0.1:8080/**（前后端一体）

| 账号 | 密码 | 说明 |
|------|------|------|
| admin | admin123 | 超级管理员（全部权限） |
| zhangsan | pass1234 | 普通用户 |
| lisi | pass1234 | 普通用户 |

> 密码规则：至少 6 位，且同时包含字母和数字。若库是早期种子数据，请用管理端重置用户密码。

## 部署到服务器

见 **[deploy/DEPLOY.md](deploy/DEPLOY.md)**。

Windows 本机一键打 Linux 包：

```bat
deploy\pack.bat
```

上传 `dist-release\tdy-manager` 到服务器后执行 `sudo bash install.sh`，编辑 `/opt/tdy-manager/config.yaml` 后 `systemctl start tdy-manager`。

## 配置

编辑 `backend/config.yaml`（参考 `config.example.yaml`）：

- `server.mode`: `release` 生产模式
- `server.jwt_secret`: 务必改成强随机串
- `server.cors_origins`: 前端来源白名单
- `business.withdraw_max_yuan` / `withdraw_daily_max_yuan`: 提现限额

## 主要能力

- 派单 / 批量派单 / 改派 / 截止与超期
- 附件上传、验收入账（幂等）
- 钱包、提现（单笔/日限额、批量审核）
- 对账 + CSV 导出
- 仪表盘、站内通知、操作审计
- 多管理员 + 细粒度权限（内置超管/运营/财务/审计，可自定义角色）
- 登录限流、用户禁用、改密
- `/api/health` 健康检查；Schema 版本迁移跳过

## 开发模式（热更新）

```bash
# 终端1
cd backend && go run .

# 终端2
cd frontend && npm run dev
```

前端：http://127.0.0.1:5173
