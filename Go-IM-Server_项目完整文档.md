# Go-IM-Server 项目文档
## 基于Gin的高性能即时通讯系统

> **目标**：4周内完成，作为Go后端求职的核心项目  
> **技术栈**：Gin + GORM + MySQL + Redis + WebSocket  
> **适用人群**：有编程基础但Go不熟的转行者  
> **原则**：用框架、不造轮子、先跑通再优化

---

## 一、项目概述

### 1.1 项目定位

一个**功能完整、可演示、可讲清楚**的即时通讯后端系统。支持用户注册登录、添加好友、实时单聊、历史消息查询、在线状态检测。

**不是**：分布式IM、群聊、文件传输、视频通话、微服务架构。

### 1.2 核心功能（4个模块）

| 模块 | 功能点 | 技术亮点 |
|------|--------|---------|
| **用户系统** | 注册、登录、JWT鉴权、查看个人信息 | 密码加密、Token刷新、统一响应格式 |
| **好友系统** | 发送好友申请、同意/拒绝、删除好友、好友列表 | 双向关系存储、状态机（pending/accepted/rejected） |
| **单聊系统** | 实时消息收发、历史消息分页查询、消息已读回执 | WebSocket长连接、消息持久化、分页优化 |
| **在线状态** | 谁在线/谁离线、心跳检测、断线重连提示 | Redis TTL、定时清理、WebSocket连接管理 |

### 1.3 技术选型理由

| 技术 | 选型 | 理由 |
|------|------|------|
| Web框架 | **Gin** | 国内Go后端标准，面试必问，生态成熟 |
| ORM | **GORM v2** | 最流行，文档全，面试官熟悉 |
| 数据库 | **MySQL 8.0** | 免费、标准、面试八股文成熟 |
| 缓存 | **Redis** | 存在线状态、JWT黑名单、限流计数 |
| WebSocket | **gorilla/websocket** | 最稳定，文档全，示例多 |
| 密码加密 | **bcrypt** | 行业标准，不可逆，自带盐值 |
| JWT | **jwt-go（dgrijalva/jwt-go）** | 标准库，面试常问实现原理 |
| 配置 | **Viper** | 支持yaml/json/env，热更新 |
| 日志 | **Zap** | 高性能，结构化日志，生产级 |

---

## 二、系统架构

### 2.1 整体架构图

```
┌─────────────────────────────────────────────────────────────┐
│                        客户端                                │
│  (Postman / 网页 / 命令行工具)                              │
└──────────────────────┬──────────────────────────────────────┘
                       │
         ┌─────────────┴─────────────┐
         │                           │
   HTTP :8080                  WebSocket :8081
         │                           │
┌────────▼────────┐        ┌────────▼────────┐
│   Gin Router    │        │  WS Hub/Manager │
│                 │        │                 │
│  POST /register │        │  连接管理        │
│  POST /login    │        │  消息路由        │
│  GET  /friends  │        │  心跳检测        │
│  ...            │        │  广播/单播       │
└────────┬────────┘        └────────┬────────┘
         │                           │
┌────────▼───────────────────────────▼────────┐
│              业务层 (Service)                │
│  UserService / FriendService / MsgService   │
└────────┬───────────────────────────┬────────┘
         │                           │
┌────────▼────────┐        ┌────────▼────────┐
│   DAO/Model     │        │     Redis       │
│  (GORM + MySQL) │        │  (go-redis)     │
│                 │        │                 │
│  users          │        │  online:{uid}   │
│  friendships    │        │  jwt:blacklist  │
│  messages       │        │  rate:limit     │
│  conversations  │        │                 │
└─────────────────┘        └─────────────────┘
```

### 2.2 分层职责

```
┌─────────────┐  Router层：URL路由绑定，中间件注册（JWT校验、日志、恢复）
│   Router    │
├─────────────┤  Handler/Controller层：参数绑定、校验、调用Service、返回JSON
│   Handler   │
├─────────────┤  Service层：业务逻辑编排（注册流程、发消息流程、加好友流程）
│   Service   │
├─────────────┤  DAO层：数据库原子操作（CRUD），不处理业务逻辑
│   DAO/Model │
├─────────────┤  Storage层：MySQL + Redis，由GORM和go-redis封装
│   Database  │
└─────────────┘
```

