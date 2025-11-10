# Poker Score 部署指南

本文档介绍如何在单台服务器上部署 Poker Score 项目：前端通过 Nginx 提供静态资源并反向代理后端 API，后端 Go 服务与前端部署在同一台机器，域名为 `poker.iamwsll.cn`。部署环境为 1Panel，Nginx 运行在容器内，只能访问宿主机 `/opt/1panel/www/sites` 目录。

## 0. 环境总览与快速参考

### 本地开发环境（默认配置，无需额外改动）

- **后端**
  - 代码默认使用 `APP_ENV=development`，无需配置环境变量即可运行。
  - 数据库为 `backend/database.db`，与生产环境隔离。
  - GoLand 中直接运行 `main.go`（或使用 `go run .`）即可启动，监听 `:8080`。
- **前端**
  - 执行 `npm install` 后运行 `npm run dev`；WebStorm 可使用内置 npm 脚本一键启动。
  - Vite 代理已在 `vite.config.ts` 设置，自动将 `/api` 请求转发到 `http://localhost:8080`。
  - 无需设置 `VITE_API_BASE_URL` 或 `VITE_BACKEND_ORIGIN`，默认即为开发环境配置。
- **WebSocket/Cookie**
  - WebSocket 自动连接 `ws://localhost:8080`，会话 Cookie 域名留空，符合本地调试需求。
  - 本地数据库、会话与生产完全隔离，不应共用。

> 如需临时连接远程测试环境，可手动在本地创建 `.env.local` 配置相关地址，但不是常规流程。

### 生产环境（需显式配置）

- 依赖 1Panel 管理的 Nginx 和 TLS。
- 后端与前端均部署在 `/opt/1panel/www/sites/poker.iamwsll.cn` 下，详见后续章节。
- 必须设置环境变量以限定 CORS、Cookie 等安全参数。
- 前端构建产物通过 rsync 上传，或在服务器上独立构建。

## 1. 架构概览

- Nginx 监听 80/443 端口，提供 `poker.iamwsll.cn` 的静态页面，并将 `/api` 与 `/api/ws` 反向代理到后端服务。
- 后端 Go 应用监听本地 `:8080`，与 SQLite 数据库放置在 `/opt/1panel/www/sites/poker.iamwsll.cn/backend`。
- 前端使用 Vite 构建，产物上传至 `/opt/1panel/www/sites/poker.iamwsll.cn/dist`，通过 Nginx `try_files` 支持单页路由。

## 2. 环境准备

1. **系统组件**
   - Go ≥ 1.22（推荐使用与构建机一致版本）。
   - Node.js ≥ 20.19（参考 `poker_score_frontend/package.json` 中的 `engines`）。
   - npm（或 pnpm/yarn，文档示例使用 npm）。
   - Nginx（由 1Panel 管理，站点配置可通过 1Panel 面板或容器内 `/etc/nginx` 修改）。
   - SQLite（随 Go 的 `mattn/go-sqlite3` 驱动运行，无需单独安装）。
2. **服务器目录规划（示例）**
   - `/opt/1panel/www/sites/poker.iamwsll.cn/backend`：后端二进制、配置、日志、数据库。
   - `/opt/1panel/www/sites/poker.iamwsll.cn/dist`：前端静态文件（Nginx 容器可直接访问）。
   - `/opt/1panel/www/sites/poker.iamwsll.cn/ssl`：TLS 证书（若 1Panel 已生成证书，可沿用该位置）。

## 3. 后端部署

### 3.1 拉取与构建

> 本地开发机为 macOS arm64，而服务器为 Linux x86_64，且项目依赖 `mattn/go-sqlite3`（需要 CGO）。**推荐直接在服务器上编译**，避免交叉编译链路。

#### 方式 A：在服务器上编译（推荐）

```bash
# 将源码同步到服务器（仅首次或有更新时执行）
rsync -avz backend/ user@server:/opt/1panel/www/sites/poker.iamwsll.cn/backend-source/

# 登录服务器后执行
ssh user@server
cd /opt/1panel/www/sites/poker.iamwsll.cn/backend-source
echo 'export GOPROXY=https://goproxy.cn,direct' >> ~/.bashrc
go mod tidy
go build -o ../backend/poker_server

# 可选：同步数据库（仅首次部署需要）
cp ./database.db ../backend/database.db
```

以上命令会在 `/opt/1panel/www/sites/poker.iamwsll.cn/backend/poker_server` 生成可执行文件，并保留独立的源码目录 `backend-source` 便于后续更新。

#### 方式 B：本地交叉编译后上传（仅在已配置交叉编译环境时使用）

如需在 macOS 上直接编译 Linux/amd64 可执行文件，需要安装对应的交叉编译工具链（例如 `brew install SergioBenitez/osxct/x86_64-linux-gnu`）并指定 `CC`：

```bash
cd /Users/wsll/workspace/code/poker_score/backend
go mod tidy
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC=x86_64-linux-gnu-gcc go build -o poker_server

# 上传到服务器
scp poker_server user@server:/opt/1panel/www/sites/poker.iamwsll.cn/backend/
```

