#!/bin/bash
# ============================================================
# 测试环境初始化脚本
# ============================================================
# 效果：启动独立的 PostgreSQL、Redis 和 MinIO 测试环境
# ============================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MANUS_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

COMPOSE_FILE="$MANUS_ROOT/docker-compose.test.yml"

cleanup_on_error() {
    echo "=== 测试环境启动失败，输出容器日志 ==="
    docker compose -f "$COMPOSE_FILE" logs --no-color || true
    docker compose -f "$COMPOSE_FILE" down -v --remove-orphans || true
}
trap cleanup_on_error ERR

echo "=== 启动独立测试基础设施 ==="
docker compose -f "$COMPOSE_FILE" up -d --wait postgres redis minio

echo "=== 在 manus_test 上执行所有迁移 ==="
for f in "$MANUS_ROOT"/api/migrations/*.sql; do
    echo "  执行 $(basename "$f") ..."
    docker compose -f "$COMPOSE_FILE" exec -T postgres \
        psql -U postgres -d manus_test -v ON_ERROR_STOP=1 < "$f"
done

echo "=== 创建测试 bucket go-manus-test-files（若不存在） ==="
docker compose -f "$COMPOSE_FILE" run --rm minio-init

trap - ERR

echo ""
echo "=== 测试环境就绪 ==="
echo "  PostgreSQL: localhost:${API_TEST_POSTGRES_PORT:-15432} / manus_test"
echo "  Redis:      localhost:${API_TEST_REDIS_PORT:-16379} / db 1"
echo "  MinIO:      localhost:${API_TEST_S3_PORT:-19000} / go-manus-test-files"
echo ""
echo "可运行集成测试: cd $MANUS_ROOT/api && go test -tags=integration -v ./tests/..."
