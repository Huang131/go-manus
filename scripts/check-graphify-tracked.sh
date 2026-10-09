#!/usr/bin/env sh
# 阻止 Graphify 状态目录被误提交。
#
# .graphify/ 是当前版本的默认状态目录，graphify-out/ 是旧版本遗留的同名目录，
# 两者都属于本机运行态，不应进入版本库（见 docs/CODE_INTELLIGENCE.md）。
#
# 用法：
#   scripts/check-graphify-tracked.sh      # 由 make check-graphify 调用
#   make install-hooks                     # 安装为 .git/hooks/pre-commit
set -eu

pattern='(^|/)(graphify-out|\.graphify)/'

offenders=$(git ls-files | grep -E "$pattern" || true)

if [ -n "$offenders" ]; then
    {
        echo "ERROR: Graphify 状态目录不允许被提交。"
        echo ""
        echo "以下文件已被 Git 跟踪："
        echo "$offenders" | sed 's/^/  /'
        echo ""
        echo "修复方式："
        echo "  git rm -r --cached <路径>"
        echo "并确认 .gitignore 中包含："
        echo "  .graphify/"
        echo "  **/graphify-out/"
    } >&2
    exit 1
fi
