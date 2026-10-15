# Docker 构建优化实践笔记

> 时间：2026-09-12
> 项目：go-manus（AI 代码智能体平台）
> 优化对象：API / UI / Sandbox 三个服务的 Docker 构建系统

---

## 背景

go-manus 是一个由 Go 后端 + Next.js 前端 + Python 沙箱组成的多服务应用。在日常开发中，频繁修改代码后需要重新构建 Docker 镜像，但原始构建存在以下问题：

1. **构建慢**：每次 rebuild 即使只改了几行代码，也要重新下载依赖、重新构建基础层
2. **缓存失效**：Go/Node/Python 依赖频繁重新下载，浪费时间
3. **跨平台问题**：Apple Silicon Mac 使用 Docker Desktop 时，硬编码的 `GOARCH=amd64` 导致交叉编译问题
4. **缺少健康检查**：服务间依赖缺少健康检查机制，可能启动顺序不当
5. **安全风险**：sandbox 服务使用了不必要的 `privileged` 模式和 `docker.sock` 挂载

---

## 优化成果

| 服务 | 优化前首次构建 | 优化前增量构建 | 优化后首次构建 | 优化后增量构建 |
|------|----------------|----------------|----------------|----------------|
| API (Go) | ~170s | ~170s | ~170s | **2.5s** |
| UI (Next.js) | ~120s | ~120s | ~120s | **10s** |
| Sandbox (Python) | ~13min | ~13min | ~13min | **2.3s** |

**核心指标提升**：增量构建从分钟级降到秒级，开发体验显著改善。

---

## 一、API 服务（Go）优化

### 1.1 问题分析

#### 问题 1：Cache Mount 分散导致缓存失效

原始 Dockerfile 将 `go mod download` 和 `go build` 分成两个 RUN：

```dockerfile
# 原始写法
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o go-manus ./cmd/server
```

**Why：** Docker 层缓存机制中，每个 RUN 指令形成独立层。第一次构建后，`go mod download` 层被缓存，但 `go build` 层失效时，整个层重新执行，导致第二次构建仍需重新下载依赖。

**Trade-off：**
- 分离 RUN：便于调试，每层失败可快速定位
- 合并 RUN：利用 BuildKit 缓存机制，减少依赖下载

#### 问题 2：硬编码 GOARCH 导致 Apple Silicon 构建失败

```dockerfile
# 原始写法
GOARCH=amd64  # 硬编码
```

**Why：** Apple Silicon Mac 使用 ARM 架构，但 Docker Desktop 默认构建 AMD64 镜像。硬编码 `amd64` 在某些场景下会导致 `exec format error`。

**Alternatives：**
- 使用 `ARG TARGETARCH` 动态获取平台架构
- 使用 `docker buildx bake` 定义多平台构建

#### 问题 3：冗余文件拷贝

```dockerfile
# 原始写法
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=builder /build/pkg ./pkg  # 静态编译不需要
```

**Why：** Go 静态编译不依赖 `pkg` 目录，拷贝是无效操作。

### 1.2 解决方案

```dockerfile
ARG TARGETARCH
WORKDIR /build
COPY go.mod go.sum ./
COPY . .

# 合并 cache mount，减少依赖下载
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} go build \
    -ldflags="-s -w" \
    -o go-manus \
    ./cmd/server
```

**How：**
1. 使用 `--mount=type=cache` 挂载持久化缓存，BuildKit 会缓存 `/go/pkg/mod` 和 `/root/.cache/go-build`
2. 合并 `go mod download` 到 `go build` 命令，减少 RUN 层
3. 使用 `ARG TARGETARCH` + `${TARGETARCH:-amd64}` 动态适配架构

**Why this works：**
- BuildKit 的 cache mount 是跨构建持久化的，不像 Docker 层缓存那样受层失效影响
- 依赖文件和编译缓存被持久化到宿主机，后续构建直接命中缓存
- 单个 RUN 层意味着只要代码变化，依赖缓存层就不会失效

### 1.3 验证结果

```bash
# 首次构建：~170s
# 修改代码后增量构建：2.5s（仅编译步骤，无依赖下载）
```

---

## 二、UI 服务（Next.js）优化

### 2.1 问题分析

#### 问题：Makefile 构建命令冗余

原始 Makefile 使用 `docker compose build` + `docker compose up`：

```makefile
rebuild-api:
	docker compose build api
	docker compose up -d api
```

**Why：**
- `docker compose build` 会构建所有依赖服务的镜像（浪费时间）
- `docker compose up` 会在构建后启动所有依赖服务（可能冲突）

### 2.2 解决方案

```makefile
rebuild-ui:
	@echo "重建 UI 服务..."
	docker buildx build --load -f ui/Dockerfile -t go-manus-ui:latest ui
	docker compose up -d --no-deps ui
```

