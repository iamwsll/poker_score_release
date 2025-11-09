# 技术栈说明文档

## 项目概述
德州扑克 / 牛牛计分系统，通过 Web 前端 + Go 后端提供实时房间积分管理，支持 REST 接口与 WebSocket 广播。

---

## 前端技术栈

### 核心框架
- **Vue 3.5.x**：使用 Composition API 与 `<script setup>` 驱动页面
- **TypeScript 5.9**：开启严格模式，配合 `vue-tsc` 做类型检查
- **Vite 7**：开发调试与生产构建工具

### 状态与路由
- **Pinia 3.0**：集中管理用户、房间等全局状态
- **Vue Router 4.6**：基于 history 的单页应用路由

### UI 与工具
- **Ant Design Vue 4.2**：按需引入的组件库
- **Axios 1.13**：HTTP 请求封装，默认 `baseURL=/api` 且 `withCredentials: true`
- **Day.js 1.11**：日期与时间计算
- 原生 **WebSocket API**：订阅房间推送

### 代码规范
- **ESLint 9 + eslint-plugin-vue 10.5**：统一代码质量
- **Prettier 3.6**：格式化规则
- **TypeScript 严格模式**：杜绝隐式 any 与类型不匹配

---

## 后端技术栈

### 核心框架
- **Go 1.25**：使用标准库 goroutine 处理并发
- **Gin 1.11**：REST 框架，负责路由、中间件与验证

### 数据存储
- **SQLite**：嵌入式数据库，位于 `backend/database.db`
- **Gorm 1.31 + gorm.io/driver/sqlite 1.6**：ORM，负责 AutoMigrate、事务、链式查询

### 实时能力
- **gorilla/websocket 1.5.3**：WebSocket 协议实现；通过自建 Hub 广播房间事件

### 其它依赖
- **golang.org/x/crypto/bcrypt**：密码哈希
- **github.com/google/uuid 1.6**：Session ID、结算批次号
- **标准库 log**：统一日志输出

---

## 通信协议

### REST API
- 统一前缀：`/api`
- 认证：基于 Cookie 的 Session（Cookie 名 `poker_session`，默认有效期 10 年）
- CORS：仅允许 `http://localhost:5173`、`http://localhost:3000`、`http://localhost:80`，必须携带凭证
- 响应结构：`code` / `message` / `data`

### WebSocket
- 地址：`ws://localhost:8080/api/ws/room/:room_id`
- 认证：复用 REST Session Cookie
- 心跳：服务端每 ~54s 发送 Ping 帧；客户端可定期发送 `{"type":"ping"}`
- 断线：自动将成员状态标记为 `offline`

---

## 开发环境

### 前端
- `npm install` → `npm run dev`（默认端口 5173）
- 通过 Vite 代理将 `/api` 转发至 `http://localhost:8080`
- Axios 默认启用 `withCredentials`，确保浏览器允许 Cookie

### 后端
- `go run main.go`
- 配置文件：`config/config.go`（端口 / CORS / Session 等）
- 首次启动自动迁移表结构并创建默认管理员账号

### 常用工具
- Chrome DevTools：网络与移动端调试
- Postman / HTTPie：接口调试
- WebSocket 客户端：现场测试广播与心跳

---

## 部署参考

### 前端
```bash
cd poker_score_frontend
npm ci
npm run build  # 产物位于 dist/
```

### 后端
```bash
cd backend
go mod download
go build -o poker_score_server main.go
./poker_score_server
```

### Nginx 反向代理
```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        root /path/to/poker_score_frontend/dist;
        try_files $uri $uri/ /index.html;
    }

    location /api {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

---

## 项目结构概览

### 前端
```
poker_score_frontend/
├── src/
│   ├── api/         # Axios 封装
│   ├── stores/      # Pinia 仓库
│   ├── views/       # 页面组件
│   ├── utils/       # 工具函数
│   └── types/       # TS 类型定义
├── public/
├── vite.config.ts
├── package.json
└── tsconfig*.json
```

### 后端
```
backend/
├── main.go
├── config/
├── controllers/
├── services/
├── models/
├── middlewares/
├── websocket/
├── utils/
└── database.db
```

---

## 性能与可靠性

### 前端优化
1. Ant Design Vue 按需加载，减少 bundle 体积
2. 路由懒加载降低首屏加载压力
3. WebSocket 断线重连由前端自行兜底

### 后端优化
1. 积分变更统一封装在数据库事务中，避免并发竞争
2. 背景协程每小时扫描，12 小时无操作的房间自动 `dissolved`
3. Hub 负责房间广播，避免重复遍历所有连接
4. SQLite 连接池配置：`MaxIdle=10`、`MaxOpen=100`、`ConnMaxLifetime=1h`

---

## 安全策略

### 前端
1. 默认模板转义，避免 XSS；禁止使用 `v-html`
2. 所有请求依赖 SameSite=Lax Cookie 抵抗 CSRF
3. 不在本地存储敏感信息，完全依赖服务器 Session

### 后端
1. 密码使用 bcrypt 哈希
2. 认证、管理员校验通过 Gin 中间件完成
3. CORS 白名单控制来源，统一返回 401/403/404
4. 日志记录关键操作与异常

---

## 测试与质量
- `test_api.py` 自动化脚本覆盖主要 REST 流程（注册、房间、结算、管理员等）
- 运行结果示例见 `docs/test_report.md`
- 建议在功能改动后重新跑一次回归脚本

---

## 监控建议
- 关注在线用户数、活跃房间数、WebSocket 连接数
- 采集 API 响应时间、错误码分布
- 建议将 `log.Printf` 输出接入集中日志系统

---

## 扩展路线
1. 将 Session / 房间状态缓存迁移至 Redis
2. 将 SQLite 升级为 PostgreSQL / MySQL 支撑多实例
3. 借助 Redis Pub/Sub 或 MQ 做跨实例 WebSocket 广播
4. 将后台管理接口拆分为独立服务，提升安全隔离

---

## 依赖版本

### 前端 `package.json`
```json
{
  "ant-design-vue": "^4.2.6",
  "axios": "^1.13.2",
  "dayjs": "^1.11.19",
  "pinia": "^3.0.3",
  "vue": "^3.5.22",
  "vue-router": "^4.6.3"
}
```

### 后端 `go.mod`
```go
require (
    github.com/gin-gonic/gin v1.11.0
    gorm.io/gorm v1.31.1
    gorm.io/driver/sqlite v1.6.0
    github.com/gorilla/websocket v1.5.3
    github.com/google/uuid v1.6.0
    golang.org/x/crypto v0.43.0
)
```

---

## 开发约定
1. 注释优先中文，说明业务背景与边界
2. 前端组件文件使用大驼峰命名，逻辑变量小驼峰；后端导出符号使用大驼峰
3. Git 提交遵循语义化（feat/fix/docs/refactor...）
4. 所有接口统一返回结构体并在 `controllers` 层封装响应
5. 时间统一使用 UTC 存储，前端视图再做本地化转换

---

## 常见问题
- **为什么选择 SQLite？** 系统规模较小，嵌入式数据库部署简单，性能充足。
- **Session 为什么不用 JWT？** 需要服务端主动失效能力，Cookie Session 更容易管理。
- **房间何时解散？** 默认 12 小时没有新操作会被后台协程自动设为 `dissolved`。
- **如何保证积分正确？** 所有积分操作都在事务内完成，结算前会校验桌面余额必须为 0。

---

## 更新历史
- **2025-11-07**：同步代码版本（Go 1.25 / Vue 3.5 / Vite 7），补充 CORS 与自动解散机制说明
