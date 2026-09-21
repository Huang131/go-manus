#!/bin/bash
# ============================================================
# 测试环境清理脚本
# ============================================================
# 作用：停止独立测试环境并删除测试 volume
# ============================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MANUS_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="$MANUS_ROOT/docker-compose.test.yml"

echo "=== 停止并删除独立测试环境 ==="
docker compose -f "$COMPOSE_FILE" down -v --remove-orphans

echo ""
echo "=== 测试环境已清理 ==="
echo "  测试容器、网络和 volume 已删除"
