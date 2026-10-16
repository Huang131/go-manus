#!/usr/bin/env sh

# 验证网关在 Docker Compose 服务重建后仍会重新解析服务地址。
set -eu

PROJECT_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
CONFIG="$PROJECT_ROOT/nginx/conf.d/default.conf"

if ! grep -Eq '^[[:space:]]*resolver 127[.]0[.]0[.]11 valid=10s ipv6=off;[[:space:]]*$' "$CONFIG"; then
    echo '缺少 Docker DNS resolver 配置' >&2
    exit 1
fi

if ! grep -Eq '^[[:space:]]*set [$]api_upstream api:8080;[[:space:]]*$' "$CONFIG"; then
    echo '缺少 API 动态上游配置' >&2
    exit 1
fi

if ! grep -Eq '^[[:space:]]*set [$]sandbox_upstream sandbox:8080;[[:space:]]*$' "$CONFIG"; then
    echo '缺少 sandbox 动态上游配置' >&2
    exit 1
fi

if ! grep -Eq '^[[:space:]]*set [$]ui_upstream ui:3000;[[:space:]]*$' "$CONFIG"; then
    echo '缺少 UI 动态上游配置' >&2
    exit 1
fi

if ! grep -Eq '^[[:space:]]*proxy_pass http://[$]api_upstream;[[:space:]]*$' "$CONFIG"; then
    echo '缺少 API 动态代理配置' >&2
    exit 1
fi

if ! grep -Eq '^[[:space:]]*proxy_pass http://[$]sandbox_upstream;[[:space:]]*$' "$CONFIG"; then
    echo '缺少 sandbox 动态代理配置' >&2
    exit 1
fi

if ! grep -Eq '^[[:space:]]*rewrite [^ ]*sandbox/[(].[*][)][$] /[$]1 break;[[:space:]]*$' "$CONFIG"; then
    echo 'sandbox 代理必须剥离网关路径前缀' >&2
    exit 1
fi

if ! grep -Eq '^[[:space:]]*location = /sandbox [ {][[:space:]]*$' "$CONFIG"; then
    echo '缺少 sandbox 根路径的斜杠规范化配置' >&2
    exit 1
fi

if ! grep -Eq '^[[:space:]]*proxy_pass http://[$]ui_upstream;[[:space:]]*$' "$CONFIG"; then
    echo '缺少 UI 动态代理配置' >&2
    exit 1
fi

if grep -Eq '^upstream (api|sandbox|ui)_backend' "$CONFIG"; then
    echo '不应使用启动期解析的静态 upstream 配置' >&2
    exit 1
fi