**How：**
1. 使用 `docker buildx build --load` 直接构建指定镜像，不触发依赖构建
2. 使用 `docker compose up -d --no-deps` 仅启动当前服务，不启动依赖

**Why this works：**
- `docker buildx build` 是独立构建命令，不受 compose 文件中 `depends_on` 约束
- `--no-deps` 参数告诉 compose 只启动指定服务，跳过依赖服务

### 2.3 验证结果

```bash
# 首次构建：~120s（npm ci 36s + next build 70s）
# 增量构建：~10s（仅 next build 步骤）
```

---

## 三、Sandbox 服务（Python）优化

### 3.1 问题分析

#### 问题 1：pip 安装慢

原始 Dockerfile 使用 `pip3 install`：

```dockerfile
RUN pip3 install -r requirements.txt
```

**Why：** pip 默认从 PyPI 下载，依赖解析慢，且没有锁文件机制。`requirements.txt` 顺序变化会导致依赖重新解析。

**Alternatives：**
- 使用 `pip-compile` 生成 `requirements.txt` 锁文件
- 使用 `uv sync --frozen`（更快的方案）
- 使用预编译 wheel 镜像

#### 问题 2：缺少健康检查

原始配置没有健康检查，API 服务通过 `depends_on` 判断 sandbox 是否就绪：

```yaml
# 原始配置
sandbox:
  # 无 healthcheck
api:
  depends_on:
    - sandbox  # 仅检查容器启动，不检查服务就绪
```

**Why：** Docker Compose 的 `depends_on` 只检查容器是否启动，不检查服务是否就绪（如数据库连接、端口监听）。sandbox 服务可能启动了但 FastAPI 还未完全初始化。

**Trade-off：**
- 添加健康检查：增加启动时间（需等待健康检查通过）
- 不添加健康检查：依赖方可能提前连接失败

#### 问题 3：使用不必要的特权模式

```yaml
# 原始配置
sandbox:
  privileged: true
  volumes:
    - /var/run/docker.sock:/var/run/docker.sock
```

**Why：** 搜索代码发现 sandbox 服务并未使用 Docker-in-Docker，`privileged` 和 `docker.sock` 是过度授权，存在安全风险。

#### 问题 4：uvicorn 路径错误

```ini
# 历史 supervisord.conf（当前配置已移除生产固定的 --reload）
command=uvicorn app.main:app --host 0.0.0.0 --port 8080 --reload
environment=HOME=/home/ubuntu,PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
```

**Why：** uv 安装到 `/venv/bin/uvicorn`，但 `PATH` 和 `command` 未更新。

### 3.2 解决方案

#### 3.2.1 使用 uv 替代 pip

```dockerfile
# 从 uv 官方镜像复制工具
COPY --from=ghcr.io/astral-sh/uv:latest /uv /uvx /bin/

WORKDIR /build
COPY pyproject.toml uv.lock ./
RUN uv sync --frozen --no-dev
```

**How：**
1. 从 `ghcr.io/astral-sh/uv:latest` 复制 uv 工具
2. 使用 `uv sync --frozen --no-dev` 安装依赖
3. `--frozen` 强制使用锁文件，`--no-dev` 跳过开发依赖

**Why this works：**
- uv 使用 Rust 实现，比 pip 快 10-100 倍
- `--frozen` 确保依赖版本严格一致，避免重解析

**验证数据：**
```bash
# pip install：34s+
# uv sync --frozen --no-dev：2.3s
```

#### 3.2.2 添加健康检查端点

`sandbox/app/main.py` 添加 `/health` 端点：

```python
@app.get("/health", tags=["健康检查"])
async def health_check():
    """健康检查端点，供 API 服务检测沙箱状态"""
    return {"status": "healthy"}
```

`sandbox/Dockerfile` 添加 HEALTHCHECK：

```dockerfile
HEALTHCHECK --interval=15s --timeout=10s --start-period=10s --retries=3 \
    CMD python3 -c "import urllib.request; urllib.request.urlopen('http://localhost:8080/health')" || exit 1
```

**Why this works：**
- HEALTHCHECK 会周期性地执行检查命令，Docker 根据结果判断容器健康状态
- `--start-period=10s` 给予服务足够初始化时间
- `--retries=3` 容忍短暂抖动

#### 3.2.3 更新 docker-compose.yml

```yaml
sandbox:
  healthcheck:
    test: ["CMD-SHELL", "python3 -c \"import urllib.request; urllib.request.urlopen('http://localhost:8080/health')\""]
    interval: 15s
    timeout: 10s
    retries: 3
    start_period: 10s
  depends_on:
    sandbox:
      condition: service_healthy  # 改为 service_healthy
```

