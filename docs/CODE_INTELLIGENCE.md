# 代码智能数据使用说明

本项目使用两类代码智能工具辅助架构分析、代码导航和变更影响评估：

- **Graphify**：生成可审查的项目知识快照，提交到仓库，供团队成员直接浏览和查询。
- **GitNexus**：生成本机代码关系索引，不提交到仓库；每位开发者在本机建立自己的索引。

## 提交策略

### Graphify

仓库只提交根目录 `graphify-out/` 的当前快照：

```text
graphify-out/
├── .graphify_root
├── .graphify_labels.json
├── .graphify_labels.json.sig
├── GRAPH_REPORT.md
├── graph.json
├── graph.html
└── manifest.json
```

以下内容不提交：

```text
graphify-out/cache/
graphify-out/YYYY-MM-DD/
api/graphify-out/
```

根目录快照是整个 `go-manus` 项目的统一图谱。`api/graphify-out/` 是 API 子目录图谱，不单独维护，避免出现两份容易过期的数据。

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

如果未安装，按照 Graphify 官方安装方式安装。项目不把 Python 虚拟环境或 Graphify 缓存提交到仓库。

## Clone 后的首次使用

在项目根目录执行：

```bash
# 检查已提交的 Graphify 快照
graphify query "项目整体架构和主要调用关系"

# 创建本机 GitNexus 索引
npx gitnexus analyze
npx gitnexus status
```

Graphify 的 `graph.json` 和 `GRAPH_REPORT.md` 可以直接用于浏览项目结构；GitNexus 建立索引后，可以进一步使用调用关系、影响分析和流程追踪能力。

## 日常更新

代码、文档或目录结构发生变化后，建议在提交前更新两类数据：

```bash
# 更新根目录 Graphify 快照
graphify update .

# 更新本机 GitNexus 索引
npx gitnexus analyze
npx gitnexus status
```

Graphify 更新后，只提交根目录当前快照文件：

```bash
git add graphify-out/.graphify_root \
  graphify-out/.graphify_labels.json \
  graphify-out/.graphify_labels.json.sig \
  graphify-out/GRAPH_REPORT.md \
  graphify-out/graph.json \
  graphify-out/graph.html \
  graphify-out/manifest.json
```

GitNexus 的 `.gitnexus/` 不应加入提交。

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
# 确认 GitNexus 不会被提交
git status --short --ignored .gitnexus

# 确认 Graphify 只暴露当前快照
git status --short --ignored graphify-out api/graphify-out

# 检查文件格式和空白错误
git diff --check
```

预期结果：

- `.gitnexus/` 显示为 ignored。
- `graphify-out/cache/` 和 `graphify-out/YYYY-MM-DD/` 显示为 ignored。
- `graphify-out/` 根目录当前快照文件可被 `git add` 选择。
- `api/graphify-out/` 显示为 ignored。
- `.claude/settings.local.json` 显示为 ignored；`.claude/settings.json` 是否提交由团队配置用途决定。

## 常见问题

### Graphify 文件仍被全局 Git ignore 忽略

某些开发环境会在用户级 `.gitignore` 中写入 `graphify-out/`。仓库 `.gitignore` 已为根目录当前快照添加反向规则；如果本机 Git 版本或规则配置仍然拦截，可以显式添加当前快照：

```bash
git add -f graphify-out/.graphify_root \
  graphify-out/.graphify_labels.json \
  graphify-out/.graphify_labels.json.sig \
  graphify-out/GRAPH_REPORT.md \
  graphify-out/graph.json \
  graphify-out/graph.html \
  graphify-out/manifest.json
```

不要使用 `git add -f graphify-out/`，否则可能把缓存和历史备份一并加入提交。

### GitNexus 显示仓库未索引

在项目根目录执行：

```bash
npx gitnexus analyze
npx gitnexus status
```

如果只执行 `npx gitnexus status`，不会自动创建索引。
