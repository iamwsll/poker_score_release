# 数据库设计文档

## 技术选型
- 数据库：SQLite
- ORM：Gorm
- 字符编码：UTF-8

## 表结构设计

### 1. users - 用户表
存储用户基础信息

| 字段名 | 类型 | 说明 | 约束 |
|--------|------|------|------|
| id | INTEGER | 用户ID | PRIMARY KEY, AUTO_INCREMENT |
| phone | VARCHAR(11) | 手机号 | UNIQUE, NOT NULL |
| nickname | VARCHAR(50) | 昵称 | NOT NULL |
| password_hash | VARCHAR(255) | 密码哈希值（bcrypt） | NOT NULL |
| role | VARCHAR(20) | 用户角色（admin/user） | NOT NULL, DEFAULT 'user' |
| created_at | DATETIME | 创建时间 | NOT NULL |
| updated_at | DATETIME | 更新时间 | NOT NULL |

**索引：**
- idx_phone: (phone) UNIQUE
- idx_role: (role)

---

### 2. sessions - Session表
管理用户登录Session

| 字段名 | 类型 | 说明 | 约束 |
|--------|------|------|------|
| id | INTEGER | Session ID | PRIMARY KEY, AUTO_INCREMENT |
| session_id | VARCHAR(255) | Session标识符（UUID） | UNIQUE, NOT NULL |
| user_id | INTEGER | 用户ID | NOT NULL, FOREIGN KEY |
| created_at | DATETIME | 创建时间 | NOT NULL |
| expires_at | DATETIME | 过期时间 | NOT NULL |

**索引：**
- idx_session_id: (session_id) UNIQUE
- idx_user_id: (user_id)
- idx_expires_at: (expires_at)

**外键：**
- user_id → users.id

---

### 3. rooms - 房间表
存储房间基础信息

| 字段名 | 类型 | 说明 | 约束 |
|--------|------|------|------|
| id | INTEGER | 房间ID | PRIMARY KEY, AUTO_INCREMENT |
| room_code | VARCHAR(6) | 6位房间号 | NOT NULL |
| room_type | VARCHAR(20) | 房间类型（texas/niuniu） | NOT NULL |
| chip_rate | VARCHAR(20) | 积分与人民币比例（如"20:1"） | NOT NULL |
| status | VARCHAR(20) | 房间状态（active/dissolved） | NOT NULL, DEFAULT 'active' |
| created_by | INTEGER | 创建者用户ID | NOT NULL, FOREIGN KEY |
| created_at | DATETIME | 创建时间 | NOT NULL |
| dissolved_at | DATETIME | 解散时间 | NULL |

**索引：**
- idx_room_code: (room_code, created_at)
- idx_status: (status)
- idx_created_by: (created_by)

**外键：**
- created_by → users.id

**注意：** room_code不做唯一约束，因为历史房间解散后，房间号可以重复使用。通过room_code + status='active'来查询活跃房间。

---

### 4. room_members - 房间成员表
记录用户进出房间的历史

| 字段名 | 类型 | 说明 | 约束 |
|--------|------|------|------|
| id | INTEGER | 记录ID | PRIMARY KEY, AUTO_INCREMENT |
| room_id | INTEGER | 房间ID | NOT NULL, FOREIGN KEY |
| user_id | INTEGER | 用户ID | NOT NULL, FOREIGN KEY |
| joined_at | DATETIME | 加入时间 | NOT NULL |
| status | VARCHAR(20) | 状态（online/offline） | NOT NULL, DEFAULT 'online' |

**索引：**
- idx_room_user: (room_id, user_id)
- idx_status: (status)
- idx_joined_at: (joined_at)

**外键：**
- room_id → rooms.id
- user_id → users.id

**注意：** 当前实现仅在用户首次进入房间时创建记录，后续通过 `status` 字段标记 `online/offline` 状态，不再单独记录离开时间。

---

### 5. user_balances - 用户积分余额表
记录用户在各个房间的当前积分余额

| 字段名 | 类型 | 说明 | 约束 |
|--------|------|------|------|
| id | INTEGER | 记录ID | PRIMARY KEY, AUTO_INCREMENT |
| room_id | INTEGER | 房间ID | NOT NULL, FOREIGN KEY |
| user_id | INTEGER | 用户ID | NOT NULL, FOREIGN KEY |
| balance | INTEGER | 当前积分余额（可为负） | NOT NULL, DEFAULT 0 |
| updated_at | DATETIME | 更新时间 | NOT NULL |

**索引：**
- idx_room_user: (room_id, user_id) UNIQUE
- idx_room_id: (room_id)

**外键：**
- room_id → rooms.id
- user_id → users.id

**约束：** 同一房间同一用户只有一条记录（UNIQUE约束），积分更新通过数据库事务完成，`updated_at`由Gorm自动维护。

---

### 6. room_operations - 房间操作记录表
记录房间内所有操作历史