**Why this works：**
- `condition: service_healthy` 要求依赖服务通过健康检查后才启动
- 配合 HEALTHCHECK 形成"服务就绪感知"机制

#### 3.2.4 更新 supervisord.conf

```ini
[program:app]
command=/venv/bin/uvicorn app.main:app --host 0.0.0.0 --port 8080 %(ENV_UVI_ARGS)s
environment=HOME=/home/ubuntu,PATH=/venv/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
```

#### 3.2.5 创建 .dockerignore

```gitignore
__pycache__
*.pyc
*.pyo
*.pyd
.Python
*.so
*.egg
*.egg-info
dist
build
.pytest_cache
.coverage
htmlcov
.env
.venv
node_modules
.git
.gitignore
.DS_Store
```

**Why：** 减少构建上下文大小，加快构建速度。

#### 3.2.6 移除特权模式

```yaml
# 移除
sandbox:
  privileged: true
  volumes:
    - /var/run/docker.sock:/var/run/docker.sock
```

---

## 四、构建缓存管理

### 4.1 查看 buildx 缓存

```bash
docker buildx du
docker buildx inspect <builder-name>
```

### 4.2 清理 buildx 缓存

```bash
# 查看缓存大小
docker buildx du

# 清理超过 24 小时的缓存
docker buildx prune --keep-duration 24h

# 清理所有缓存（谨慎）
docker buildx prune --all

# 限制缓存大小（如保留 10GB）
docker buildx prune --keep-storage 10GB
```

### 4.3 缓存机制说明

**Docker BuildKit 的 cache mount：**
- 挂载到 `/go/pkg/mod` 和 `/root/.cache/go-build`
- 缓存在宿主机，不会随镜像层失效而失效
- 跨构建持久化，直到手动清理

**缓存位置：** 由 buildx 驱动决定，通常在 `~/.docker/buildx` 或 Docker Desktop 的 VM 内部。

---

## 五、面试亮点总结

### 技术深度

1. **Docker BuildKit 高级特性**
   - `cache mount` 类型：跨构建持久化依赖缓存
   - 多平台构建：`ARG TARGETARCH` 动态适配架构
   - 构建效率：从分钟级降到秒级

2. **依赖管理工具选型**
   - pip vs uv 对比：uv 速度快 10-100 倍
   - `--frozen` 锁文件机制：确保构建一致性

3. **服务健康检查机制**
   - Docker HEALTHCHECK + Compose `service_healthy`
   - 端到端的就绪感知，不依赖启动顺序

### 工程意识

1. **安全最小化原则**
   - 移除不必要的 `privileged` 模式
   - 移除不必要的 `docker.sock` 挂载

2. **构建缓存策略**
   - 利用 BuildKit 缓存减少重复下载
   - 合理的层分离和缓存失效控制

3. **开发体验优化**
   - `--no-deps` 避免不必要的依赖构建
   - 增量构建从 170s 降到 2.5s

### 可迁移经验

| 场景 | 解决方案 | 可迁移到 |
|------|----------|----------|
| Go 项目构建慢 | 合并 cache mount | 任何 Go 项目 |
| Python 项目依赖安装慢 | uv sync --frozen | 任何 Python 项目 |
| 服务启动顺序问题 | HEALTHCHECK + service_healthy | 微服务依赖管理 |
| Apple Silicon 构建问题 | ARG TARGETARCH | 多平台 Go 项目 |

---

## 六、相关文件清单

| 文件 | 修改类型 | 说明 |
|------|----------|------|
| `api/Dockerfile` | 修改 | 合并 cache mount，支持多平台 |
| `ui/Dockerfile` | 不变 | 无需修改 |
| `sandbox/Dockerfile` | 修改 | uv sync，添加 HEALTHCHECK |
| `sandbox/app/main.py` | 修改 | 添加 /health 端点 |
| `sandbox/supervisord.conf` | 修改 | 修正 uvicorn 路径 |
| `docker-compose.yml` | 修改 | 添加 healthcheck |
| `Makefile` | 修改 | 改用 buildx build |
| `sandbox/.dockerignore` | 新增 | 减少构建上下文 |
| `sandbox/pyproject.toml` | 不变 | uv 锁文件配置 |
| `sandbox/uv.lock` | 不变 | uv 依赖锁文件 |

---

## 七、后续可优化方向

1. **并行构建**：使用 `docker buildx bake` 并行构建多架构镜像
2. **依赖层缓存**：为 UI 服务添加 npm/yarn 缓存
3. **构建产物分析**：使用 `dive` 工具分析镜像层大小
4. **GitHub Actions 集成**：配置远程构建缓存（cache-from/cache-to）

---

*文档生成时间：2026-09-12*
