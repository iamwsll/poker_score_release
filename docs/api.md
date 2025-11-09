# API 接口文档

更新日期：2025-11-07

## 基础信息

- Base URL：`http://localhost:8080/api`
- 数据格式：JSON，编码 UTF-8
- 认证方式：基于 Cookie 的会话（默认 Cookie 名为 `poker_session`，有效期 10 年，`HttpOnly` + `SameSite=Lax`）
- CORS：默认仅允许 `http://localhost:5173`、`http://localhost:3000`、`http://localhost:80`，且必须携带凭证（`withCredentials: true`）

## 通用响应结构

### 成功
```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

### 失败
```json
{
  "code": 400,
  "message": "错误信息",
  "data": null
}
```

### 错误码约定

- `0`：成功
- `400`：参数错误或业务校验失败（手机号已注册、桌面积分不足等）
- `401`：未登录或 Session 已过期
- `403`：权限不足（需要管理员权限）
- `404`：资源不存在（房间不存在、历史记录为空等）
- `500`：服务器内部错误

## 1. 认证模块

| 接口 | 方法 | 说明 |
| ---- | ---- | ---- |
| `/auth/register` | POST | 注册并自动登录 |
| `/auth/login` | POST | 登录并刷新 Session |
| `/auth/logout` | POST | 登出，清除 Session |
| `/auth/me` | GET | 获取当前登录用户信息 |
| `/auth/nickname` | PUT | 修改昵称 |
| `/auth/password` | PUT | 修改密码 |

### 1.1 注册 `POST /api/auth/register`

请求体：
```json
{
  "phone": "13800138000",
  "nickname": "张三",
  "password": "123456"
}
```

成功响应：
```json
{
  "code": 0,
  "message": "注册成功",
  "data": {
    "user": {
      "id": 16,
      "phone": "13800138000",
      "nickname": "张三",
      "role": "user",
      "created_at": "2025-11-07T05:52:23.920808Z"
    },
    "session_id": "2cec5947-5b00-4c2b-8b37-6a5668f3fd14"
  }
}
```

服务器会同时下发 `Set-Cookie: poker_session=<session_id>; HttpOnly; SameSite=Lax`。手机号重复时返回 `400`，错误信息为“手机号已注册”。

### 1.2 登录 `POST /api/auth/login`

请求体：
```json
{
  "phone": "13800138000",
  "password": "123456"
}
```

成功响应与注册相同（`message` 为“登录成功”）。手机号或密码错误时返回 `400`，错误信息统一为“手机号或密码错误”。

### 1.3 登出 `POST /api/auth/logout`

需要认证。成功后删除 Session 并设置过期 Cookie：
```json
{
  "code": 0,
  "message": "登出成功",
  "data": null
}
```

### 1.4 当前用户 `GET /api/auth/me`

返回登录用户基本信息：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 16,
    "phone": "13862494743",
    "nickname": "测试用户1",
    "role": "user",
    "created_at": "2025-11-07T05:52:23.920808Z"
  }
}
```

### 1.5 修改昵称 `PUT /api/auth/nickname`

请求体：
```json
{
  "nickname": "新昵称测试"
}
```

成功：
```json
{
  "code": 0,
  "message": "修改成功",
  "data": {
    "nickname": "新昵称测试"
  }
}
```

### 1.6 修改密码 `PUT /api/auth/password`

请求体：
```json
{
  "old_password": "123456",
  "new_password": "654321"
}
```

成功时 `message` 为“密码修改成功”，`data` 为 `null`。旧密码错误会得到 `400` 与提示“旧密码错误”。

## 2. 房间管理

| 接口 | 方法 | 说明 |
| ---- | ---- | ---- |
| `/rooms` | POST | 创建房间（创建者会自动加入） |
| `/rooms/join` | POST | 通过 6 位房间号加入房间 |
| `/rooms/last` | GET | 返回用户最近一次加入且仍为 `active` 的房间 |
| `/rooms/:room_id` | GET | 获取房间详情（要求当前仍是成员） |
| `/rooms/:room_id/leave` | POST | 将自己状态标记为离线 |
| `/rooms/:room_id/kick` | POST | 将某成员标记为 `offline` 并广播踢出事件 |

