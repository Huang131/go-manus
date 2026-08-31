# ============================================================
# go-manus 一站式运维脚本
# ============================================================
# 包含: API (Go) + 沙箱服务 (Python) + 前端 (Next.js)
# ============================================================

.PHONY: help

# 默认目标
help:
	@echo "go-manus 一站式运维脚本"
	@echo ""
	@echo "=== 本地开发 (推荐) ==="
	@echo "  make dev-up        - 启动所有服务 (简化版，无需 nginx)"
	@echo "  make dev-down      - 停止所有服务 (简化版)"
	@echo ""
	@echo "=== 生产部署 (通过 nginx 网关) ==="
	@echo "  make up            - 启动所有服务 (含 nginx 网关)"
	@echo "  make down          - 停止所有服务"
	@echo ""
	@echo "=== 服务管理 ==="
	@echo "  make build         - 构建所有 Docker 镜像"
	@echo "  make logs          - 查看所有服务日志"
	@echo "  make ps            - 查看服务状态"
	@echo "  make restart       - 重启所有服务"
	@echo ""
	@echo "=== 单独服务日志 ==="
	@echo "  make logs-api      - API 服务日志"
	@echo "  make logs-ui       - UI 服务日志"
	@echo "  make logs-sandbox  - 沙箱服务日志"
	@echo "  make logs-postgres - 数据库日志"
	@echo "  make logs-redis    - Redis 日志"
	@echo ""
	@echo "=== 进入容器 ==="
	@echo "  make shell-api     - 进入 API 容器"
	@echo "  make shell-sandbox - 进入沙箱容器"
	@echo ""
	@echo "=== 开发命令 (在 api/ 目录) ==="
	@echo "  cd api && make build    - 编译 Go 程序"
	@echo "  cd api && make test     - 运行测试"
	@echo "  cd api && make lint     - 代码检查"
	@echo ""
	@echo "=== 清理 ==="
	@echo "  make clean         - 清理未使用的 Docker 资源"
	@echo ""

# ============================================================
# 本地开发 (简化版，无需 nginx)
# ============================================================

# 启动所有服务 (本地开发版)
dev-up:
	@echo "启动所有服务 (本地开发版)..."
	docker-compose -f docker-compose.dev.yml up -d
	@echo ""
	@echo "服务已启动!"
	@echo "  - API:      http://localhost:8080"
	@echo "  - API 文档: http://localhost:8080/health"
	@echo "  - UI:       http://localhost:3000"
	@echo "  - Sandbox:  http://localhost:8090"
	@echo "  - Postgres: localhost:5432"
	@echo "  - Redis:    localhost:6379"
	@echo ""
	@echo "查看日志: make dev-logs"
	@echo "停止服务: make dev-down"

# 停止所有服务 (本地开发版)
dev-down:
	@echo "停止所有服务 (本地开发版)..."
	docker-compose -f docker-compose.dev.yml down

# 查看本地开发服务日志
dev-logs:
	docker-compose -f docker-compose.dev.yml logs -f

# 查看特定服务日志 (本地开发版)
dev-logs-api:
	docker-compose -f docker-compose.dev.yml logs -f api

dev-logs-ui:
	docker-compose -f docker-compose.dev.yml logs -f ui

# ============================================================
# 生产部署 (含 nginx 网关)
# ============================================================

# 启动所有服务 (生产版)
up:
	@echo "启动所有服务 (生产版)..."
	docker-compose up -d
	@echo ""
	@echo "服务已启动!"
	@echo "  - 网关:    http://localhost (端口 80)"
	@echo "  - API:     http://localhost:8080"
	@echo "  - UI:      http://localhost:3000"
	@echo "  - Postgres: localhost:5432"
	@echo "  - Redis:   localhost:6379"
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

# 查看服务状态
ps:
	docker-compose ps

# 查看服务健康状态
health:
	@echo "检查服务健康状态..."
	@echo ""
	@echo "=== API 健康检查 ==="
	@curl -s http://localhost:8080/api/v1/status 2>/dev/null | jq '.' || echo "API 服务未启动"
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
	@echo "完整清理 Docker 资源..."
	@read -p "确定要删除所有 go-manus 相关镜像吗? (yes/no): " confirm; \
	if [ "$$confirm" = "yes" ]; then \
		docker-compose down --rmi local; \
		docker system prune -f; \
		echo "清理完成!"; \
	else \
		echo "取消操作"; \
	fi

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