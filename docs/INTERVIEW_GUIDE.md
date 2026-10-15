# go-manus 面试指南

> 面试材料中的示例用于解释设计思路；接口、字段和目录如与实现不一致，以当前代码及 [`ARCHITECTURE.md`](./ARCHITECTURE.md) 为准。

> 本文档面向资深 Go/后端开发工程师面试准备，基于 go-manus AI Agent 平台真实代码深度解析。

## 目录

1. [项目概述](#1-项目概述)
2. [技术栈总结](#2-技术栈总结)
3. [核心知识点详解](#3-核心知识点详解)
4. [可能的面试追问](#4-可能的面试追问)
5. [注意事项和踩坑点](#5-注意事项和踩坑点)

---

## 1. 项目概述

### 1.1 项目定位

go-manus 是一个 **AI Agent 平台**，核心功能是：
- 提供多 Agent 协作能力（Planner Agent + React Agent）
- 支持 MCP（Model Context Protocol）工具扩展
- 支持 A2A（Agent-to-Agent）协议通信
- 提供 Web UI（Next.js）和 API 服务（Go）
- 内置沙箱环境（Python）和浏览器自动化

### 1.2 核心模块

| 模块 | 路径 | 职责 |
|------|------|------|
| **Agent** | `internal/agent/` | Agent 核心逻辑、ReAct 循环、工具执行 |
| **Repository** | `internal/repository/` | 数据持久化（PostgreSQL） |
| **Service** | `internal/service/` | 业务逻辑层（会话、文件、清理） |
| **Model** | `internal/model/` | 数据模型定义 |
| **Handler** | `internal/handler/` | HTTP 路由处理 |
| **External** | `internal/external/` | 外部依赖适配器（LLM、Redis、MinIO） |

### 1.3 数据库设计

**Sessions 表**（会话存储）：
```sql
CREATE TABLE sessions (
    id              VARCHAR(255) PRIMARY KEY,
    sandbox_id      VARCHAR(255),          -- 关联的沙箱容器 ID
    task_id         VARCHAR(255),          -- 外部任务 ID
    title           VARCHAR(255),
    unread_message_count INT DEFAULT 0,    -- 未读消息数（UI 小红点）
    latest_message  TEXT,                  -- 最新消息摘要
    latest_message_at TIMESTAMP,           -- 用于列表排序
    events          JSONB,                 -- 事件历史（完整对话记录）
    memories        JSONB,                 -- AI 记忆（per-agent）
    status          VARCHAR(255),          -- pending/running/completed
    deleted_at      TIMESTAMP,             -- 软删除标记
    created_at, updated_at                 -- 时间戳
);
```

**Files 表**（文件元数据）：
```sql
CREATE TABLE files (
    id          VARCHAR(255) PRIMARY KEY,
    session_id  VARCHAR(255) NOT NULL,     -- 关联会话
    filename    VARCHAR(255),              -- 原始文件名
    filepath    VARCHAR(255),              -- 服务器路径
    key         VARCHAR(255),              -- MinIO 存储 Key
    extension   VARCHAR(255),              -- 扩展名
    mime_type   VARCHAR(255),              -- MIME 类型
    size        BIGINT,                    -- 字节数
    created_at  TIMESTAMP
);
```

---

## 2. 技术栈总结

| 类别 | 技术选型 | 替代方案 |
|------|----------|----------|
| **语言** | Go 1.21+ | - |
| **Web 框架** | Gin | Echo, Chi |
| **数据库** | PostgreSQL + pgx/v5 | database/sql, go-pg |
| **ORM** | 纯 SQL（pgx） | GORM, sqlx, sqlc |
| **缓存** | Redis | - |
| **对象存储** | MinIO (S3 兼容) | OSS, COS |
| **AI SDK** | Anthropic Claude API | OpenAI, 本地模型 |
| **测试** | Go testing + docker-compose | - |
| **构建标签** | `//go:build integration` | - |

---

## 3. 核心知识点详解

### 3.1 Go 数据库错误处理：errors.Is() vs ==

#### 问题背景

项目中有这样一个 **真实踩坑记录**：

```go
// ❌ 错误写法（早期代码）
err := row.Scan(&session)
if err == sql.ErrNoRows {
    return nil, nil
}
```

```go
// ✅ 正确写法
err := row.Scan(&session)
if errors.Is(err, pgx.ErrNoRows) {
    return nil, nil
}
```

#### Why：为什么不能直接用 == 比较？

**这是 Go 1.13+ error wrapping 的核心考点。**

```
┌─────────────────────────────────────────────────────────────────┐
│  Go 错误类型层次                                                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  sql.ErrNoRows (database/sql 包)                                 │
│       ↓                                                          │
│       是 unexported 的 sentinel error 实例                       │
│       值: "sql: no rows in result set"                          │
│                                                                  │
│  pgx.ErrNoRows (jackc/pgx/v5 包)                                 │
│       ↓                                                          │
│       是 unexported 的 sentinel error 实例                       │
│       值: "pgx: no rows in result set"                          │
│                                                                  │
│  ⚠️ 两者都是 "sql: no rows in result set"                        │
│     但它们是两个完全不同的指针地址！                               │
│                                                                  │
│  错误比较：                                                        │
│     sql.ErrNoRows == pgx.ErrNoRows  →  false ❌                   │
│                                                                  │
│     因为 == 比较的是指针地址，不是字符串值                         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

#### errors.Is() 工作原理

```go
// 伪代码实现
func Is(err, target error) bool {
    // 1. 直接比较
    if err == target {
        return true
    }

    // 2. 检查错误链（逐层 unwrap）
    for err != nil {
        if err == target {
            return true
        }
        // 获取被包装的错误（Go 1.13+ fmt.Errorf %w）
        err = Unwrap(err)
    }
    return false
}
```

**关键点**：
1. `errors.Is()` 会遍历整个错误链
2. 支持 `fmt.Errorf("context: %w", err)` 包装的错误
3. 两个不同的 sentinel error，即使字符串相同，`==` 也返回 false

#### 项目中的正确写法

```go
// session_repository.go 第 145 行
err := q.QueryRow(ctx, query, id).Scan(...)
if err != nil {
    if errors.Is(err, pgx.ErrNoRows) {  // ✅ 使用 errors.Is()
        return nil, nil
    }
    return nil, err
}
```

```go
// file_repository.go 第 115 行
err := q.QueryRow(ctx, query, sessionID, filepath).Scan(...)
if err != nil {
    if errors.Is(err, pgx.ErrNoRows) {  // ✅ 使用 errors.Is()
        return nil, nil
    }
    return nil, err
}
```

#### 面试追问

| 问题 | 考察点 | 回答要点 |
|------|--------|----------|
| `errors.As()` 和 `errors.Is()` 区别？ | 类型断言 vs 误差比较 | `Is` 比较 identity，`As` 提取具体类型 |
| 为什么要用 unexported sentinel error？ | API 设计 | 防止外部伪造错误，只允许 `Is()` 比较 |
| `%v` vs `%w` vs `%q`？ | 格式化 verb | `%v` 只显示，`%w` 可包装，`%q` 带引号 |

---

### 3.2 PostgreSQL INTERVAL 与参数化查询

#### 问题背景

```go
// ❌ 错误写法
query := `WHERE created_at < NOW() - INTERVAL $1`
_, err := q.Query(ctx, query, "24 hours")

// ✅ 正确写法
query := `WHERE created_at < NOW() - $1::interval`
_, err := q.Query(ctx, query, "24 hours")
```

#### Why：为什么 INTERVAL 不能参数化？

```
┌─────────────────────────────────────────────────────────────────┐
│  PostgreSQL INTERVAL 语法限制                                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  语法规则：INTERVAL 后面必须是字符串常量                          │
│                                                                  │
│  ✅ 正确写法：                                                    │
│     INTERVAL '24 hours'        -- 字符串常量                     │
│     INTERVAL '7 days'          -- 字符串常量                     │
│     $1::interval               -- 类型转换表达式                 │
│                                                                  │
│  ❌ 错误写法：                                                    │
│     INTERVAL $1                -- 参数占位符，不支持！            │
│                                                                  │
│  PostgreSQL 在解析阶段需要确定类型，参数化查询在执行阶段才传值     │
│  这两个阶段不同步，导致类型解析失败                                │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

#### 项目中的正确实现

```go
// file_repository.go 第 225-255 行
func (r *PostgresFileRepository) GetExpiredFiles(ctx context.Context,
    expireDuration string, limit int64) ([]*model.File, error) {

    query := `
        SELECT f.id, f.session_id, f.filename, ...
        FROM files f
        LEFT JOIN sessions s ON f.session_id = s.id
        WHERE f.created_at < NOW() - $1::interval      -- ✅ 类型转换
          AND (f.session_id IS NULL OR s.deleted_at IS NOT NULL)
        ORDER BY f.created_at ASC
        LIMIT $2
    `

    rows, err := q.Query(ctx, query, expireDuration, limit)
    // expireDuration = "24 hours" 或 "7 days" 等
}
```

#### 面试追问

| 问题 | 考察点 | 回答要点 |
|------|--------|----------|
| `$1::text` 是什么语法？ | PostgreSQL 类型转换 | `::` 是 PostgreSQL 的类型转换运算符 |
| 参数化查询如何防 SQL 注入？ | 安全意识 | 参数和 SQL 分离传输，恶意内容被当作字面值 |
| `::text[]` 是什么？ | PostgreSQL 数组 | `text[]` 是字符串数组，`ANY($1)` 展开 |

---

### 3.3 Repository 层架构设计

#### 接口设计

```go
// session_repository.go 第 21-47 行
type SessionRepository interface {
    Create(ctx context.Context, session *model.Session) error
    GetByID(ctx context.Context, id string) (*model.Session, error)
    GetAll(ctx context.Context) ([]*model.Session, error)
    List(ctx context.Context, limit, offset int) ([]*model.Session, int, error)
    Update(ctx context.Context, session *model.Session) error
    Delete(ctx context.Context, id string) error

    // 事件操作
    AppendEvent(ctx context.Context, id string, event *model.Event) error

    // 内存操作（per-agent）
    GetMemory(ctx context.Context, id string, agentName string) ([]llmcore.Message, error)
    SaveMemory(ctx context.Context, id string, agentName string, messages []llmcore.Message) error

    // 原子更新
    UpdateTitle(ctx context.Context, id string, title string) error
    UpdateLatestMessage(ctx context.Context, id string, message string) error
    UpdateStatus(ctx context.Context, id string, status model.SessionStatus) error
    IncrementUnreadCount(ctx context.Context, id string) error
    DecrementUnreadCount(ctx context.Context, id string) error
    SetUnreadCount(ctx context.Context, id string, count int) error

    // 事务支持
    WithTx(ctx context.Context, fn func(repo SessionRepository) error) error
}
```

#### 软删除模式

```go
// session_repository.go 第 271-278 行
// Delete 删除会话（软删除）
// 设置 deleted_at 时间戳，会话关联的文件在过期后会被自动清理
func (r *PostgresSessionRepository) Delete(ctx context.Context, id string) error {
    q := r.queryer(ctx)
    query := `UPDATE sessions SET deleted_at = NOW() WHERE id = $1`
    _, err := q.Exec(ctx, query, id)
    return err
}
```

**软删除的 Where 条件**：
```go
// 查询时过滤
query := `FROM sessions WHERE id = $1 AND deleted_at IS NULL`

// 文件清理时关联查询（孤儿文件判定）
query := `
    FROM files f
    LEFT JOIN sessions s ON f.session_id = s.id
    WHERE f.created_at < NOW() - $1::interval
      AND (f.session_id IS NULL OR s.deleted_at IS NOT NULL)  -- 孤儿文件
`
```

#### 事务支持：QueryContext 接口

```go
// session_repository.go 第 72-77 行
type QueryContext interface {
    QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
    Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
    Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}
```

**为什么需要 QueryContext？**

```
┌─────────────────────────────────────────────────────────────────┐
│  事务切换的实现方式                                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  非事务模式：                                                     │
│      poolQueryContext → pgxpool.Pool.QueryRow()                 │
│                      → 使用连接池                                │
│                                                                  │
│  事务模式：                                                       │
│      txQueryContext → pgx.Tx.QueryRow()                         │
│                    → 使用当前事务                                │
│                                                                  │
│  业务代码无感知：                                                 │
│      r.queryer() 根据 tx != nil 判断返回哪个实现                  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

```go
// session_repository.go 第 65-70 行
func (r *PostgresSessionRepository) queryer(ctx context.Context) QueryContext {
    if r.tx != nil {
        return &txQueryContext{tx: r.tx, ctx: ctx}
    }
    return &poolQueryContext{pool: r.db.Pool}
}
```

#### WithTx 事务封装

```go
// session_repository.go 第 419-455 行
func (r *PostgresSessionRepository) WithTx(ctx context.Context,
    fn func(repo SessionRepository) error) error {

    // 如果已经在事务中，直接执行（嵌套事务）
    if r.tx != nil {
        return fn(r)
    }

    tx, err := r.db.Pool.Begin(ctx)
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }

    // 确保事务回滚
    var committed bool
    defer func() {
        if !committed {
            if p := recover(); p != nil {
                tx.Rollback(ctx)
                panic(p)  // 重新抛出 panic
            }
            tx.Rollback(ctx)  // 失败时回滚
        }
    }()

    err = fn(&PostgresSessionRepository{db: r.db, tx: tx})
    if err != nil {
        return err
    }

    if commitErr := tx.Commit(ctx); commitErr != nil {
        return fmt.Errorf("commit transaction: %w", commitErr)
    }
    committed = true
    return nil
}
```

**关键设计点**：
1. **嵌套事务支持**：检测到 `r.tx != nil` 直接执行
2. **panic 处理**：recover 并回滚，符合 Go 错误处理哲学
3. **错误包装**：每一步都包装错误，便于溯源

#### 构造函数设计：普通版 vs 事务版

```go
// 普通版
func NewSessionRepository(db *infrastructure.Postgres) SessionRepository {
    return &PostgresSessionRepository{db: db}
}

// 事务版（接收已开启的 tx）
func NewSessionRepositoryWithTx(tx pgx.Tx) SessionRepository {
    return &PostgresSessionRepository{tx: tx}
}
```

**使用场景**：
```go
// 外部调用
err := repo.WithTx(ctx, func(txRepo repository.SessionRepository) error {
    // 在事务中，可以使用 txRepo
    // NewSessionRepositoryWithTx(tx) 在 WithTx 内部创建
    return txRepo.UpdateTitle(ctx, id, title)
})
```

#### 面试追问

| 问题 | 考察点 | 回答要点 |
|------|--------|----------|
| 软删除 vs 硬删除？ | 数据安全 | 软删除可恢复，硬删除不可恢复；文件清理延迟处理 |
| 事务隔离级别？ | 数据库基础 | pgx 默认 Read Committed，可设置 `BeginTx` |
| 连接池 vs 事务？ | 资源管理 | 事务持有连接，连接池管理复用 |

---

### 3.4 集成测试

#### 构建标签

```go
// integration_test.go 第 1 行
//go:build integration
```

**运行方式**：
```bash
go test -tags=integration -v ./tests/...
```

#### TestMain 基础设施

```go
// integration_test.go 第 50-87 行
func TestMain(m *testing.M) {
    gin.SetMode(gin.TestMode)

    cfg, err := config.LoadWithValidation("../config.test.yaml")

    app, err := bootstrap.Build(cfg, bootstrap.Options{
        EnablePostgres:    true,   // 真实 PostgreSQL
        EnableRedis:       true,   // 真实 Redis
        EnableStorage:     true,   // 真实 MinIO
        EnableLLM:         false,  // 不需要 AI
        EnableSandbox:     false,  // 不需要沙箱
        EnableBrowser:     false,  // 不需要浏览器
        EnableAgent:       false,  // 不需要 Agent
        EnableRoutes:      true,   // 需要路由测试
        EnableHealthCheck: true,
    })

    testCfg = cfg
    testApp = app
    testServer = app.Engine
    testDB = app.Postgres

    code := m.Run()

    if testApp != nil {
        testApp.Close()
    }

    os.Exit(code)
}
```

#### docker-compose 测试环境

```yaml
# 来自项目 docker-compose.yml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: manus_test
      POSTGRES_USER: manus
      POSTGRES_PASSWORD: manus123

  redis:
    image: redis:7-alpine

  minio:
    image: minio/minio
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
```

**启动脚本**：
```bash
# scripts/test-env-up.sh
cd ../../scripts && ./test-env-up.sh

# scripts/test-env-down.sh（清理）
cd ../../scripts && ./test-env-down.sh
```

#### 测试辅助函数

```go
// 响应断言
func assertOK(t *testing.T, w *httptest.ResponseRecorder) { ... }
func assertSuccess(t *testing.T, w *httptest.ResponseRecorder) response.Response { ... }
func assertError(t *testing.T, w *httptest.ResponseRecorder, expectedCode int) response.Response { ... }

// HTTP 请求
func postJSON(t *testing.T, path string, body any) *httptest.ResponseRecorder { ... }
func getJSON(t *testing.T, path string) *httptest.ResponseRecorder { ... }
func putJSON(t *testing.T, path string, body any) *httptest.ResponseRecorder { ... }
func deleteRequest(t *testing.T, path string) *httptest.ResponseRecorder { ... }
func postFormData(t *testing.T, path string, fields map[string]string,
    fileName string, fileContent []byte) *httptest.ResponseRecorder { ... }

// 资源清理
func CleanupSession(t *testing.T, sessionID string) { ... }
func CleanupFile(t *testing.T, fileID string) { ... }
func CleanupAppConfig(t *testing.T, configType, configKey string) { ... }
```

#### 竞态检测

```bash
# -race 标志检测数据竞争
go test -tags=integration -race -v ./tests/...
```

**数据竞争场景**：
- 多 goroutine 并发读写同一 Session
- 未使用 mutex 保护的统计字段
- Repository 并发调用

#### 面试追问

| 问题 | 考察点 | 回答要点 |
|------|--------|----------|
| 为什么不用 table-driven test？ | 测试风格 | 表驱动适合同质化测试，集成测试需独立环境 |
| `-race` 检测原理？ | Go 运行时 | 插桩检测 memory access patterns |
| mock vs real？ | 测试策略 | 单元测试用 mock，集成测试用 real |

---

### 3.5 Session 与 Agent 架构

#### AppendEvent 业务逻辑

```go
// session_repository.go 第 280-324 行
func (r *PostgresSessionRepository) AppendEvent(ctx context.Context,
    id string, event *model.Event) error {

    // 1. 补全事件元数据
    if event.ID == "" {
        event.ID = uuid.New().String()
    }
    if event.CreatedAt.IsZero() {
        event.CreatedAt = time.Now()
    }

    // 2. 从事件中提取最新消息，判断是否 assistant 回复
    var message string
    isAssistantReply := false
    if event.Type == model.EventTypeMessage {
        var msgEvent model.MessageEvent
        if err := sonic.Unmarshal(event.Data, &msgEvent); err == nil {
            message = msgEvent.Message
            isAssistantReply = msgEvent.Role == "assistant"  // 只计 assistant
        }
    }

    // 3. 原子更新：追加事件 + 最新消息 + 未读计数
    query := `
        UPDATE sessions SET
            events = events || $2::jsonb,                    -- 追加事件
            latest_message = CASE WHEN $3 != '' THEN $3 ELSE latest_message END,
            latest_message_at = CASE WHEN $3 != '' THEN $4 ELSE latest_message_at END,
            unread_message_count = unread_message_count + $5, -- 只有 assistant +1
            updated_at = $4
        WHERE id = $1
    `
    _, err := q.Exec(ctx, query, id, event.ToJSON(), message, timestamp, unreadDelta)
    return err
}
```

**设计要点**：

1. **事件回填**：上游只设置 Type/Data，补全 ID 和 CreatedAt
2. **角色判断**：从 `MessageEvent.Role` 判断是否是 assistant
3. **未读计数逻辑**：用户发消息不算未读，assistant 回复才算
4. **原子更新**：一次 SQL 完成多个字段更新

#### MessageEvent 结构

```go
// model/event.go 第 92-106 行
type MessageEvent struct {
    Type        EventType `json:"type"`
    Role        string    `json:"role"`     // user / assistant
    Message     string    `json:"message"`  // 消息内容
    Attachments []File    `json:"attachments,omitempty"`
}
```

#### Agent Memory 机制

```go
// agent/memory.go
type Memory interface {
    Add(msg llmcore.Message) error
    MergeMessages(msgs []llmcore.Message) error
    GetMessages() []llmcore.Message
    Clear()
    Compact(keepCount int) error
}

type SimpleMemory struct {
    mu       sync.RWMutex
    messages []llmcore.Message
    maxSize  int
}
```

**Memory 持久化到数据库**：

```go
// session_repository.go 第 326-346 行
func (r *PostgresSessionRepository) GetMemory(ctx context.Context,
    id string, agentName string) ([]llmcore.Message, error) {

    query := `SELECT memories->>$2 FROM sessions WHERE id = $1`
    // memories 是 JSONB：{"PlannerAgent": [...], "ReactAgent": [...]}
}

// session_repository.go 第 348-366 行
func (r *PostgresSessionRepository) SaveMemory(ctx context.Context,
    id string, agentName string, messages []llmcore.Message) error {

    query := `
        UPDATE sessions SET
            memories = JSONB_SET(COALESCE(memories, '{}'::jsonb), $2::text[], $3),
            updated_at = NOW()
        WHERE id = $1
    `
    // memories = JSONB_SET(memories, '{PlannerAgent}', '[...]')
}
```

**JSONB 设计**：
```
memories = {
    "PlannerAgent": [
        {"role": "user", "content": "..."},
        {"role": "assistant", "content": "..."}
    ],
    "ReactAgent": [...]
}
```

#### 面试追问

| 问题 | 考察点 | 回答要点 |
|------|--------|----------|
| 为什么用 JSONB 而不是关系表？ | 灵活性 vs 查询能力 | 动态 schema，per-agent 独立存储 |
| 未读数为什么用字段而不是计算？ | 性能 | 避免每次 COUNT，未读变化频繁 |
| JSONB 如何建索引？ | PostgreSQL | `CREATE INDEX ON sessions USING GIN (memories)` |

---

### 3.6 文件清理服务

#### 业务规则：孤儿文件判定

```go
// file_repository.go 第 219-255 行
func (r *PostgresFileRepository) GetExpiredFiles(ctx context.Context,
    expireDuration string, limit int64) ([]*model.File, error) {

    query := `
        SELECT f.id, f.session_id, ...
        FROM files f
        LEFT JOIN sessions s ON f.session_id = s.id
        WHERE f.created_at < NOW() - $1::interval
          AND (f.session_id IS NULL OR s.deleted_at IS NOT NULL)
        ORDER BY f.created_at ASC
        LIMIT $2
    `
}
```

**清理条件**：
1. `session_id IS NULL`：未关联会话的文件（异常数据）
2. `s.deleted_at IS NOT NULL`：关联会话已被软删除

**不会清理**：关联活跃会话的文件

#### 清理服务架构

```go
// file_cleanup_service.go 第 66-111 行
func (s *DefaultFileCleanupService) CleanExpiredFiles(
    ctx context.Context, expireDuration string, batchSize int) (int, error) {

    s.cleanupMu.Lock()           // 防止并发清理
    defer s.cleanupMu.Unlock()

    totalCleaned := 0
    for {
        files, err := s.repo.GetExpiredFiles(ctx, expireDuration, int64(batchSize))
        if len(files) == 0 {
            break
        }

        // 1. 从 MinIO 删除文件
        for _, file := range files {
            s.storage.Delete(ctx, file.Key)
        }

        // 2. 从数据库删除记录
        s.repo.DeleteByIDs(ctx, fileIDs)

        totalCleaned += len(files)
    }

    return totalCleaned, nil
}
```

#### 调度器设计

```go
// file_cleanup_service.go 第 206-285 行
type FileCleanupScheduler struct {
    cleanupService FileCleanupService
    expireDuration string
    batchSize      int
    interval       time.Duration
    stopChan       chan struct{}
}

func (s *FileCleanupScheduler) Start() {
    s.wg.Add(1)
    go s.run()
}

func (s *FileCleanupScheduler) run() {
    defer s.wg.Done()

    // 首次清理延迟（等待系统启动）
    timer := time.NewTimer(firstCleanupDelay)
    defer timer.Stop()

    for {
        select {
        case <-s.stopChan:
            return
        case <-timer.C:
            s.cleanup()
            timer.Reset(s.interval)
        }
    }
}
```

**设计要点**：
1. **首次延迟**：系统启动后等待一段时间再清理
2. **固定间隔**：周期性清理
3. **优雅关闭**：`stopOnce` + `stopChan` + `WaitGroup`

#### 面试追问

| 问题 | 考察点 | 回答要点 |
|------|--------|----------|
| 为什么先删 MinIO 再删数据库？ | 数据一致性 | 反过来会导致数据库有记录但文件不存在 |
| 清理失败了怎么办？ | 错误处理 | 不删数据库记录，下次重试；记录日志 |
| 为什么不一次性删除所有？ | 资源控制 | 避免长时间事务、大量 I/O |

---

## 4. 可能的面试追问

### 4.1 Go 语言层面

| 问题 | 考察点 | 参考回答 |
|------|--------|----------|
| `sync.Mutex` vs `sync.RWMutex`？ | 并发控制 | 读多写少用 RWMutex，读写都加写锁 |
| `context.WithTimeout` 实现原理？ | Context | 基于 `deadline` + `select` + `timer` |
| `defer` 在 `panic` 时的执行顺序？ | 错误处理 | 按 LIFO 顺序执行，常用于清理 |
| 为什么用 `interface{}` 而不用泛型？ | Go 版本 | 项目用 Go 1.21，之前需要类型断言 |
| `pgx.Row.Scan` 为什么会返回 `pgx.ErrNoRows`？ | 数据库驱动 | pgx 在扫描前检查，没有行就返回 sentinel error |

### 4.2 数据库层面

| 问题 | 考察点 | 参考回答 |
|------|--------|----------|
| JSONB vs JSON 区别？ | PostgreSQL | JSONB 预解析、可建索引、查询更快 |
| `JSONB_SET` 语法？ | PostgreSQL | `JSONB_SET(jsonb, path, value)` path 是 text[] |
| `COALESCE` 作用？ | PostgreSQL | 返回第一个非 NULL 值，`COALESCE(a, '{}')` |
| LEFT JOIN + WHERE 技巧？ | SQL 优化 | `f.session_id IS NULL OR s.deleted_at IS NOT NULL` |
| `ORDER BY f.created_at ASC LIMIT $2`？ | 分页策略 | 早期文件优先清理，先进先出 |

### 4.3 架构设计层面

| 问题 | 考察点 | 参考回答 |
|------|--------|----------|
| Repository 模式的优缺点？ | 架构 | 优点：测试友好、数据源透明；缺点：样板代码多 |
| 为什么用 `WithTx` 封装事务？ | 封装性 | 业务代码不感知事务存在，统一入口 |
| 事件存储 vs 快照存储？ | 数据模型 | 事件存储适合追加写，快照适合读取 |
| 为什么用 `deleted_at IS NOT NULL` 而不是 `WHERE deleted_at = NULL`？ | SQL 语义 | NULL 不等于任何东西，必须用 IS (NOT) NULL |

### 4.4 测试层面

| 问题 | 考察点 | 参考回答 |
|------|--------|----------|
| `//go:build integration` vs `-tags`？ | 构建标签 | 前者是 Go 1.17+ 标准，后者是兼容写法 |
| docker-compose 如何保证测试隔离？ | 测试环境 | 独立 DB、端口，独立 volume |
| 为什么用 `httptest.ResponseRecorder`？ | HTTP 测试 | 拦截响应，不需要真实网络 |
| `-race` 能检测什么？ | 竞态检测 | 读写冲突、double-close、未同步的并发访问 |

---

## 5. 注意事项和踩坑点

### 5.1 Go 错误处理

```go
// ⚠️ 常见错误：直接比较不同包的 sentinel error
if err == sql.ErrNoRows {  // ❌ 永远 false
    // ...
}

// ✅ 正确做法
if errors.Is(err, pgx.ErrNoRows) {  // 或 sql.ErrNoRows
    // ...
}

// ⚠️ 常见错误：忽略 errors.Is
if err != nil {
    return fmt.Errorf("query failed: %v", err)  // ❌ 丢失错误链
}

// ✅ 正确做法
if err != nil {
    return fmt.Errorf("query failed: %w", err)  // ✅ 包装错误
}
```

### 5.2 PostgreSQL 参数化

```go
// ⚠️ 常见错误：INTERVAL 不能参数化
query := `WHERE created_at < NOW() - INTERVAL $1`  // ❌

// ✅ 正确做法：类型转换
query := `WHERE created_at < NOW() - $1::interval`

// ⚠️ 常见错误：text[] 数组参数
query := `WHERE id = ANY($1)`  // ❌ $1 需要是数组类型

// ✅ 正确做法：直接传 []string
rows, err := q.Query(ctx, query, []string{"a", "b"})
```

### 5.3 Repository 设计

```go
// ⚠️ 常见错误：在事务方法中又开启事务
func (r *Repo) WithTx(ctx context.Context, fn func(repo Repo) error) error {
    if r.tx != nil {
        return fn(r)  // ✅ 检测嵌套
    }
    tx, err := r.db.Begin(ctx)
    // ...
}
```

### 5.4 软删除查询

```go
// ⚠️ 常见错误：忘记软删除条件
query := `SELECT * FROM sessions WHERE id = $1`  // ❌ 可能查到已删除记录

// ✅ 正确做法
query := `SELECT * FROM sessions WHERE id = $1 AND deleted_at IS NULL`
```

### 5.5 文件清理

```go
// ⚠️ 常见错误：只删数据库不删文件
s.repo.DeleteByIDs(ctx, fileIDs)  // ❌ MinIO 文件还在

// ✅ 正确做法：先删文件再删记录
s.storage.Delete(ctx, file.Key)
s.repo.DeleteByIDs(ctx, fileIDs)
```

---

## 附录：关键代码位置

| 功能 | 文件位置 | 关键行号 |
|------|----------|----------|
| errors.Is 正确用法 | `session_repository.go` | 145 |
| INTERVAL 类型转换 | `file_repository.go` | 231 |
| 软删除查询 | `session_repository.go` | 134 |
| 孤儿文件判定 | `file_repository.go` | 230-232 |
| AppendEvent | `session_repository.go` | 280-324 |
| Memory 持久化 | `session_repository.go` | 326-366 |
| 文件清理调度 | `file_cleanup_service.go` | 206-285 |
| 集成测试 TestMain | `tests/integration_test.go` | 50-87 |

---

> 本文档基于 go-manus 项目真实代码编写，最后更新：2026-09-10
