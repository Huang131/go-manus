# go-manus - 通用 AI Agent 系统

go-manus 是一个通用的 AI Agent 系统，使用Go + Gin，支持完全私有化部署，使用 A2A + MCP 连接 Agent/Tool，同时支持在沙箱中运行各种内置工具和操作。

## 核心概念

### 1. Agent 架构

本项目实现了 **ReAct (Reasoning + Acting)** 模式的 Agent：

```
用户输入 → 规划 Agent → 执行计划
                      ↓
                  ReAct Agent → 工具调用 → 观察结果
                      ↓
                  循环直到完成
```

**核心模块**：
- `planner_agent.go` - 任务规划 Agent，将复杂任务分解为步骤
- `react_agent.go` - 执行 Agent，循环调用工具完成每个步骤
- `planner_react_flow.go` - 流程编排，协调规划与执行

### 2. A2A (Agent to Agent)

Agent 之间的通信协议，支持多 Agent 协作：

```go
// 核心接口定义
type Agent interface {
    Invoke(ctx context.Context, message *model.Message) (*model.Message, error)
}
```

**特点**：
- 支持 Agent 发现和注册
- 支持同步/异步消息传递
- 支持任务委托和结果聚合

### 3. MCP (Model Context Protocol)

大模型上下文协议，标准化工具调用：

```go
type Tool interface {
    GetDefinition() ToolDefinition  // 获取工具定义
    Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error)
}
```

**内置工具**：
- 文件操作 (`tool_file.go`)
- Shell 命令 (`tool_shell.go`)
- 浏览器控制 (`tool_browser.go`)
- 搜索功能 (`tool_search.go`)
- 消息通知 (`tool_message.go`)
- A2A 通信 (`tool_a2a.go`)
- MCP 集成 (`tool_mcp.go`)

### 4. 沙箱环境

隔离的代码执行环境，支持：
- 安全执行用户代码
- 文件系统操作
- Shell 命令执行
- VNC 远程桌面

---

## 学习路径

### 第一阶段：理解 Agent 基础

1. 阅读 [Agent 核心架构](./docs/AGENT_ARCHITECTURE.md)
2. 理解 ReAct 循环：`api/internal/agent/react_agent.go`
3. 理解规划流程：`api/internal/agent/planner_react_flow.go`

### 第二阶段：掌握工具系统

1. 阅读 [工具系统文档](./docs/TOOL_SYSTEM.md)
2. 学习工具定义和注册机制：`api/internal/agent/tools.go`
3. 实践自定义工具开发

### 第三阶段：理解外部集成

1. LLM 接口抽象：`api/internal/external/llm.go`
2. 沙箱服务通信：`api/internal/external/sandbox.go`
3. 浏览器自动化：`api/internal/external/browser.go`

### 第四阶段：部署和扩展

1. 阅读 [部署指南](./docs/DEPLOYMENT.md)
2. 理解服务架构和依赖关系
3. 根据需求扩展功能

## 项目结构

```
go-manus/
├── api/                    # 后端 API 服务（Go + Gin）
│   ├── cmd/server/main.go  # 入口文件
│   ├── config/             # 配置加载
│   ├── internal/           # 内部包
│   │   ├── agent/          # ⭐ Agent 核心逻辑
│   │   │   ├── base.go     # Agent 基类
│   │   │   ├── config.go   # Agent 配置
│   │   │   ├── flow.go     # 流程接口定义
│   │   │   ├── memory.go   # 记忆系统
│   │   │   ├── planner_agent.go     # 规划 Agent
│   │   │   ├── react_agent.go       # ReAct 执行 Agent
│   │   │   ├── planner_react_flow.go # 规划-执行流程
│   │   │   ├── tool_*.go   # 工具实现
│   │   │   └── ...
│   │   ├── external/       # 外部服务接口
│   │   │   ├── llm.go      # LLM 接口抽象
│   │   │   ├── sandbox.go  # 沙箱通信
│   │   │   └── browser.go  # 浏览器自动化
│   │   ├── handler/        # HTTP 处理器
│   │   ├── service/        # 业务逻辑层（当前主要逻辑在 agent/）
│   │   ├── repository/     # 数据访问层
│   │   ├── router/         # 路由定义
│   │   └── model/          # 数据模型
│   ├── migrations/         # 数据库迁移脚本
│   └── pkg/                # 公共工具包
├── ui/                     # 前端服务（Next.js）
├── sandbox/                # 沙箱服务（Python FastAPI）
├── nginx/                  # Nginx 网关配置（含 conf.d）
├── docs/                   # 技术文档
├── docker-compose.yml      # 主部署配置（含 nginx 网关）
├── docker-compose.dev.yml  # 旧版简化配置（已弃用，仅作参考）
└── Makefile
```

### 核心代码指引

| 文件 | 作用 | 学习优先级 |
|------|------|-----------|
| `agent/react_agent.go` | ReAct 执行循环 | ⭐⭐⭐⭐⭐ |
| `agent/planner_agent.go` | 任务规划 | ⭐⭐⭐⭐⭐ |
| `agent/planner_react_flow.go` | 流程编排 | ⭐⭐⭐⭐ |
| `agent/tools.go` | 工具注册中心 | ⭐⭐⭐⭐ |
| `agent/tool_*.go` | 具体工具实现 | ⭐⭐⭐ |
| `external/llm.go` | LLM 接口 | ⭐⭐⭐⭐ |

## 快速部署

### 前置要求

- Docker >= 20.10
- Docker Compose >= 2.0

### 一键部署（推荐）

所有服务通过 **Nginx 网关统一暴露**（与旧项目 mooc-manus 一致），只暴露 80 端口，
避免 CORS、SSE/WebSocket 反代、前端相对路径等坑。