交叉编译对本地环境要求较高（需要有效的 Linux/amd64 C 编译器），若出现 CGO 相关报错，可退回到方式 A。

首次部署可连同仓库内的 `database.db` 上传；若需保留历史数据，请在服务器上单独备份数据库文件。

### 3.2 环境变量

后端通过环境变量区分开发/生产环境，核心变量如下。`poker.env` 需要手动创建，可通过以下方式生成：

```bash
ssh user@server
mkdir -p /opt/1panel/www/sites/poker.iamwsll.cn/backend
cat <<'EOF' >/opt/1panel/www/sites/poker.iamwsll.cn/backend/poker.env
APP_ENV=production
SERVER_PORT=:8080
SERVER_ALLOWED_ORIGINS=https://poker.iamwsll.cn
SERVER_COOKIE_DOMAIN=poker.iamwsll.cn
SERVER_COOKIE_SECURE=true
SERVER_COOKIE_SAME_SITE=Lax

# 数据库与 Session 配置可按需调整
DATABASE_PATH=/opt/1panel/www/sites/poker.iamwsll.cn/backend/database/poker_score.db
SESSION_COOKIE_NAME=poker_session
SESSION_MAX_AGE=87600h  # 10 年，保持与代码默认一致
EOF
```

必要时可使用 `nano`、`vim` 等编辑器进行修改。

```ini
APP_ENV=production
SERVER_PORT=:8080
SERVER_ALLOWED_ORIGINS=https://poker.iamwsll.cn,capacitor://localhost,https://localhost,http://localhost
SERVER_COOKIE_DOMAIN=poker.iamwsll.cn
SERVER_COOKIE_SECURE=true
SERVER_COOKIE_SAME_SITE=Lax

# 数据库与 Session 配置可按需调整
DATABASE_PATH=/opt/1panel/www/sites/poker.iamwsll.cn/backend/database/poker_score.db
SESSION_COOKIE_NAME=poker_session
SESSION_MAX_AGE=87600h  # 10 年，保持与代码默认一致
```

> 说明
>
> - `APP_ENV` 默认为 `development`，显式设为 `production` 可触发生产默认值。
> - `SERVER_ALLOWED_ORIGINS` 必须包含前端访问域名，否则浏览器会因 CORS 拒绝请求。
> - 若部署在同域名下，通过 `/api` 访问即可，无需额外跨域头部；该变量仍建议保留，以便未来拆分部署。
> - 若将数据库迁移到其他路径，请同步更新 `DATABASE_PATH` 并确保运行用户具备读写权限。
数据库的路径记得要创建.也就是 `/opt/1panel/www/sites/poker.iamwsll.cn/backend/database/` 目录.
### 3.3 systemd 服务示例

在服务器创建 `/etc/systemd/system/poker-score.service`：

```ini
[Unit]
Description=Poker Score Backend
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/1panel/www/sites/poker.iamwsll.cn/backend
ExecStart=/opt/1panel/www/sites/poker.iamwsll.cn/backend/poker_server
EnvironmentFile=/opt/1panel/www/sites/poker.iamwsll.cn/backend/poker.env
Restart=on-failure
RestartSec=5
User=root
Group=www-data

[Install]
WantedBy=multi-user.target
```

> 若使用其他用户运行，请同步调整文件/目录权限。

启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now poker-score.service
sudo systemctl status poker-score.service
```

确保 `:8080` 仅对本机开放（例如使用 `ufw` 或 `firewalld` 拒绝外部访问），以免暴露后端端口。

## 4. 前端构建与部署

### 4.1 本地构建

```bash
cd /Users/wsll/workspace/code/poker_score/poker_score_frontend
npm install
npm run build
```

构建输出位于 `poker_score_frontend/dist`。

### 4.2 上传与发布

```bash
rsync -avz dist/ user@server:/opt/1panel/www/sites/poker.iamwsll.cn/dist/
```

如需在服务器上直接构建，可在服务器执行（先在目标目录创建 `frontend-source` 或任意自定义目录）：

```bash
mkdir -p /opt/1panel/www/sites/poker.iamwsll.cn/frontend-source
cd /opt/1panel/www/sites/poker.iamwsll.cn/frontend-source
npm install
npm install --production=false
npm run build
```

根据需要创建 `npmrc` 或使用 `pnpm` 来加速安装。

### 4.3 Vite 环境变量

前端新增了 `VITE_API_BASE_URL` 与 `VITE_BACKEND_ORIGIN` 两个变量：

- 默认无需创建额外文件，未设置时 `axios` 会将 `/api` 视为基准路径，并依赖 Nginx 反向代理。
- 若希望在开发环境直接指向远程后端，可在本地创建 `poker_score_frontend/.env.local`：

  ```ini
  VITE_API_BASE_URL=https://poker.iamwsll.cn/api
  VITE_BACKEND_ORIGIN=https://poker.iamwsll.cn
  ```

- WebSocket 默认在开发模式直连 `http://localhost:8080`，生产环境使用当前站点域名。

## 5. Nginx 配置