通用返回结构：
```json
{
  "room_id": 7,
  "room_code": "941425",
  "room_type": "texas",
  "chip_rate": "20:1",
  "status": "active",
  "table_balance": 0,
  "my_balance": 0,
  "members": [
    {
      "user_id": 16,
      "nickname": "测试用户1",
      "balance": 0,
      "status": "online"
    },
    {
      "user_id": 18,
      "nickname": "测试用户3",
      "balance": 0,
      "status": "online"
    }
  ]
}
```

- `room_type` 仅允许 `texas` 或 `niuniu`
- `chip_rate` 是“积分:人民币”的字符串，如 `20:1`
- `members[].status` 可能为 `online`、`offline`
- 离线或被踢出的成员仍然留在房间列表中，`left_at` 字段目前不会被写入
- `LeaveRoom` 与 `KickUser` 只改变状态，不会删除 `room_members` 记录
- 任何成员都可以调用踢人接口，服务端未限制房主

创建房间：
```json
{
  "code": 0,
  "message": "房间创建成功",
  "data": {
    "room_id": 7,
    "room_code": "941425",
    "room_type": "texas",
    "chip_rate": "20:1",
    "created_at": "2025-11-07T05:52:24.168482Z"
  }
}
```

加入房间失败时会返回 `400`，常见错误信息有“房间不存在或已解散”“您不在该房间中”。

## 3. 房间操作

| 接口 | 方法 | 说明 |
| ---- | ---- | ---- |
| `/rooms/:id/bet` | POST | 德扑支出，`amount` 必须为正整数 |
| `/rooms/:id/withdraw` | POST | 收回积分，`amount <= 0` 表示“全收” |
| `/rooms/:id/force-transfer` | POST | 将桌面所有积分强制转移给指定成员 |
| `/rooms/:id/niuniu-bet` | POST | 牛牛下注，批量给多人下注 |
| `/rooms/:id/operations` | GET | 获取房间操作历史（只包含用户本次加入后的记录） |
| `/rooms/:id/history-amounts` | GET | 最近 6 条下注/收回的快捷金额 |

### 3.1 德扑下注

请求体：`{"amount": 100}`

成功：
```json
{
  "code": 0,
  "message": "下注成功",
  "data": {
    "my_balance": -100,
    "table_balance": 100
  }
}
```

余额更新与操作记录写入在同一事务内完成。失败时返回 `400` 并附带原因。

### 3.2 收回积分

请求体：`{"amount": 0}`（0 或负数表示收回桌面全部可用积分）。

返回示例：
```json
{
  "code": 0,
  "message": "收回成功",
  "data": {
    "my_balance": 0,
    "table_balance": 0,
    "actual_amount": 150
  }
}
```

若桌面没有积分或请求金额超出桌面余额，返回 `400`。

### 3.3 牛牛下注

请求体：
```json
{
  "bets": [
    { "to_user_id": 18, "amount": 50 }
  ]
}
```

服务端会将 `amount` 累加存入操作记录并写入 `bet_records` 表。返回：
```json
{
  "code": 0,
  "message": "下注成功",
  "data": {
    "my_balance": -50,
    "total_amount": 50
  }
}
```

### 3.4 积分强制转移

请求体：`{"target_user_id": 17}`。

调用方需要同时满足：

- 仍在房间中（`room_members.left_at IS NULL`）
- 当前桌面存在可转移的积分
- 目标用户仍在房间中

成功示例：
```json
{
  "code": 0,
  "message": "积分已转移",
  "data": {
    "table_balance": 0,
    "transferred_amount": 320,
    "target_user_id": 17,
    "target_balance": 680,
    "actor_user_id": 16,
    "actor_balance": 0
  }
}
```

