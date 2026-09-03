#!/bin/bash
# ============================================================
# 测试环境清理脚本
# ============================================================
# 作用：清理测试数据库（开发 Docker 上的 manus_test）
# 注意：仅清理数据，不删除数据库本身（迁移脚本已就位）
# ============================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "=== 清理测试数据库 manus_test ==="
if docker ps --format '{{.Names}}' | grep -q 'go-manus-postgres'; then
    docker exec go-manus-postgres psql -U postgres -d manus_test -c \
        "TRUNCATE TABLE files, sessions, app_configs, llm_models RESTART IDENTITY CASCADE;" 2>/dev/null \
        && echo "  测试数据已清空" \
        || echo "  数据库可能为空或无数据"
else
    echo "  PostgreSQL 未运行，跳过"
fi

echo ""
echo "=== 测试环境已清理 ==="
echo "  manus_test 数据库保留，迁移脚本保留"
echo "  下次 test-env-up.sh 会重新初始化"