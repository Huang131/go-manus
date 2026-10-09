# 代码智能数据使用说明

> 本文只说明当前 Graphify/GitNexus 工作方式，不替代架构和运行维护文档。

本项目使用两类代码智能工具辅助架构分析、代码导航和变更影响评估。两者的产物都只留在本机，都不提交到仓库：

- **Graphify**：把项目解析成知识图谱，状态保存在 `.graphify/`，供本机查询和浏览。
- **GitNexus**：生成本机代码关系索引（`.gitnexus/`）；每位开发者在本机建立自己的索引。

## 提交策略

### Graphify

Graphify 的状态目录是 `.graphify/`，属于本机运行态，不提交：

```text
.graphify/
├── graph.json          # 知识图谱本体
├── GRAPH_REPORT.md     # 可读的架构报告
├── manifest.json       # 文件指纹索引，用于增量更新
├── scope.json          # 本次解析的输入范围
├── branch.json         # 分支与陈旧标记
├── worktree.json       # 当前 worktree 元数据
└── cache/              # AST 与统计缓存
```

`.gitignore` 用两条规则覆盖所有层级：

```gitignore
.graphify/
**/graphify-out/
```

`graphify-out/` 是 0.10 之前版本的旧状态目录（`graphify migrate-state` 的作用就是把 legacy 的 `graphify-out/` 迁到 `.graphify/`），当前版本已不再生成它，也不再提交。

图谱只在**仓库根目录**生成。不要在 `api/`、`ui/`、`sandbox/` 等子目录里单独运行 `graphify`，否则会在子目录内额外产生一份状态目录（历史上 `api/graphify-out/`、`api/.graphify/`、`ui/graphify-out/` 就是这么出现的）。

### GitNexus

`.gitnexus/` 是 GitNexus 的本机数据库索引，包含本地解析缓存和数据库文件，绑定当前机器路径，通常体积较大，因此不提交。

每位开发者 clone 项目后，在项目根目录执行一次：

```bash
npx gitnexus analyze
npx gitnexus status
```

GitNexus 会在本机生成 `.gitnexus/`，并在用户目录维护仓库注册信息。

### Claude Code 配置

`.claude/skills/gitnexus/` 中的 GitNexus 使用说明属于团队共享规则，可以提交。

`settings.json` 如果用于团队共享的 Claude Code 项目配置，可以提交；`settings.local.json` 属于个人本机配置，不提交：

```text
.claude/settings.local.json
```

根目录的 `AGENTS.md` 和 `CLAUDE.md` 如果包含项目团队规则，应与现有规则合并后提交；它们不是 GitNexus 数据库，不能替代本机执行 `npx gitnexus analyze`。

## 安装工具

### GitNexus

无需全局安装，使用 `npx` 执行：

```bash
npx gitnexus status
npx gitnexus analyze
```

### Graphify

确认本机已安装 `graphify` 命令：

```bash
graphify --help
```

如果未安装，按照 Graphify 官方安装方式安装。项目不把 Graphify 状态目录（`.graphify/`）或任何图谱产物提交到仓库。

## Clone 后的首次使用

仓库不提交图谱快照，需要在本机生成一次。在**项目根目录**执行：

```bash
# 生成 Graphify 知识图谱（写入本机 .graphify/）
graphify update .

# 创建本机 GitNexus 索引
npx gitnexus analyze
npx gitnexus status
```

生成后即可用 `graphify query` / `path` / `explain` 查询；需要 HTML 视图时执行 `graphify export html --out .graphify/graph.html`。没有配置 LLM API key 时，`graphify update .` 会跳过节点描述生成，可加 `--no-description` 显式静默该提示。

## 日常更新

代码、文档或目录结构发生变化后，重新生成本机数据：

```bash
# 增量更新 Graphify 知识图谱
graphify update .

# 更新本机 GitNexus 索引
npx gitnexus analyze
npx gitnexus status
```

两者都只写本机目录（`.graphify/`、`.gitnexus/`），**不产生需要提交的文件**。如果 `graphify check-update` 提示图谱落后于 `HEAD`，重跑一次 `graphify update .` 即可。

## 查询示例

### Graphify

```bash
# 广泛了解项目结构
graphify query "项目整体架构、服务边界和主要依赖"

# 查询两个模块之间的关系
graphify path "AgentService" "Sandbox"

# 解释一个概念或模块
graphify explain "ReAct Agent"
```

### GitNexus

```bash
# 查看索引是否过期
npx gitnexus status

# 重新分析项目
npx gitnexus analyze
```

在支持 GitNexus MCP 的开发工具中，还可以使用项目上下文、调用关系、影响分析和执行流程资源。索引过期时，先运行 `npx gitnexus analyze`。

## 提交前检查

```bash
# 确认 Graphify 状态目录没有被误提交（正常时无输出并返回 0）
make check-graphify

# 确认 GitNexus 不会被提交
git status --short --ignored .gitnexus

# 检查文件格式和空白错误
git diff --check
```

预期结果：

- `make check-graphify` 无输出并返回 0。
- `.gitnexus/` 显示为 ignored。
- `.graphify/` 以及任意层级的 `graphify-out/` 显示为 ignored。
- `.claude/settings.local.json` 显示为 ignored；`.claude/settings.json` 是否提交由团队配置用途决定。

`make install-hooks` 会把 `make check-graphify` 装成 `.git/hooks/pre-commit`，从源头拦住误提交；该命令只需在每个 clone 上执行一次。

## 常见问题

### Graphify 图谱提示落后于 HEAD

重新生成一次即可：

```bash
graphify update .
graphify check-update
```

### GitNexus 显示仓库未索引

在项目根目录执行：

```bash
npx gitnexus analyze
npx gitnexus status
```

如果只执行 `npx gitnexus status`，不会自动创建索引。
