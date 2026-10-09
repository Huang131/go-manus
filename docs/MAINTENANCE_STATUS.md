# 维护状态

## 已完成

- 多模型配置支持草稿测试和已保存配置测试。
- OpenAI/Anthropic 适配器统一到 `llmcore` 协议。
- Planner 与 ReAct 的结构化响应、工具调用和文本流式边界已明确。
- Logger 不再关闭进程级 stdout/stderr；Sandbox 资源清理、文件权限和取消路径已有回归覆盖。
- API 测试已按 unit、component、HTTP integration 和 external smoke 分层；Chat 成功、SSE 顺序和取消终态进入 `make test-api` 门禁。
- MCP 配置 JSON 字段已统一为 `name`，后端 integration fixture 和前端消费者使用同一契约。

## 验证基线

- Go 定向单元测试和 `go vet` 应通过。
- UI `npm run build` 应通过；lint 中已有的未使用变量和 Next Image 警告不作为阻断错误。
- `make test-component` 和 `make test-api` 会启动并清理独立 PostgreSQL、Redis、MinIO 环境；Sandbox 测试使用单独的 `make test-sandbox`。

## 已知限制

- Planner 依赖模型结构化输出；模型不遵守 JSON 契约时应返回明确错误并保留可诊断信息。
- Run/Session 重构仍处于方案阶段；当前生产语义仍是 Session/RedisStreamTask。
- 历史 review 中的性能数字、接口示例和 TODO 不代表当前实现，修改前必须以代码和测试为准。
- 仓库不再提交 Graphify 图谱快照；clone 后需在仓库根目录执行一次 `graphify update .` 才能使用图谱查询（见 `docs/CODE_INTELLIGENCE.md`）。

## 已修复问题

### Graphify 状态目录不一致：已提交快照无法再由工具刷新（2026-10-09）

- **复现**：`graphify update .` 只更新 `.graphify/`，不触碰已提交的 `graphify-out/`——实测执行前后 `graphify-out/graph.json`、`graphify-out/manifest.json` 的 mtime 保持不变，而 `.graphify/` 下的同名文件被重写。`graphify migrate-state` 的语义即 "Migrate legacy graphify-out state into .graphify"；`graphify portable-check` 也把 commit-safe 产物定位在 `.graphify/`。
- **根因**：Graphify 0.10 起状态目录由 `graphify-out/` 改为 `.graphify/`（源码常量 `DEFAULT_GRAPHIFY_STATE_DIR=".graphify"`、`LEGACY_GRAPHIFY_STATE_DIR="graphify-out"`）。仓库的 `.gitignore`、文档与规则文件仍按旧的"提交 `graphify-out/` 快照"流程书写，并被白名单反向规则提交，导致快照只能靠人工编辑维持、持续腐化。另有两处衍生问题：`.gitignore` 只忽略 `api/graphify-out/` 而漏掉 `ui/graphify-out/`（其 cache 被跟踪）；`api/.graphify/`、`api/graphify-out/` 是有人在 `api/` 子目录单独运行 graphify 产生的冗余状态目录。
- **修复**：按方案 A 退役提交快照。删除根 `graphify-out/`、`ui/graphify-out/`、`api/graphify-out/`、`api/.graphify/`（约 150M）并解除前两者的跟踪；`.gitignore` 归一为 `.graphify/` 与 `**/graphify-out/` 两条全层级规则；`docs/CODE_INTELLIGENCE.md` 及工作区根 `AGENTS.md`/`CLAUDE.md` 同步改为 `.graphify/` 本机态流程；新增 `scripts/check-graphify-tracked.sh`、`make check-graphify`、`make install-hooks` 防止回归。
- **验证**：`git ls-files | grep -c graphify` 为 0；反向探针（强塞 `graphify-out/probe.json`）使 `make check-graphify` 非 0 退出并输出修复提示；真实探针确认 `.graphify/` 与任意层级 `graphify-out/` 均被 ignore；`.git/hooks/pre-commit` 安装后运行返回 0。

## 维护规则

新增问题记录应补充复现、根因、修复、验证四项；完成后更新本文档，不再新增无日期的临时计划文件。