| 字段名 | 类型 | 说明 | 约束 |
|--------|------|------|------|
| id | INTEGER | 记录ID | PRIMARY KEY, AUTO_INCREMENT |
| room_id | INTEGER | 房间ID | NOT NULL, FOREIGN KEY |
| user_id | INTEGER | 操作用户ID | NOT NULL, FOREIGN KEY |
| operation_type | VARCHAR(50) | 操作类型 | NOT NULL |
| amount | INTEGER | 涉及积分数量 | NULL |
| target_user_id | INTEGER | 目标用户ID（踢人、给某人下注） | NULL, FOREIGN KEY |
| description | TEXT | 操作描述（大部分为可读文本，牛牛下注会写入JSON字符串） | NULL |
| created_at | DATETIME | 操作时间 | NOT NULL |

**操作类型（operation_type）：**
- `join`: 加入房间
- `leave`: 离开房间
- `bet`: 下注/支出
- `withdraw`: 收回
- `kick`: 被踢出
- `settlement_initiated`: 发起结算
- `settlement_confirmed`: 确认结算
- `niuniu_bet`: 牛牛下注（给某人下注）

**索引：**
- idx_room_id: (room_id, created_at)
- idx_user_id: (user_id)
- idx_operation_type: (operation_type)

**外键：**
- room_id → rooms.id
- user_id → users.id
- target_user_id → users.id

---

### 7. settlements - 结算记录表
记录每次结算的详细数据

| 字段名 | 类型 | 说明 | 约束 |
|--------|------|------|------|
| id | INTEGER | 记录ID | PRIMARY KEY, AUTO_INCREMENT |
| room_id | INTEGER | 房间ID | NOT NULL, FOREIGN KEY |
| user_id | INTEGER | 用户ID | NOT NULL, FOREIGN KEY |
| chip_amount | INTEGER | 积分盈亏（正为盈，负为亏） | NOT NULL |
| rmb_amount | DECIMAL(10,2) | 人民币盈亏（根据比例计算） | NOT NULL |
| settled_at | DATETIME | 结算时间 | NOT NULL |
| settlement_batch | VARCHAR(50) | 结算批次号（UUID） | NOT NULL |

**索引：**
- idx_room_id: (room_id, settled_at)
- idx_user_id: (user_id, settled_at)
- idx_settlement_batch: (settlement_batch)
- idx_settled_at: (settled_at)

**外键：**
- room_id → rooms.id
- user_id → users.id

**注意：** 同一批次结算的所有记录具有相同的settlement_batch值，便于查询。

---

### 8. bet_records - 牛牛下注记录表
专门记录牛牛游戏中"给某人下注"的详细记录

| 字段名 | 类型 | 说明 | 约束 |
|--------|------|------|------|
| id | INTEGER | 记录ID | PRIMARY KEY, AUTO_INCREMENT |
| room_id | INTEGER | 房间ID | NOT NULL, FOREIGN KEY |
| from_user_id | INTEGER | 下注者用户ID | NOT NULL, FOREIGN KEY |
| to_user_id | INTEGER | 被下注者用户ID | NOT NULL, FOREIGN KEY |
| amount | INTEGER | 下注积分数量 | NOT NULL |
| created_at | DATETIME | 下注时间 | NOT NULL |

**索引：**
- idx_room_id: (room_id, created_at)
- idx_from_user: (from_user_id)
- idx_to_user: (to_user_id)

**外键：**
- room_id → rooms.id
- from_user_id → users.id
- to_user_id → users.id

**注意：** 此表仅用于牛牛游戏，用于记录“谁对谁下注了多少”。业务侧目前在牛牛下注时写入，结算时结合余额与操作记录计算盈亏。

---

## 数据约束与业务规则

### 1. 积分守恒原则
- 业务约定同一房间内所有用户的`balance`求和为0
- 代码通过统一的事务更新与操作记录来维持该约定，当前不会额外做单独校验

### 2. 桌面积分计算
- 桌面积分 = 所有下注（含牛牛下注）金额之和 - 所有收回金额之和
- 结果会被限制为不小于0，主要用于结算校验与前端展示

### 3. 房间状态管理
- 后台协程每小时巡检一次，若房间12小时内没有新的操作则标记为`dissolved`
- 离开房间或踢人仅更新`status`字段，不会立即解散房间
- `dissolved`状态房间无法被再次加入

### 4. Session管理
- Session默认有效期为10年（可在配置中调整）
- 每次请求都会检查过期时间，过期即删除Session并返回401

### 5. 结算条件
- 只有当桌面积分=0时才能发起结算
- 结算时清空所有用户的balance

### 6. 历史记录
- 用户只能看到自己加入房间后的操作记录
- 管理员可以看到所有历史记录

---

## 初始化数据

### 默认管理员账户
```sql
phone: 13800138000
password: admin123
nickname: 系统管理员
role: admin
```

---

## 数据库迁移

使用Gorm的AutoMigrate功能自动创建和更新表结构：

```go
db.AutoMigrate(
    &models.User{},
    &models.Session{},
    &models.Room{},
    &models.RoomMember{},
    &models.UserBalance{},
    &models.RoomOperation{},
    &models.Settlement{},
    &models.BetRecord{},
)
```

---

## 性能优化建议

1. **定期清理过期Session**：使用定时任务删除expires_at < NOW()的记录
2. **历史数据归档**：对于dissolved的房间，可以考虑定期归档
3. **索引优化**：根据实际查询情况调整索引
4. **事务处理**：涉及积分变动的操作必须使用事务确保原子性