1. **复制环境变量配置**

   ```bash
   # 本地开发（使用 Docker 内置数据库）
   cp api/.env.example .env

   # 或使用远程测试环境资源（参考 .env.test）
   cp .env.test .env
   ```

2. **配置环境变量**

   编辑 `.env` 文件：

   ```bash
   # 必须修改的配置
   COS_SECRET_ID=your_cos_secret_id_here       # 腾讯云 COS SecretId（或 MinIO 占位）
   COS_SECRET_KEY=your_cos_secret_key_here     # 腾讯云 COS SecretKey（或 MinIO 占位）
   COS_BUCKET=your_cos_bucket_here             # COS 存储桶名称

   # LLM 配置
   LLM_BASE_URL=https://api.openai.com         # 或其他兼容 API
   LLM_API_KEY=your_api_key_here
   LLM_MODEL_NAME=gpt-4
   ```

3. **启动所有服务**

   ```bash
   # 一键启动（含 nginx、postgres、redis、minio、api、sandbox、ui）
   docker-compose up -d --build

   # 或使用 Makefile
   make up
   ```

4. **访问系统**

   - 🌐 **统一入口（推荐）**：`http://localhost/`
     - `/` → UI（Next.js）
     - `/api/*` → API（Go + Gin）
     - `/sandbox/*` → Sandbox（Python）
   - 直连端口（调试用）：
     - UI：`http://localhost:3000`
     - API：`http://localhost:8080`
     - Sandbox：`http://localhost:8090`
     - MinIO 控制台：`http://localhost:9001`（minioadmin/minioadmin）

> **重要**：浏览器侧请求必须走 80 端口的 nginx，UI 内部已写死 `NEXT_PUBLIC_API_BASE_URL=/api`，
> 直连 8080/3000 会因 `/api` 前缀缺失导致 404。

### 服务架构（Nginx 网关模式）

```
                          ┌─────────────────────────────────────────────┐
                          │              Docker Network                 │
                          │                                             │
   ┌────────┐  :80        │  ┌──────────┐    ┌──────────┐  ┌──────────┐ │
   │ Browser│────────────►│  │  nginx   │───►│  ui:3000 │  │ api:8080 │ │
   │        │             │  │ (gateway)│    │  Next.js │  │ Go + Gin │ │
   │        │             │  └─────┬────┘    └──────────┘  └────┬─────┘ │
   │        │             │        │ /api/*                      │       │
   │        │             │        └─────────────────────────────┘       │
   └────────┘             │              │             │                 │
                          │              ▼             ▼                 │
                          │       ┌──────────┐  ┌──────────┐  ┌──────────┐│
                          │       │ postgres │  │  redis   │  │ minio    ││
                          │       │  :5432   │  │  :6379   │  │ :9000    ││
                          │       └──────────┘  └──────────┘  └──────────┘│
                          │                              ┌──────────┐    │
                          │                              │ sandbox  │    │
                          │                              │  :8090   │    │
                          │                              └──────────┘    │
                          └─────────────────────────────────────────────┘
```

### 容器列表

| 容器名称 | 服务 | 内部端口 | 外部端口 |
|---------|------|----------|----------|
| go-manus-nginx | Nginx（网关） | 80 | **80**（统一入口）|
| go-manus-ui | Next.js | 3000 | 3000（直连调试）|
| go-manus-api | Go + Gin | 8080 | 8080（直连调试）|
| go-manus-sandbox | Python FastAPI | 8080 | 8090（直连调试）|
| go-manus-postgres | PostgreSQL | 5432 | 5432 |
| go-manus-redis | Redis | 6379 | 6379 |
| go-manus-minio | MinIO | 9000 | 9000 / 9001（控制台）|

### 测试环境使用

如果需要连接远程测试环境资源（远程 PG/Redis/MinIO），参考 `.env.test` 文件：

```bash
# 方式1：直接使用 .env.test
cp .env.test .env
docker-compose up -d

# 方式2：本地运行 API（连接远程数据库，不走 docker）
cd api
go run cmd/server/main.go
```

**注意**：
- `.env.test` 包含敏感信息，已在 `.gitignore` 中忽略，不会提交到 Git
- 当前项目仅支持 PostgreSQL 数据库
- ES 配置已预留，但暂未集成到代码中

## 常用命令

```bash
# 启动所有服务
make up
# 或
docker-compose up -d --build

# 停止所有服务
make down
# 或
docker-compose down

# 查看服务状态
make ps

# 查看所有服务日志
make logs

# 查看特定服务日志
make logs-api      # API 日志
make logs-ui       # UI 日志
make logs-nginx    # Nginx 日志
make logs-sandbox  # 沙箱日志

# 重启所有服务
make restart

# 重启特定服务
make rebuild-api   # 重建 API 服务

# 进入容器
make shell-api     # 进入 API 容器
make shell-sandbox # 进入沙箱容器

# 清理 Docker 资源
make clean         # 清理未使用的资源
make clean-all     # 完全清理（谨慎）
```

## 本地开发

### API 开发

```bash
cd api

# 编译
make build

# 运行测试
make test

# 代码检查
make lint

# 本地运行（需要 Postgres + Redis）
make run
```

### 健康检查

```bash
make health
```

## API 文档

启动服务后访问：
- API 健康检查: `http://localhost:8080/api/v1/status`
- API 直接访问: `http://localhost:8080`
- **VNC 远程桌面**（WebSocket）: `ws://localhost:8080/api/sessions/:id/vnc`
  - 前端"虚拟机浏览器"按钮连接此端点，通过 sandbox 内 websockify 转发到 VNC (:5901)


## 许可证

MIT License