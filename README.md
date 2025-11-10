# 德扑牛牛计分系统

> 双模式（德州扑克 / 牛牛）的实时计分、结算与后台管理一体化解决方案。

一个基于 **Go + Vue3** 的多人在线记分系统，覆盖建房、操作、结算到历史统计的完整流程。  
为了脱敏，界面中使用「单记分」指代德州扑克，「多记分」指代牛牛；踢出用户主要用于处理“有人离开,但没有主动退出”场景。

线上环境示例：`https://poker.iamwsll.cn`

---

## 功能亮点
- **实时房间管理**：六位房间号创建/加入、最近房间一键返回、成员进出记录、自动无人解散。
- **操作与异常修复**：德扑下注/收回/全收、牛牛多玩家下注、强制转账修正、房主踢人、解散房间。
- **结算与统计**：桌面积分守恒校验、自动生成结算路径、并查集聚合同桌玩家、今晚战绩统计。
- **后台管理**：账号管理、房间列表、盈亏明细、进出房间历史，支持自定义管理员账号。
- **实时同步与移动适配**：WebSocket 推送积分变化，前端对桌面/移动端双端适配。
- **可运维性**：配置化 CORS & Cookie 策略、数据库自动迁移、脚本化部署与测试报告。
## 功能演示
- 注册
![alt text](docs/image/1762708073619)
- 主页面
![alt text](docs/image/1762753611134)
- 创建房间
![alt text](docs/image/1762753637622)
- 邀请好友
![alt text](docs/image/1762753668591)
- 下注
![alt text](docs/image/1762753699451)
- 收回
![alt text](docs/image/1762753721979)
- 排行
![alt text](docs/image/1762753744141)
- 结算
![alt text](docs/image/1762753778598)
- 积分强制转移/踢出用户/解散房间
![alt text](docs/image/1762753837045)
- 个人信息
![alt text](docs/image/1762753859679)
- 历史战绩查询
![alt text](docs/image/1762753882736)
- 管理员后台管理
![alt text](docs/image/1762753909879)
更多功能请见api文档.

---

## 技术架构
### 前端
- Vue 3.5 + TypeScript 5.9（Composition API）
- Pinia 3、Vue Router 4.6、Ant Design Vue 4.2
- Axios 1.13、Vite 7.1、vite-plugin-vue-devtools

### 后端
- Go 1.25.4
- Gin 1.11、GORM 1.31（SQLite 驱动）
- gorilla/websocket、bcrypt（密码哈希）
- 模块化的 Service / Controller / Middleware 分层

### 其他
- SQLite3 持久化（默认 `backend/database.db`）
- Python + Requests 自动化接口测试
- Nginx 反向代理 + HTTPS 强制跳转（生产环境）

---

## 目录结构
```
poker_score/
├── backend/                    # Go 后端
│   ├── config/                 # 环境配置解析
│   ├── controllers/            # HTTP 控制器
│   ├── middlewares/            # 鉴权、CORS 等中间件
│   ├── models/                 # GORM 模型与数据库初始化
│   ├── services/               # 业务逻辑层
│   ├── utils/                  # 通用工具
│   ├── websocket/              # 房间 WebSocket Hub
│   ├── main.go                 # 入口
│   └── database.db             # 默认开发库
├── docs/                       # 文档中心
│   ├── api.md                  # 接口说明
│   ├── database.md             # 数据库设计
│   ├── deployment.md           # 生产部署手册
│   ├── tech-stack.md           # 技术说明
│   └── test_report.md          # 最近一次自动化测试报告
├── poker_score_frontend/       # Vue 前端
│   ├── src/                    # 页面、组件、API、状态
│   ├── public/                 # 静态资源
│   ├── dist/                   # 构建产物（示例）
│   └── package.json            # 前端依赖
├── test/                       # 自动化测试与辅助脚本
│   ├── test_api.py             # 后端接口回归脚本
│   ├── promote_admin.py        # 提升账号为管理员
│   └── requirements_test.txt   # 测试依赖
├── deploy.sh                   # 本地一键同步示例脚本
└── README.md
```

---

## 本地开发
### 1. 准备环境
- Go ≥ **1.25.4**
- Node.js ≥ **20.19**（建议与 `package.json` `engines` 一致）
- npm（或 pnpm / yarn）
- Python ≥ 3.10（用于运行自动化测试，可选）
- SQLite 随 `mattn/go-sqlite3` 驱动自动使用，无需额外安装

