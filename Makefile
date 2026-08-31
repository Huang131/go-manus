# ============================================================
# go-manus 一站式运维脚本
# ============================================================
# 标准部署: docker-compose.yml (nginx 网关模式)
# 包含: nginx 网关 + UI (Next.js) + API (Go) + 沙箱 (Python) + postgres + redis + minio
# ============================================================

.PHONY: help

# 默认目标
help:
	@echo "go-manus 一站式运维脚本"
	@echo ""
	@echo "=== 标准部署 (nginx 网关) ==="
	@echo "  make up            - 启动所有服务 (含 nginx 网关)"
	@echo "  make down          - 停止所有服务"
	@echo "  make down-v        - 停止并删除数据卷（慎用）"
	@echo ""
	@echo "=== 服务状态 ==="
	@echo "  make health        - 检查 API 健康状态 + 容器状态"
	@echo ""
	@echo "=== 构建 ==="
	@echo "  make build         - 构建所有 Docker 镜像"
	@echo "  make rebuild-api   - 重建并启动 API (同 rebuild-ui / rebuild-sandbox)"
	@echo "  make restart-api   - 重启指定服务 (同 restart-ui / restart-sandbox)"
	@echo ""
	@echo "=== 日志查看 ==="
	@echo "  make logs          - 所有服务日志"
	@echo "  make logs-api      - API 日志"
	@echo "  make logs-nginx    - nginx 日志 (排查 502 必备)"
	@echo "  make logs-ui       - UI 日志 (同 logs-sandbox / logs-postgres / logs-redis)"
	@echo ""
	@echo "=== 进入容器 ==="
	@echo "  make shell-api     - 进入 API 容器"
	@echo "  make shell-postgres - 进入 PostgreSQL 命令行"
	@echo "  make shell-sandbox - 进入沙箱容器"
	@echo ""
	@echo "=== 开发命令 (在 api/ 目录) ==="
	@echo "  cd api && make build    - 编译 Go 程序"
	@echo "  cd api && make test     - 运行测试"
	@echo "  cd api && make lint     - 代码检查"
	@echo ""
	@echo "=== 清理 ==="
	@echo "  make clean         - 清理未使用的 Docker 资源"
	@echo "  make down-v        - 停止并删除数据卷（慎用）"
	@echo ""

# ============================================================
# 标准部署 (含 nginx 网关)
# ============================================================

# 启动所有服务 (nginx 网关模式)
up:
	@echo "启动所有服务 (nginx 网关模式)..."
	docker-compose up -d
	@echo ""
	@echo "服务已启动!"
	@echo "  - 网关:    http://localhost (端口 80) ← 统一入口"
	@echo "  - API:     http://localhost:8080"
	@echo "  - UI:      http://localhost:3000"
	@echo "  - Sandbox: http://localhost:8090"
	@echo "  - Postgres: localhost:5432"
	@echo "  - Redis:   localhost:6379"
	@echo "  - MinIO:   localhost:9000 (S3 API) / localhost:9001 (控制台)"
	@echo ""
	@echo "查看日志: make logs"

# 停止所有服务
down:
	@echo "停止所有服务..."
	docker-compose down

# 停止并删除数据卷 (完全清理)
down-v:
	@echo "停止所有服务并删除数据..."
	docker-compose down -v

# ============================================================
# 构建命令
# ============================================================

# 构建所有 Docker 镜像
build:
	@echo "构建所有 Docker 镜像..."
	docker-compose build

# 重新构建所有镜像 (不带缓存)
build-no-cache:
	@echo "重新构建所有 Docker 镜像 (不带缓存)..."
	docker-compose build --no-cache

# ============================================================
# 服务状态
# ============================================================

# 查看服务状态 (建议使用 make health 获取更详细的状态)
ps:
	docker-compose ps

# 查看服务健康状态
health:
	@echo "检查服务健康状态..."
	@echo ""
	@echo "=== API /health ==="
	@curl -s http://localhost:8080/health 2>/dev/null | jq '.' || echo "API 服务未启动"
	@echo ""
	@echo "=== API /api/status ==="
	@curl -s http://localhost:8080/api/status 2>/dev/null | jq '.' || echo "API 服务未启动"
	@echo ""
	@echo "=== Docker 服务状态 ==="
	docker-compose ps

# ============================================================
# 日志查看
# ============================================================

# 查看所有服务日志
logs:
	docker-compose logs -f

# 查看特定服务日志
logs-api:
	docker-compose logs -f api

logs-ui:
	docker-compose logs -f ui

logs-sandbox:
	docker-compose logs -f sandbox

logs-postgres:
	docker-compose logs -f postgres

logs-redis:
	docker-compose logs -f redis

logs-nginx:
	docker-compose logs -f nginx

# ============================================================
# 重启服务
# ============================================================

# 重启所有服务
restart:
	@echo "重启所有服务..."
	docker-compose restart

# 重启特定服务
restart-api:
	docker-compose restart api

restart-ui:
	docker-compose restart ui

restart-sandbox:
	docker-compose restart sandbox

# ============================================================
# 进入容器
# ============================================================

# 进入 API 容器
shell-api:
	docker exec -it go-manus-api /bin/sh

# 进入沙箱容器
shell-sandbox:
	docker exec -it go-manus-sandbox /bin/sh

# 进入数据库
shell-postgres:
	docker exec -it go-manus-postgres psql -U postgres -d manus

# ============================================================
# 清理
# ============================================================

# 清理未使用的 Docker 资源
clean:
	@echo "清理未使用的 Docker 资源..."
	docker system prune -f
	@echo "清理完成!"

# 完整清理 (包括镜像)
clean-all:
	@echo "完整清理 Docker 资源 (删除 go-manus 所有镜像)..."
	docker-compose down --rmi local
	docker system prune -f
	@echo "清理完成!"

# ============================================================
# 重建特定服务
# ============================================================

# 重建并启动特定服务
rebuild-api:
	@echo "重建 API 服务..."
	docker-compose up -d --build api

rebuild-ui:
	@echo "重建 UI 服务..."
	docker-compose up -d --build ui

rebuild-sandbox:
	@echo "重建沙箱服务..."
	docker-compose up -d --build sandbox