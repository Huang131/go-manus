# ============================================================
# go-manus API Makefile - Go 代码开发和测试
# ============================================================

.PHONY: help build test lint clean run docker-build docker-up docker-down docker-logs

# 默认目标
help:
	@echo "go-manus API 开发命令"
	@echo ""
	@echo "构建命令:"
	@echo "  make build         - 编译 Go 程序"
	@echo "  make test          - 运行所有测试"
	@echo "  make test-cover    - 运行测试 (覆盖率报告)"
	@echo "  make lint          - 代码检查"
	@echo "  make clean         - 清理构建产物"
	@echo ""
	@echo "本地运行:"
	@echo "  make run           - 本地运行服务 (需要 Postgres + Redis)"
	@echo ""
	@echo "Docker 命令:"
	@echo "  make docker-build  - 构建 API Docker 镜像"
	@echo ""

# ============================================================
# 构建命令
# ============================================================

# 编译 Go 程序
build:
	@echo "编译 Go 程序..."
	go build -ldflags="-s -w" -o bin/go-manus ./cmd/server

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

# 代码检查
lint:
	@echo "运行代码检查..."
	go vet ./...
	golangci-lint run 2>/dev/null || echo "golangci-lint 未安装，跳过"

# 清理构建产物
clean:
	@echo "清理构建产物..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# ============================================================
# 本地运行
# ============================================================

# 本地运行服务
run:
	@echo "启动本地服务..."
	@echo "请确保 Postgres 和 Redis 已运行"
	go run ./cmd/server

# ============================================================
# Docker 命令
# ============================================================

# 构建 API Docker 镜像
docker-build:
	@echo "构建 API Docker 镜像..."
	docker build -t go-manus-api:latest .

# ============================================================
# 开发命令
# ============================================================

# 安装依赖
deps:
	@echo "安装依赖..."
	go mod download
	go mod tidy

# 格式化代码
fmt:
	@echo "格式化代码..."
	go fmt ./...

# 查看依赖
deps-view:
	@echo "依赖关系..."
	go mod graph

# ============================================================
# 健康检查 (本地)
# ============================================================

# 检查本地服务健康状态
health:
	@echo "检查服务健康状态..."
	@curl -s http://localhost:8080/api/v1/status | jq '.' || echo "服务未启动"