# 阶段 6：删除旧执行基础设施

本阶段只删除已由阶段 4/5 证明不可达的代码。不要按静态名称猜测死代码；每项删除前都要有生产引用扫描和行为测试。

## 删除前置条件

- `STATUS.md` 中阶段 5 complete。
- Handler、bootstrap、UI 不再创建或读取旧 Task。
- Run API、取消、等待输入、SSE 续读和最终消息查询通过回归。
- 当前分支有可回滚提交，开发数据库已备份；项目未上线，可以使用破坏式 migration。

## 删除清单

### Agent 旧 Task 链（6A 已完成）

- `Task`、`Stream`、`RedisStreamTask`、`defaultTaskRegistry` 已删除。
- `taskBySession`、旧 AgentService Chat/Stop/GetTaskEvents 实现已删除。
- 只为旧 Task 服务的队列适配、Runner 投影和测试已删除。
- `AgentService` 现在只负责 ToolProvider 生命周期和 AgentSettings 热更新；RunExecutor 通过窄 `AcquireTools` 接口获取 ToolSet。

删除依据：全仓生产代码扫描未发现旧 Task、旧 registry、旧 Chat/Stop/事件读取入口的消费者；Bootstrap、Router、RunApplicationService 和 RunExecutor 均已走 Run 领域链路。保留 Redis 的 `RunEventPublisher/RunEventStream`，因为它们仍是当前 Run SSE 的实时旁路。

### Session 执行字段

- `model.Session.TaskID`、`Status`、`Events`。
- `sessions.task_id`、`sessions.status`、`sessions.events` 及索引和 SQL。
- `SessionRepository.UpdateStatus`、`AppendEvent` 等旧执行方法。
- 保留 `SandboxID`、标题、未读数、最近消息、软删除和时间字段。

当前源码没有 `Session.Memories` 字段，也没有 `sessions.memories` 持久化链路，不新增也不删除不存在的伪链路。`SimpleMemory` 是 Agent 运行期内存，是否保留由 Engine 使用情况决定。

### 模型包边界

在独立检查后，把 JSON 序列化和 logger 依赖从 `model` 移到 API/存储边界；这是包边界收紧，不应与 Run 切换混在同一提交。

## 数据库迁移

新增破坏式 migration 删除旧 Session 执行列；不得新增触发器把 Run 状态写回 Session。迁移只在阶段 5 完成后执行。

## 实施提交

- `refactor(api): remove legacy task infrastructure`
- `refactor(api): make session a conversation container`
- `refactor(api): tighten model package boundary`

每个提交都必须编译、测试、可单独 revert；不通过复制旧 Task 临时修复编译错误。

## 完成定义

- 生产代码不再包含旧 Task/registry 和 Session 执行写路径。
- RunService/RunRepository 是唯一执行状态写入口。
- `model` 不依赖 logger、sonic、repository 或 external 实现。
- API 全量 test、race、vet 及 UI lint/build 通过。
- `STATUS.md` 记录最终删除清单、migration、验证结果和回滚方式。