**铁律**：
- Handler 不直接调 DAO，必须走 Service
- Service 不直接操作数据库，必须走 DAO
- DAO 只写原子操作，不写业务判断

---

## 三、数据库设计

### 3.1 表结构（5张表）

#### users（用户表）

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK, AUTO_INCREMENT | 主键 |
| username | VARCHAR(32) | UNIQUE, NOT NULL | 登录账号 |
| password | VARCHAR(255) | NOT NULL | bcrypt加密后的密码 |
| nickname | VARCHAR(32) | DEFAULT '' | 昵称 |
| avatar | VARCHAR(255) | DEFAULT '' | 头像URL |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | 注册时间 |
| updated_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP ON UPDATE | 更新时间 |
| deleted_at | TIMESTAMP | NULL, INDEX | GORM软删除 |

```sql
CREATE TABLE `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(32) NOT NULL,
  `password` varchar(255) NOT NULL,
  `nickname` varchar(32) DEFAULT '',
  `avatar` varchar(255) DEFAULT '',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_username` (`username`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### friendships（好友关系表）

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK | 主键 |
| user_id | BIGINT UNSIGNED | NOT NULL, FK | 发起方 |
| friend_id | BIGINT UNSIGNED | NOT NULL, FK | 接收方 |
| status | TINYINT | DEFAULT 0 | 0=pending, 1=accepted, 2=rejected |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | 申请时间 |
| updated_at | TIMESTAMP | ON UPDATE | 处理时间 |

**注意**：添加唯一索引 `(user_id, friend_id)` 防止重复申请。

```sql
CREATE TABLE `friendships` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `friend_id` bigint unsigned NOT NULL,
  `status` tinyint DEFAULT '0',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_user_friend` (`user_id`,`friend_id`),
  KEY `idx_friend_id` (`friend_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### messages（消息表）

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK | 主键 |
| conversation_id | BIGINT UNSIGNED | NOT NULL, FK | 所属会话 |
| from_user_id | BIGINT UNSIGNED | NOT NULL | 发送者 |
| to_user_id | BIGINT UNSIGNED | NOT NULL | 接收者 |
| content | TEXT | NOT NULL | 消息内容 |
| msg_type | TINYINT | DEFAULT 1 | 1=text, 2=image |
| is_read | TINYINT | DEFAULT 0 | 0=未读, 1=已读 |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | 发送时间 |

**索引策略**：
- `idx_conversation` (conversation_id, created_at) —— 查历史消息
- `idx_to_user` (to_user_id, is_read) —— 查未读消息数