若桌面没有积分或目标用户离开房间，会返回 `400` 并附带错误原因。

### 3.5 操作历史

请求：`GET /api/rooms/7/operations?limit=10&offset=0`

响应中的每条操作都包含：

- `operation_type`：`join` / `leave` / `bet` / `withdraw` / `force_transfer` / `kick` / `niuniu_bet` / `settlement_initiated` / `settlement_confirmed`
- `amount`：仅在下注、收回、牛牛下注等涉及积分时存在
- `description`：大部分操作是中文描述，牛牛下注会写入 JSON 字符串
- `target_user_id` 与 `target_nickname`：存在于踢人与积分强制转移操作

### 3.6 快捷金额

`GET /api/rooms/:room_id/history-amounts`

返回最近 6 条下注/收回金额（按时间倒序），用于前端快捷按钮：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "bet_amounts": [200, 100],
    "withdraw_amounts": [150]
  }
}
```

## 4. 结算

| 接口 | 方法 | 说明 |
| ---- | ---- | ---- |
| `/rooms/:id/settlement/initiate` | POST | 校验桌面积分并生成结算方案 |
| `/rooms/:id/settlement/confirm` | POST | 生成 settlement 记录并清空余额 |

### 4.1 发起结算

只有当 `table_balance` 为 0 时才会成功：
```json
{
  "code": 0,
  "message": "结算方案已生成",
  "data": {
    "can_settle": true,
    "table_balance": 0,
    "settlement_plan": [
      {
        "from_user_id": 18,
        "from_nickname": "测试用户3",
        "to_user_id": 16,
        "to_nickname": "测试用户1",
        "chip_amount": 200,
        "rmb_amount": 10,
        "description": "测试用户3 → 测试用户1 200积分（¥10.00）"
      }
    ]
  }
}
```

桌面积分不为 0 时返回 `400`，同时在 `data.table_balance` 中回传当前桌面值。

### 4.2 确认结算

成功后会写入 `settlements` 表、清空 `user_balances`，并广播 WebSocket 消息：
```json
{
  "code": 0,
  "message": "结算完成",
  "data": {
    "settlement_batch": "fd13a3d8-5cbe-4c91-8358-723b01344b59",
    "settled_at": "2025-11-07T05:52:50.390222Z"
  }
}
```

## 5. 战绩统计

`GET /api/records/tonight`

- 若不传时间参数，服务端自动计算“今天 7:00 到明天 7:00”（早于 7:00 则取昨日 7:00 至今日 7:00）
- 采用 BFS 扩散查找在时间段内与我同桌的所有好友，再统计结算记录
- `current_rooms`：当前仍在线的房间列表（`room_members.left_at` 为空且房间状态为 `active`）
- `friends_records`：包含 `user_id`、`nickname`、`total_chip`、`total_rmb`、`is_me`
- `total_check`：所有好友人民币盈亏求和（用于校验是否为 0，可能出现浮点误差）

## 6. 后台接口

所有 `/api/admin/**` 路径都需要管理员账号（`user.role == "admin"`）。

- `/admin/users`：分页返回所有用户，结构与 `models.User` 对应，包含 `updated_at`
- `/admin/rooms`：分页返回房间列表，附带 `member_count`、`online_count`
- `/admin/rooms/:room_id`：返回房间详情、成员列表（按 `joined_at DESC`）以及最近 100 条操作
- `/admin/users/:user_id/settlements`：按照时间范围过滤结算记录，并汇总 `total_chip` 和 `total_rmb`
- `/admin/room-member-history`：支持 `user_id`、`room_id` 过滤，`duration_minutes` 只有当 `left_at` 不为空时才会计算

## 7. WebSocket

- URL：`ws://localhost:8080/api/ws/room/:room_id`
- 认证：与 REST 接口相同，通过 Cookie 验证 Session
- 限制：只有当前仍在房间（`room_members.left_at IS NULL`）的用户才能建立连接
- 心跳：服务端每 ~54 秒发送一次 Ping 帧；客户端可定期发送 `{"type":"ping"}`，服务端会回复 `{"type":"pong"}`
- 断线：连接关闭后会把该成员状态置为 `offline`

服务端广播的消息类型：

```json
{ "type": "user_joined", "data": { "user_id": 18, "nickname": "测试用户3", "balance": 0, "status": "online", "joined_at": "2025-11-07T05:52:24.220433Z" } }
{ "type": "user_left", "data": { "user_id": 18, "nickname": "测试用户3", "status": "offline", "left_at": "2025-11-07T05:55:24Z" } }
{ "type": "user_kicked", "data": { "user_id": 18, "nickname": "测试用户3", "kicked_by": 16, "kicked_by_nickname": "测试用户1", "status": "offline", "kicked_at": "2025-11-07T05:56:36Z" } }
{ "type": "bet", "data": { "user_id": 16, "nickname": "测试用户1", "amount": 100, "balance": -100, "table_balance": 100, "created_at": "2025-11-07T05:52:30Z" } }
{ "type": "withdraw", "data": { "user_id": 16, "nickname": "测试用户1", "amount": 150, "balance": 0, "table_balance": 0, "created_at": "2025-11-07T05:52:40Z" } }
{ "type": "niuniu_bet", "data": { "user_id": 16, "nickname": "测试用户1", "total_amount": 50, "balance": -50, "table_balance": 50, "bets": [ { "to_user_id": 17, "to_nickname": "测试用户2", "amount": 50 } ], "created_at": "2025-11-07T05:52:45Z" } }
{ "type": "force_transfer", "data": { "user_id": 16, "nickname": "测试用户1", "target_user_id": 17, "target_nickname": "测试用户2", "amount": 320, "actor_balance": 0, "target_balance": 680, "table_balance": 0, "created_at": "2025-11-07T05:53:10Z" } }
{ "type": "settlement_initiated", "data": { "initiated_by": 16, "initiated_by_nickname": "测试用户1", "initiated_at": "2025-11-07T05:52:50Z", "table_balance": 0, "settlement_plan": [] } }
{ "type": "settlement_confirmed", "data": { "confirmed_by": 16, "confirmed_by_nickname": "测试用户1", "settlement_batch": "fd13a3d8-5cbe-4c91-8358-723b01344b59", "settled_at": "2025-11-07T05:52:50.390222Z" } }
{ "type": "room_dissolved", "data": { "room_id": 6, "dissolved_at": "2025-11-07T11:52:00Z" } }
```

当房间长时间（默认 12 小时）没有新的操作记录时，后台守护协程会将房间标记为 `dissolved` 并广播 `room_dissolved`。

## 8. 常见错误示例

```json
// 未登录
{ "code": 401, "message": "未登录或Session已过期，请重新登录", "data": null }

// 权限不足
{ "code": 403, "message": "权限不足，需要管理员权限", "data": null }

// 房间不存在
{ "code": 404, "message": "房间不存在或已解散", "data": null }

// 桌面积分未清零
{ "code": 400, "message": "桌面积分不为0，当前桌面积分：500，无法结算", "data": { "table_balance": 500 } }
```

## 9. 运行注意事项

1. 所有余额相关操作均包裹在数据库事务中，确保原子性与一致性。
2. `user_balances` 记录不会被删除；结算后统一重置为 0。
3. `room_members.left_at` 当前未被写入，暂依靠 `status` 区分 `online/offline`（踢出事件会单独通知，状态同 `offline`）。
4. Session 有效期较长（10 年），如需手动失效可删除 `sessions` 表记录。
5. 服务端日志使用标准库 `log` 输出到控制台。
6. WebSocket 广播采用房间级 Hub，消息在同一连接上使用换行符分隔多条 JSON。