### 2. 安装依赖
```bash
# 后端
cd /Users/wsll/workspace/code/poker_score/backend
go mod download

# 前端
cd /Users/wsll/workspace/code/poker_score/poker_score_frontend
npm install
```

### 3. 启动服务
```bash
# 后端 (默认监听 http://localhost:8080)
cd /Users/wsll/workspace/code/poker_score/backend
go run .

# 前端 (默认监听 http://localhost:5173，已内置 /api 代理)
cd /Users/wsll/workspace/code/poker_score/poker_score_frontend
npm run dev
```

### 4. 访问与登录
- 浏览器打开 `http://localhost:5173`
- 默认管理员账号  
  - 手机号：`13800138000`  
  - 密码：`admin123`

---

## 配置说明（后端环境变量）
注意,详细的部署说明在 `docs/deployment.md` 中,这里仅仅摘要处理

| 变量名 | 默认值 | 说明 |
| --- | --- | --- |
| `APP_ENV` | `development` | 运行环境标识，生产请设置为 `production` |
| `SERVER_PORT` | `:8080` | Gin 监听端口，支持 `:port` 或 `80` 形式 |
| `SERVER_ALLOWED_ORIGINS` | `http://localhost:5173,...` | 允许的 CORS 来源，生产默认 `https://poker.iamwsll.cn` |
| `SERVER_COOKIE_DOMAIN` | 空 | Cookie Domain，生产默认 `poker.iamwsll.cn` |
| `SERVER_COOKIE_SECURE` | `false` (prod 默认 `true`) | 是否仅在 HTTPS 传输 Cookie |
| `SERVER_COOKIE_SAME_SITE` | `Lax` | Cookie SameSite 策略 |
| `DATABASE_PATH` | `./database.db` | SQLite 文件路径 |
| `DATABASE_MAX_IDLE_CONNS` | `10` | 数据库连接池最小空闲连接 |
| `DATABASE_MAX_OPEN_CONNS` | `100` | 数据库连接池最大连接数 |
| `DATABASE_CONN_MAX_LIFETIME` | `1h` | 连接最大生命周期 |
| `SESSION_COOKIE_NAME` | `poker_session` | 会话 Cookie 名称 |
| `SESSION_MAX_AGE` | `87600h` | Session 有效期（默认 10 年） |

生产环境配置示例与 systemd/Nginx 模板详见 `docs/deployment.md`。

---

## API 概览
- 认证：注册、登录、登出、获取个人信息、修改昵称/密码
- 房间：创建、加入、返回最近房间、离开/返回、踢人、解散
- 操作：德扑下注/收回、牛牛批量下注、强制转账、操作历史、历史额度
- 结算：发起结算、确认结算（积分守恒校验）
- 战绩：今晚战绩及时间段筛选
- 管理后台：用户、房间、结算记录、进出历史（需管理员角色）
- WebSocket：`/api/ws/room/:room_id` 房间实时广播

完整请求/响应示例请查看 `docs/api.md`。

---

## 自动化测试
1. 启动本地后端（必须）。
2. 安装依赖并运行：
   ```bash
   cd /Users/wsll/workspace/code/poker_score/test
   pip install -r requirements_test.txt
   python3 test_api.py
   ```
3. 测试结果会在终端输出，并生成 `docs/test_report.md`。

额外工具：`python3 test/promote_admin.py <phone>` 可将指定手机号用户提升为管理员（直接操作开发库）。

---

## 部署摘要
1. 参考 `docs/deployment.md` 准备服务器目录、环境变量与 systemd 服务。
2. 前端执行 `npm run build`，将 `poker_score_frontend/dist` 上传至服务器（可使用仓库内 `deploy.sh` 作为示例）。
3. 后端在服务器上构建 `go build -o poker_server` 并通过 systemd 启动。
4. Nginx 开启 HTTPS 并将 `/api`、`/api/ws` 反向代理到后端，启用 HTTP → HTTPS 重定向。

---

## 常见问题
- 务必检查 Go 版本、`DATABASE_PATH` 是否存在.
- 生产环境必须使用 HTTPS.
- 管理员权限问题：自己改个数据库得了

---

## 文档导航
- `docs/api.md`：完整 API 文档
- `docs/database.md`：表结构与关系
- `docs/deployment.md`：生产部署细节
- `docs/tech-stack.md`：技术选型说明
- `docs/test_report.md`：最近一次自动化测试结果

---

## License
MIT

## 联系方式
如有问题，欢迎提交 Issue 或 PR。