在不影响现有 `code.iamwsll.cn` 的前提下，为 `poker.iamwsll.cn` 新增 server 块（1Panel 通常会在容器内 `/etc/nginx/conf.d/` 创建配置文件，下例以 `poker.iamwsll.cn.conf` 为例）：

> **⚠️ 重要**：必须配置HTTP到HTTPS的强制重定向！
> 
> 后端设置了 `Secure` Cookie标志，只能在HTTPS下工作。如果用户通过HTTP访问，浏览器会拒绝设置Cookie，导致无法登录。

```nginx

map $http_upgrade $connection_upgrade {
  default upgrade;
  ''      close;
}

# HTTP server - 强制重定向到HTTPS
server {
  listen 80;
  server_name poker.iamwsll.cn;

  # 强制重定向所有HTTP请求到HTTPS
  return 301 https://$server_name$request_uri;
}

# HTTPS server - 主服务
server {
  listen 443 ssl http2;
  server_name poker.iamwsll.cn;

  root /www/sites/poker.iamwsll.cn/index;
  index index.html;

  ssl_certificate     /www/sites/poker.iamwsll.cn/ssl/fullchain.pem;
  ssl_certificate_key /www/sites/poker.iamwsll.cn/ssl/privkey.pem;
  ssl_protocols TLSv1.3 TLSv1.2;
  ssl_ciphers ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-SHA384:ECDHE-RSA-AES128-SHA256:!aNULL:!eNULL:!EXPORT:!DSS:!DES:!RC4:!3DES:!MD5:!PSK:!KRB5:!SRP:!CAMELLIA:!SEED;
  ssl_session_cache shared:SSL:10m;
  ssl_session_timeout 10m;
  add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

  # 静态文件 & SPA 路由
  location / {
    try_files $uri $uri/ /index.html =404;
  }

  # 反向代理后端 API
  location /api/ {
    proxy_pass http://127.0.0.1:8080/api/;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection $connection_upgrade;
  }

  # WebSocket (保持与 /api/ws 前缀一致)
  location /api/ws/ {
    proxy_pass http://127.0.0.1:8080/api/ws/;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
  }

  error_page 497 https://$host$request_uri;
  access_log /www/sites/poker.iamwsll.cn/log/access.log main;
  error_log  /www/sites/poker.iamwsll.cn/log/error.log;
}
```

> 若证书尚未申请，可参考 `certbot` 或使用已有的自动化方式；务必为新域名签发有效证书。

如果通过命令行管理容器化 Nginx，可执行：

```bash
sudo docker exec -it <nginx_container_name> nginx -t
sudo docker exec -it <nginx_container_name> nginx -s reload
```

或者使用 1Panel 面板提供的「校验配置并重载」功能。

## 6. 验证与运维

1. 访问 `https://poker.iamwsll.cn`，确认页面能够加载并通过浏览器网络面板验证 `https://poker.iamwsll.cn/api/ping` 返回 `pong`。
2. 登录后检查 Cookie：应包含 `poker_session`，`Secure` 标记为 `true`，`Domain` 为 `poker.iamwsll.cn`。
3. 在房间页面发起操作，确认 WebSocket 连接 `wss://poker.iamwsll.cn/api/ws/room/<id>` 成功。
4. 后端日志可在 `journalctl -u poker-score.service` 或自定义日志文件中查看，数据库文件建议定期备份。

## 7. 开发/生产快速切换

- **后端**：通过调整 `APP_ENV` 与相关环境变量即可切换；开发环境默认允许 `http://localhost:5173` 并关闭 HTTPS Cookie。
- **前端**：`npm run dev` 自动读取 `import.meta.env`，默认通过 Vite 代理访问 `localhost:8080`；若需指向远程环境，可设置 `VITE_API_BASE_URL` 与 `VITE_BACKEND_ORIGIN`。
- **Nginx**：开发阶段可使用本地代理（如 `vite.config.ts` 中的代理配置），生产环境按上述 server 块部署。

## 8. Capacitor 原生应用打包说明

将前端通过 Capacitor 封装为 Android/iOS 原生应用时，需要额外配置网络访问：
0. 环境准备:
```bash
# 在 Vue 项目中安装 Capacitor 依赖
npm install @capacitor/core @capacitor/cli

# 初始化 Capacitor
npx cap init

# 安装 Android 平台支持
npm install @capacitor/android

# 将 Android 平台“添加”到你的项目中
npx cap add android

```
1. **前端环境变量**
   - 参考`.env.native` ,这里配置了必要的环境变量.
   - 并通过 `npm run build-only -- --mode native` 进行构建.
   - 进行 `npx cap sync android`。
   - 进行 `npx cap open android`。

2. **后端 CORS 设置**
   - 这个设置已经在生产环境后端的.poker.env里配置了.(server_allowed_origins)
   - 变更配置后需重启后端服务。

3. **Android 端其他注意事项**
   - 调试网络请求,在chrome里打开chrome://inspect/#devices,启用Discover USB devices,找到手机设备,点击inspect.
4. 打包.

按照以上配置，打包后的原生应用即可正常访问生产后端并与 Web 端保持一致行为。

按照以上步骤即可在不影响已部署站点的情况下新增 `poker.iamwsll.cn` 并完成前后端部署。

