# ============================================================
# go-manus API Makefile - Go 代码开发和测试
# ============================================================

.PHONY: help build test test-cover lint clean run fmt deps deps-view tidy health docker-build

# 默认目标
help:
	@echo "go-manus API 开发命令"
	@echo ""
	@echo "=== 构建 ==="
	@echo "  make build         - 编译 Go 程序"
	@echo "  make tidy          - 整理依赖 (清理/下载)"
	@echo ""
	@echo "=== 测试 ==="
	@echo "  make test          - 运行所有测试"
	@echo "  make test-cover    - 运行测试 (生成覆盖率报告)"
	@echo ""
	@echo "=== 代码质量 ==="
	@echo "  make lint          - 代码检查 (go vet + golangci-lint)"
	@echo "  make fmt           - 格式化代码"
	@echo ""
	@echo "=== 本地运行 ==="
	@echo "  make run           - 本地运行服务 (需要 Postgres + Redis)"
	@echo "  make health        - 检查服务健康状态"
	@echo ""
	@echo "=== 依赖管理 ==="
	@echo "  make deps          - 下载并整理依赖"
	@echo "  make deps-view     - 查看依赖关系图"
	@echo ""
	@echo "=== Docker ==="
	@echo "  make docker-build  - 构建 API Docker 镜像"
	@echo ""
	@echo "=== 清理 ==="
	@echo "  make clean         - 清理构建产物"

# ============================================================
# 构建命令
# ============================================================

# 编译 Go 程序
build:
	@echo "编译 Go 程序..."
	go build -ldflags="-s -w" -trimpath -o bin/go-manus ./cmd/server

# ============================================================
# 测试命令
# ============================================================

# 运行测试
test:
	@echo "运行测试..."
	go test -v -cover ./...

# 带覆盖率测试
test-cover:
	@echo "运行测试 (覆盖率)..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

# ============================================================
# 代码质量
# ============================================================

# 代码检查
lint:
	@echo "运行代码检查..."
	go vet ./...
	golangci-lint run 2>/dev/null || echo "golangci-lint 未安装，跳过"

# 格式化代码
fmt:
	@echo "格式化代码..."
	go fmt ./...

# ============================================================
# 本地运行
# ============================================================

# 本地运行服务
run:
	@echo "启动本地服务..."
	@echo "请确保 Postgres 和 Redis 已运行"
	go run ./cmd/server

# 检查本地服务健康状态
health:
	@echo "检查服务健康状态..."
	@curl -s http://localhost:8080/health 2>/dev/null | jq '.' || echo "服务未启动"
	@curl -s http://localhost:8080/api/status 2>/dev/null | jq '.' || echo "服务未启动"

# ============================================================
# 依赖管理
# ============================================================

# 整理依赖 (下载缺失 + 清理无用的)
tidy:
	@echo "整理依赖..."
	go mod tidy

# 安装依赖
deps:
	@echo "安装依赖..."
	go mod download
	go mod tidy

# 查看依赖关系
deps-view:
	@echo "依赖关系..."
	go mod graph

# ============================================================
# Docker 命令
# ============================================================

# 构建 API Docker 镜像
docker-build:
	@echo "构建 API Docker 镜像..."
	docker build -t go-manus-api:latest .

# ============================================================
# 清理
# ============================================================

# 清理构建产物
clean:
	@echo "清理构建产物..."
	rm -rf bin/
	rm -f coverage.out coverage.html