```sql
CREATE TABLE `messages` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `conversation_id` bigint unsigned NOT NULL,
  `from_user_id` bigint unsigned NOT NULL,
  `to_user_id` bigint unsigned NOT NULL,
  `content` text NOT NULL,
  `msg_type` tinyint DEFAULT '1',
  `is_read` tinyint DEFAULT '0',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_conversation` (`conversation_id`,`created_at`),
  KEY `idx_to_user` (`to_user_id`,`is_read`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### conversations（会话表）

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK | 主键 |
| user1_id | BIGINT UNSIGNED | NOT NULL | 用户A（小ID） |
| user2_id | BIGINT UNSIGNED | NOT NULL | 用户B（大ID） |
| last_message_id | BIGINT UNSIGNED | NULL | 最后一条消息ID |
| last_message_at | TIMESTAMP | NULL | 最后消息时间 |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | 创建时间 |

**设计说明**：
- user1_id 永远 < user2_id，保证唯一性
- 唯一索引 `(user1_id, user2_id)`
- 查某人的所有会话：`WHERE user1_id = X OR user2_id = X`

```sql
CREATE TABLE `conversations` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user1_id` bigint unsigned NOT NULL,
  `user2_id` bigint unsigned NOT NULL,
  `last_message_id` bigint unsigned DEFAULT NULL,
  `last_message_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users` (`user1_id`,`user2_id`),
  KEY `idx_user1` (`user1_id`),
  KEY `idx_user2` (`user2_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 3.2 GORM Model定义

```go
package models

import (
    "gorm.io/gorm"
    "time"
)

type User struct {
    ID        uint64         `gorm:"primarykey" json:"id"`
    Username  string         `gorm:"size:32;uniqueIndex;not null" json:"username"`
    Password  string         `gorm:"size:255;not null" json:"-"` // 密码不序列化
    Nickname  string         `gorm:"size:32;default:''" json:"nickname"`
    Avatar    string         `gorm:"size:255;default:''" json:"avatar"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Friendship struct {
    ID        uint64    `gorm:"primarykey" json:"id"`
    UserID    uint64    `gorm:"not null;uniqueIndex:idx_user_friend" json:"user_id"`
    FriendID  uint64    `gorm:"not null;uniqueIndex:idx_user_friend" json:"friend_id"`
    Status    int8      `gorm:"default:0" json:"status"` // 0=pending, 1=accepted, 2=rejected
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type Message struct {
    ID             uint64    `gorm:"primarykey" json:"id"`
    ConversationID uint64    `gorm:"not null;index:idx_conversation" json:"conversation_id"`
    FromUserID     uint64    `gorm:"not null" json:"from_user_id"`
    ToUserID       uint64    `gorm:"not null;index:idx_to_user" json:"to_user_id"`
    Content        string    `gorm:"type:text;not null" json:"content"`
    MsgType        int8      `gorm:"default:1" json:"msg_type"` // 1=text
    IsRead         int8      `gorm:"default:0;index:idx_to_user" json:"is_read"`
    CreatedAt      time.Time `json:"created_at"`
}

type Conversation struct {
    ID            uint64    `gorm:"primarykey" json:"id"`
    User1ID       uint64    `gorm:"not null;uniqueIndex:idx_users" json:"user1_id"` // 小ID
    User2ID       uint64    `gorm:"not null;uniqueIndex:idx_users" json:"user2_id"` // 大ID
    LastMessageID *uint64   `json:"last_message_id"`
    LastMessageAt *time.Time `json:"last_message_at"`
    CreatedAt     time.Time `json:"created_at"`
}
```

---

## 四、API接口设计

### 4.1 统一响应格式

```json
{
    "code": 0,
    "message": "success",
    "data": {}
}
```

| code | 含义 |
|------|------|
| 0 | 成功 |
| 400 | 参数错误 |
| 401 | 未登录/Token无效 |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

```go
package response

type Response struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

func Success(data interface{}) Response {
    return Response{Code: 0, Message: "success", Data: data}
}

func Error(code int, message string) Response {
    return Response{Code: code, Message: message}
}
```

### 4.2 用户模块

| 方法 | 路径 | 说明 | 请求体 | 响应 |
|------|------|------|--------|------|
| POST | `/api/v1/register` | 注册 | `{"username":"zhangsan","password":"123456","nickname":"张三"}` | 用户信息 |
| POST | `/api/v1/login` | 登录 | `{"username":"zhangsan","password":"123456"}` | Token |
| GET | `/api/v1/user/profile` | 查看个人信息 | Header: Authorization | 用户信息 |
| POST | `/api/v1/user/refresh` | 刷新Token | Header: Authorization | 新Token |

### 4.3 好友模块

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| POST | `/api/v1/friends/apply` | 发送好友申请 | 是 |
| POST | `/api/v1/friends/handle` | 同意/拒绝申请 | 是 |
| DELETE | `/api/v1/friends/:id` | 删除好友 | 是 |
| GET | `/api/v1/friends` | 好友列表 | 是 |
| GET | `/api/v1/friends/pending` | 待处理申请列表 | 是 |

### 4.4 消息模块（HTTP部分）

| 方法 | 路径 | 说明 | 参数 |
|------|------|------|------|
| GET | `/api/v1/messages/history` | 历史消息 | `?conversation_id=1&page=1&size=20` |
| GET | `/api/v1/messages/unread` | 未读消息数 | 无 |
| POST | `/api/v1/messages/read` | 标记已读 | `{"message_ids":[1,2,3]}` |

### 4.5 WebSocket接口

| 路径 | 说明 | 认证方式 |
|------|------|---------|
| `ws://localhost:8081/ws?token=xxx` | 建立长连接 | URL参数传JWT |

**消息协议（JSON格式）**：

```json
// 客户端发送消息
{
    "type": "chat",
    "data": {
        "to_user_id": 2,
        "content": "你好"
    }
}

// 服务端推送消息
{
    "type": "message",
    "data": {
        "id": 1,
        "from_user_id": 2,
        "content": "你好",
        "created_at": "2026-08-18T10:00:00Z"
    }
}

// 心跳
{
    "type": "ping"
}
// 服务端回复
{
    "type": "pong"
}

// 错误推送
{
    "type": "error",
    "data": "对方不是你的好友"
}
```

---

## 五、核心功能实现要点

### 5.1 用户注册流程

```
[客户端] POST /register
    ↓
[Handler] 绑定JSON参数，校验（用户名3-32位，密码6-32位）
    ↓
[Service] 调用UserService.Register
    ↓
[DAO] 查数据库：username是否已存在？
    ↓
[Service] 密码bcrypt加密（cost=10）
    ↓
[DAO] 插入users表
    ↓
[Service] 生成JWT Token
    ↓
[Handler] 返回 {token, user_info}
```

**关键代码片段**：

```go
// 密码加密
import "golang.org/x/crypto/bcrypt"

hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
// 校验
err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(inputPassword))

// JWT生成
import "github.com/dgrijalva/jwt-go"

type Claims struct {
    UserID uint64 `json:"user_id"`
    jwt.StandardClaims
}

func GenerateToken(userID uint64) (string, error) {
    claims := Claims{
        UserID: userID,
        StandardClaims: jwt.StandardClaims{
            ExpiresAt: time.Now().Add(7 * 24 * time.Hour).Unix(),
            IssuedAt:  time.Now().Unix(),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte("your-secret-key"))
}
```

### 5.2 JWT中间件

```go
func JWTAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        tokenStr := c.GetHeader("Authorization")
        if tokenStr == "" {
            c.AbortWithStatusJSON(401, response.Error(401, "缺少Token"))
            return
        }

        // 去掉 "Bearer " 前缀
        tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")

        // 解析Token
        token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
            return []byte("your-secret-key"), nil
        })

        if err != nil || !token.Valid {
            c.AbortWithStatusJSON(401, response.Error(401, "Token无效"))
            return
        }

        // 提取user_id存入上下文
        if claims, ok := token.Claims.(*Claims); ok {
            c.Set("user_id", claims.UserID)
        }

        c.Next()
    }
}

// 路由注册时使用
auth := r.Group("/api/v1")
auth.Use(middleware.JWTAuth())
{
    auth.GET("/user/profile", handler.GetProfile)
    // ...
}
```

### 5.3 好友关系设计

**核心问题**：A加B为好友，数据库怎么存？

**方案**：只存一条记录，user_id是发起方，friend_id是接收方。

**查A的好友列表**：
```sql
SELECT * FROM friendships 
WHERE (user_id = A_ID OR friend_id = A_ID) AND status = 1
```

**查A的待处理申请**：
```sql
SELECT * FROM friendships WHERE friend_id = A_ID AND status = 0
```

### 5.4 WebSocket Hub设计

```go
type Hub struct {
    clients    map[uint64]*Client  // user_id -> WebSocket连接
    register   chan *Client        // 新连接注册
    unregister chan *Client        // 连接断开
    broadcast  chan *Message       // 广播消息（可选）
    mu         sync.RWMutex        // 保护clients map
}

type Client struct {
    hub    *Hub
    conn   *websocket.Conn
    userID uint64
    send   chan []byte
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            h.clients[client.userID] = client
            h.mu.Unlock()
            // 更新Redis在线状态
            redis.Set("online:"+strconv.FormatUint(client.userID, 10), "1", 5*time.Minute)

        case client := <-h.unregister:
            h.mu.Lock()
            if _, ok := h.clients[client.userID]; ok {
                delete(h.clients, client.userID)
                close(client.send)
            }
            h.mu.Unlock()
            // 清除在线状态
            redis.Del("online:" + strconv.FormatUint(client.userID, 10))

        case message := <-h.broadcast:
            // 广播逻辑（群聊用，单聊不用）
        }
    }
}

// 发送消息给指定用户
func (h *Hub) SendToUser(userID uint64, msg []byte) error {
    h.mu.RLock()
    client, ok := h.clients[userID]
    h.mu.RUnlock()

    if !ok {
        return errors.New("用户不在线")
    }

    select {
    case client.send <- msg:
        return nil
    default:
        return errors.New("发送通道已满")
    }
}
```

### 5.5 心跳检测机制

```
服务端：
  每个Client启动一个goroutine，每隔30秒检查：
  - 上次收到ping的时间 > 60秒？
  - 是 → 关闭连接，从Hub移除

客户端：
  每隔20秒发送一次 {"type":"ping"}
  收到 {"type":"pong"} 重置计时器
```

```go
func (c *Client) readPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()

    c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
        return nil
    })

    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            break
        }
        // 处理消息...
    }
}
```

### 5.6 消息发送完整流程

```
用户A发送消息给用户B：

1. A通过WebSocket发送 {"type":"chat","data":{"to_user_id":B,"content":"hi"}}
2. Hub收到消息，解析出to_user_id
3. 查数据库：A和B是否为好友？（status=1）
   - 不是 → 返回错误 "对方不是你的好友"
4. 是好友 → 生成conversation_id（取min(A,B)和max(A,B)查conversations表，没有则创建）
5. 消息写入messages表
6. 更新conversations.last_message_id和last_message_at
7. 查B是否在线（查Hub.clients或Redis）
   - 在线 → 通过WebSocket推送给B
   - 离线 → 不推送，B上线后查历史消息
8. 给A返回确认 {"type":"ack","data":{"msg_id":123}}
```

---

## 六、项目目录结构

```
go-im-server/
├── cmd/                    # 入口
│   └── server/
│       └── main.go         # main函数，初始化并启动
├── config/                 # 配置文件
│   └── config.yaml         # 数据库、Redis、JWT密钥等
├── internal/               # 内部代码（不对外暴露）
│   ├── handler/            # HTTP处理器
│   │   ├── user.go
│   │   ├── friend.go
│   │   └── message.go
│   ├── service/            # 业务逻辑层
│   │   ├── user_service.go
│   │   ├── friend_service.go
│   │   └── message_service.go
│   ├── dao/                # 数据访问层
│   │   ├── user_dao.go
│   │   ├── friend_dao.go
│   │   └── message_dao.go
│   ├── model/              # 数据模型（GORM struct）
│   │   └── models.go
│   ├── middleware/         # 中间件
│   │   ├── jwt.go
│   │   ├── cors.go
│   │   └── logger.go
│   ├── ws/                 # WebSocket相关
│   │   ├── hub.go          # Hub管理
│   │   ├── client.go       # 客户端连接
│   │   └── message.go      # WS消息结构
│   └── pkg/                # 内部工具包
│       ├── response/       # 统一响应
│       ├── database/       # 数据库初始化
│       └── redis/          # Redis初始化
├── pkg/                    # 可对外复用的工具（可选）
│   └── utils/
│       └── hash.go         # bcrypt封装
├── go.mod
├── go.sum
├── README.md               # 项目说明（面试必看）
└── docker-compose.yml      # 一键启动MySQL+Redis（可选）
```

---

## 七、4周开发排期

### Week 1：用户系统 + 项目骨架

| 天 | 任务 | 检查点 |
|----|------|--------|
| Day 1 | 项目初始化，跑通/ping | 浏览器访问返回pong |
| Day 2 | 连MySQL，配GORM，自动迁移表 | 数据库出现4张表 |
| Day 3 | 实现注册接口（含密码加密） | Postman注册成功，数据库有数据 |
| Day 4 | 实现登录接口（含JWT生成） | Postman登录返回Token |
| Day 5 | 实现JWT中间件，保护/profile接口 | 无Token返回401，有Token返回用户信息 |
| Day 6 | 统一响应格式，加日志中间件 | 所有接口返回统一JSON |
| Day 7 | 整理代码，写README | GitHub有代码，README有运行说明 |

### Week 2：好友系统 + 消息存储

| 天 | 任务 | 检查点 |
|----|------|--------|
| Day 8 | 实现发送好友申请 | Postman发送申请，数据库有记录 |
| Day 9 | 实现同意/拒绝好友申请 | 状态变更正确，双向查询正常 |
| Day 10 | 实现删除好友、好友列表查询 | 列表返回正确 |
| Day 11 | 设计消息表和会话表，实现发消息存库 | 消息写入messages表 |
| Day 12 | 实现历史消息分页查询 | 分页参数生效，按时间倒序 |
| Day 13 | 实现未读消息数查询 | 返回正确的未读数量 |
| Day 14 | 整理代码，修复Bug | 好友+消息模块联调通过 |

### Week 3：WebSocket实时通信

| 天 | 任务 | 检查点 |
|----|------|--------|
| Day 15 | 搭建WebSocket Hub基础结构 | 客户端能连上ws://localhost:8081/ws |
| Day 16 | 实现连接注册/注销，在线状态存Redis | 连接/断开时Redis状态变化 |
| Day 17 | 实现单聊消息推送 | A发消息，B实时收到 |
| Day 18 | 实现消息ACK机制 | 发送方收到ack确认 |
| Day 19 | 实现心跳检测（ping/pong） | 60秒无心跳自动断开 |
| Day 20 | 离线消息处理（上线后拉取） | 离线期间的消息上线后能看到 |
| Day 21 | WebSocket与HTTP模块联调 | 完整流程跑通 |

### Week 4：收尾 + 面试准备

| 天 | 任务 | 检查点 |
|----|------|--------|
| Day 22 | 代码重构，加注释，处理边界情况 | 代码整洁，无明显Bug |
| Day 23 | 写完整README（功能、架构、运行方式） | README能指导别人跑起来 |
| Day 24 | 写Docker Compose（可选加分） | docker-compose up能一键启动 |
| Day 25 | 整理项目亮点，准备面试讲稿 | 能讲15分钟不卡壳 |
| Day 26 | 背Go八股文（并发、GC、GMP、Channel） | 能回答Top 20问题 |
| Day 27 | 手撕代码练习（字符串、链表、排序） | 能15分钟内写出 |
| Day 28 | 海投简历，开始面试 | 每天投5-10家 |

---

## 八、部署与运行

### 8.1 环境要求

- Go 1.21+
- MySQL 8.0+
- Redis 6.0+

### 8.2 快速启动

```bash
# 1. 克隆项目
git clone https://github.com/yourname/go-im-server.git
cd go-im-server

# 2. 安装依赖
go mod download

# 3. 创建数据库
create database im_db charset=utf8mb4;

# 4. 修改配置文件 config/config.yaml
mysql:
  dsn: "root:password@tcp(127.0.0.1:3306)/im_db?charset=utf8mb4&parseTime=True"
redis:
  addr: "127.0.0.1:6379"
  password: ""
  db: 0
jwt:
  secret: "your-secret-key-change-in-production"
  expire: 168h

# 5. 运行
go run cmd/server/main.go

# 6. 测试
curl http://localhost:8080/ping
```

### 8.3 Docker Compose（可选）

```yaml
version: '3.8'
services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root123
      MYSQL_DATABASE: im_db
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql

  redis:
    image: redis:6-alpine
    ports:
      - "6379:6379"

volumes:
  mysql_data:
```

---

## 九、面试讲稿模板

### 9.1 项目介绍（2分钟）

> "这是我独立完成的一个即时通讯后端系统，基于Go语言开发。核心功能包括用户注册登录、好友管理、实时单聊、历史消息查询和在线状态检测。
> 
> 技术栈是Gin做HTTP框架，GORM操作MySQL，Redis存在线状态和JWT黑名单，WebSocket实现实时推送。
> 
> 项目亮点有三个：一是用bcrypt做密码加密、JWT做无状态鉴权；二是设计了好友关系的状态机，支持申请、同意、拒绝、删除的完整生命周期；三是WebSocket层做了Hub集中管理，支持心跳检测和断线重连，同时用Redis做在线状态兜底。"

### 9.2 技术深挖点（面试官可能问的）

**Q：为什么用Gin而不是标准库的net/http？**
> "Gin的路由基于Radix Tree，性能是标准库的40倍左右。而且Gin提供了中间件机制、参数自动绑定、统一错误处理，开发效率更高。标准库更适合做简单的服务，IM系统需要路由分组、JWT中间件、日志恢复，用Gin更合适。"

**Q：JWT怎么实现的？刷新机制怎么设计的？**
> "用户登录成功后，服务端用jwt-go生成Token，包含user_id和过期时间，用HS256签名。Token有效期7天，存客户端本地。每次请求带在Header里，服务端中间件解析验证。
> 
> 刷新机制是：访问/profile等接口时，如果Token还剩不到1天过期，服务端在响应Header里附上一个新Token，客户端下次请求自动替换。"

**Q：WebSocket怎么管理连接的？如果服务器重启，消息会丢吗？**
> "服务端有一个Hub结构，用map管理user_id到连接的映射，读写用RWMutex保护。每个连接有两个goroutine，一个读一个写，通过channel解耦。
> 
> 服务器重启时，WebSocket连接会断开，但消息已经持久化到MySQL了。用户重连后，客户端主动拉取离线期间的历史消息，所以不会丢。"

**Q：怎么保证消息顺序？**
> "消息表有自增ID和创建时间，查询历史消息时按ID或时间排序。单聊场景下，同一个conversation_id的消息，按创建时间正序排列即可保证顺序。
> 
> 如果考虑分布式部署，可以用Snowflake算法生成全局递增ID，或者用数据库的auto_increment。"

**Q：如果用户量变大，这个架构怎么扩展？**
> "当前是单体架构，扩展方向有几个：
> 1. 水平扩展：多台服务器+负载均衡，WebSocket用Redis Pub/Sub做跨服务器消息转发；
> 2. 数据库分片：按user_id分库分表，或者按conversation_id分表；
> 3. 读写分离：历史消息查询走从库，写操作走主库；
> 4. 消息队列：离线消息先入Kafka，再异步落库，降低写库压力。"

### 9.3 项目亮点总结（简历上写）

- **架构清晰**：严格分层（Router-Handler-Service-DAO），模块间解耦
- **安全实践**：bcrypt密码加密、JWT无状态鉴权、统一响应格式防信息泄露
- **实时通信**：WebSocket Hub集中管理，支持心跳检测、断线重连
- **性能考虑**：数据库索引优化（conversation_id+created_at复合索引）、Redis缓存在线状态
- **代码规范**：统一错误处理、结构化日志、Git版本管理

---

## 十、常见问题与解决方案

| 问题 | 原因 | 解决方案 |
|------|------|---------|
| GORM自动迁移失败 | 数据库连接配置错误 | 检查DSN格式，确认数据库已创建 |
| JWT验证失败 | 密钥不一致或Token过期 | 检查config.yaml里的secret，确认Token未过期 |
| WebSocket连不上 | 路径错误或跨域问题 | 确认路径是/ws，检查CORS中间件配置 |
| 消息发送成功但对方收不到 | 对方不在线或user_id错误 | 检查Hub.clients里是否有目标用户，确认user_id正确 |
| 数据库查询慢 | 缺少索引 | 检查messages表的idx_conversation索引是否存在 |
| 密码明明正确但登录失败 | bcrypt版本不兼容或密码被截断 | 确认password字段长度>=255，使用bcrypt.DefaultCost |

---

## 附录：依赖安装命令

```bash
# Web框架
go get -u github.com/gin-gonic/gin

# ORM
go get -u gorm.io/gorm
go get -u gorm.io/driver/mysql

# Redis
go get -u github.com/redis/go-redis/v9

# WebSocket
go get -u github.com/gorilla/websocket

# JWT
go get -u github.com/dgrijalva/jwt-go

# 密码加密
go get -u golang.org/x/crypto/bcrypt

# 配置管理
go get -u github.com/spf13/viper

# 日志
go get -u go.uber.org/zap
```

---

> **使用说明**：这份文档是项目的"总纲"和"说明书"。开发时按Week 1-Day 1的顺序推进，每天只聚焦当天的任务，不要提前看后面的内容。遇到具体技术问题（如"GORM怎么查第一条记录"），去搜GORM文档或问AI，但只问具体函数，不要问"怎么写注册接口"。
> 
> 每天结束时，确保当天的检查点通过，代码提交到GitHub。
