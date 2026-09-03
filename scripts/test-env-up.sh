#!/bin/bash
# ============================================================
# 测试环境初始化脚本
# ============================================================
# 依赖：开发环境 Docker (docker-compose.yml) 已启动
# 效果：在开发 Docker 上创建测试专用的数据库和 bucket
# ============================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MANUS_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "=== 检查开发 Docker 是否运行 ==="
if ! docker ps --format '{{.Names}}' | grep -q 'go-manus-postgres'; then
    echo "ERROR: 开发环境 PostgreSQL (go-manus-postgres) 未运行"
    echo "请先运行: cd $MANUS_ROOT && docker compose up -d"
    exit 1
fi

if ! docker ps --format '{{.Names}}' | grep -q 'go-manus-minio'; then
    echo "ERROR: 开发环境 MinIO (go-manus-minio) 未运行"
    echo "请先运行: cd $MANUS_ROOT && docker compose up -d"
    exit 1
fi

echo "=== 创建测试数据库 manus_test（若不存在） ==="
docker exec go-manus-postgres psql -U postgres -d postgres -c \
    "SELECT 1 FROM pg_database WHERE datname='manus_test'" | grep -q 1 \
    || docker exec go-manus-postgres psql -U postgres -d postgres -c \
    "CREATE DATABASE manus_test"

echo "=== 在 manus_test 上执行所有迁移 ==="
for f in "$MANUS_ROOT"/api/migrations/*.sql; do
    echo "  执行 $(basename "$f") ..."
    docker exec -i go-manus-postgres psql -U postgres -d manus_test < "$f"
done

echo "=== 创建测试 bucket go-manus-test-files（若不存在） ==="
docker exec go-manus-minio mc alias set local http://localhost:9000 minioadmin minioadmin 2>/dev/null || true
docker exec go-manus-minio mc mb --ignore-existing local/go-manus-test-files 2>/dev/null || true
docker exec go-manus-minio mc anonymous set download local/go-manus-test-files 2>/dev/null || true

echo ""
echo "=== 测试环境就绪 ==="
echo "  PostgreSQL: localhost:5432 / manus_test (复用开发 Docker)"
echo "  Redis:      localhost:6379 / db 1       (复用开发 Docker)"
echo "  MinIO:      localhost:9000 / go-manus-test-files (复用开发 Docker)"
echo ""
echo "可运行集成测试: cd $MANUS_ROOT/api && go test -tags=integration -v ./tests/..."