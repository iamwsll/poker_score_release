# 德扑牛牛计分系统

一个基于Vue3+Go的实时多人德州扑克/牛牛计分应用，适用于朋友聚会时的游戏计分。

## 技术栈

### 前端
- Vue 3.4+ (Composition API)
- TypeScript 5.0+
- Pinia 2.0+ (状态管理)
- Vue Router 4.0+
- Ant Design Vue 4.0+ (UI组件库)
- Vite 5.0+ (构建工具)
- Axios (HTTP客户端)
- WebSocket (实时通信)

### 后端
- Go 1.21+
- Gin 1.9+ (Web框架)
- Gorm 1.25+ (ORM)
- SQLite 3 (数据库)
- gorilla/websocket (WebSocket)
- bcrypt (密码加密)

## 功能特性

- ✅ 用户注册/登录系统
- ✅ 创建德扑/牛牛房间
- ✅ 6位房间号快速加入
- ✅ 实时积分计算
- ✅ WebSocket实时同步
- ✅ 结算方案自动生成
- ✅ 今晚战绩统计（并查集算法）
- ✅ 后台管理系统
- ✅ 移动端适配

## 快速开始

### 环境要求

- Node.js 18+ 
- Go 1.21+
- npm或yarn

### 安装依赖

#### 后端

```bash
cd backend
go mod download
```

#### 前端

```bash
cd poker_score_frontend
npm install
```

### 运行开发环境

#### 1. 启动后端服务

```bash
cd backend
go run main.go
```

后端将运行在 `http://localhost:8080`

#### 2. 启动前端开发服务器

```bash
cd poker_score_frontend
npm run dev
```

前端将运行在 `http://localhost:5173`

#### 3. 访问应用

打开浏览器访问：`http://localhost:5173`

**默认管理员账户：**
- 手机号：`13800138000`
- 密码：`admin123`

## 项目结构

```
poker_score/
├── backend/                 # 后端Go项目
│   ├── config/             # 配置
│   ├── models/             # 数据模型
│   ├── controllers/        # 控制器
│   ├── services/           # 业务逻辑
│   ├── middlewares/        # 中间件
│   ├── websocket/          # WebSocket处理
│   ├── utils/              # 工具函数
│   ├── main.go             # 入口文件
│   └── database.db         # SQLite数据库
├── poker_score_frontend/   # 前端Vue项目
│   ├── src/
│   │   ├── api/            # API请求封装
│   │   ├── stores/         # Pinia状态管理
│   │   ├── router/         # 路由配置
│   │   ├── views/          # 页面组件
│   │   ├── components/     # 公共组件
│   │   ├── utils/          # 工具函数
│   │   └── types/          # TypeScript类型
│   └── vite.config.ts      # Vite配置
└── docs/                    # 文档
    ├── database.md         # 数据库设计
    ├── api.md              # API接口文档
    └── tech-stack.md       # 技术栈说明
```

## 使用指南

### 1. 注册账户

首次使用需要注册账户，填写手机号、昵称和密码。

### 2. 创建房间

- 点击"创建房间"按钮
- 选择房间类型（德扑/牛牛）
- 设置积分与人民币比例（德扑默认20:1，牛牛默认1:1）
- 系统自动生成6位房间号

### 3. 加入房间

- 点击"加入房间"按钮
- 输入6位房间号
- 进入房间开始游戏

### 4. 游戏操作

**德州扑克：**
- 支出：下注积分
- 收回：收回已下注的积分
- 全收：一键收回所有积分

**牛牛：**
- 下注：可以给房间内任何人（包括自己）下注
- 收回：收回已下注的积分

### 5. 结算

- 点击"积分排行"查看当前积分情况
- 当桌面积分为0时，点击"结算这局"
- 系统自动生成结算方案（负积分→最高正积分→其他正积分）
- 发起者点击"确认结算"完成结算
- 结算后所有积分清零

### 6. 查看战绩

- 点击"今晚战绩"查看盈亏情况
- 系统自动计算"今晚一起玩过的好友"（并查集算法）
- 显示时间段内所有相关玩家的盈亏

### 7. 后台管理（仅管理员）

- 查看所有用户
- 查看所有房间
- 查看用户历史盈亏
- 查看用户进出房间历史

## API文档

详见 `docs/api.md`

主要接口：
- `POST /api/auth/register` - 用户注册
- `POST /api/auth/login` - 用户登录
- `POST /api/rooms` - 创建房间
- `POST /api/rooms/join` - 加入房间
- `POST /api/rooms/:id/bet` - 下注
- `POST /api/rooms/:id/withdraw` - 收回
- `POST /api/rooms/:id/settlement/initiate` - 发起结算
- `POST /api/rooms/:id/settlement/confirm` - 确认结算
- `GET /api/records/tonight` - 查询今晚战绩
- `WS /api/ws/room/:id` - WebSocket实时通信

## 数据库设计

详见 `docs/database.md`

主要表：
- `users` - 用户表
- `sessions` - Session表
- `rooms` - 房间表
- `room_members` - 房间成员表
- `user_balances` - 用户积分余额表
- `room_operations` - 房间操作记录表
- `settlements` - 结算记录表
- `bet_records` - 牛牛下注记录表

## 核心算法

### 1. 积分守恒原则

房间内所有用户的积分总和始终为0：
```
Σ(用户积分) = 0
```

### 2. 桌面积分计算

桌面积分 = 所有用户负积分的绝对值之和
```
桌面积分 = Σ|min(用户积分, 0)|
```

### 3. 结算算法

结算方案生成规则：
1. 所有负积分的人向正积分最高的人转账
2. 正积分最高的人给其他正积分的人转账
3. 转账金额按照积分与人民币比例计算

### 4. 并查集战绩统计

使用BFS算法查找"今晚一起玩过的好友"：
1. 从用户加入的房间开始
2. 查找房间内的所有成员
3. 递归查找这些成员加入的其他房间
4. 直到遍历完所有相关用户

## 生产环境部署

### 后端部署

```bash
# 编译
cd backend
go build -o poker_score_server main.go

# 运行
./poker_score_server
```

### 前端部署

```bash
# 构建
cd poker_score_frontend
npm run build

# 生成的dist目录包含静态文件
```

### Nginx配置示例

```nginx
server {
    listen 80;
    server_name your-domain.com;

    # 前端静态文件
    location / {
        root /path/to/poker_score_frontend/dist;
        try_files $uri $uri/ /index.html;
    }

    # 后端API
    location /api {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 开发注意事项

1. **Session管理**：Session存储在SQLite中，有效期30天
2. **WebSocket**：用户离开房间时会自动断开连接，房间无人时自动解散
3. **移动端适配**：使用viewport设置和Ant Design Vue的响应式组件
4. **CORS配置**：开发环境已配置CORS，生产环境使用Nginx反向代理
5. **错误处理**：前端统一拦截401错误，自动跳转登录页
6. **数据库事务**：涉及积分变动的操作都使用事务确保原子性

## 故障排查

### 后端无法启动

1. 检查8080端口是否被占用
2. 检查Go版本是否>=1.21
3. 检查依赖是否完整安装

### 前端无法连接后端

1. 检查后端是否正常运行
2. 检查Vite proxy配置是否正确
3. 检查浏览器控制台是否有CORS错误

### WebSocket连接失败

1. 检查用户是否已登录
2. 检查用户是否在房间中
3. 检查浏览器是否支持WebSocket
4. 检查网络连接状态

## License

MIT

## 联系方式

如有问题，请提交Issue。

