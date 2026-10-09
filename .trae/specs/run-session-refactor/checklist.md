# Run/Session 架构重构验收清单

- [ ] 当前检查点的行为测试先失败，再完成最小实现。
- [ ] `go test ./...` 通过；受限环境失败必须记录并在正常开发环境补跑。
- [ ] `go test -race ./internal/agent/... ./internal/service ./internal/repository -count=1` 通过。
- [ ] `go vet ./...` 通过。
- [ ] `git diff --check` 通过。
- [ ] 没有新增 Session/Task 与 Run 双写。
- [ ] 没有新增无消费方的配置、状态或抽象。
- [ ] 提交仅包含当前检查点，可单独 revert。
- [ ] `docs/refactor-run/STATUS.md` 已记录提交、验证结果和下一入口。

## 已完成检查点

- [x] 1A Settings 单一类型、严格校验、幂等 migration 与 UI 契约。
- [x] 1A 全量 Go 测试、目标包 race、vet、UI lint/build 和真实 PostgreSQL migration 验证。
- [x] 1B PromptCatalog 稳定名称、版本、内容 hash 与目录 hash。
- [x] 阶段 1 全量 Go 测试、Agent race 与 vet。
- [x] 2A StepOutcome 唯一信号，删除 `ErrWaitForUser`、`InvokeResult.WaitForUser` 与 `ToolCallResult.WaitForUser`。
- [x] 2A 等待输入、取消、失败、完成和 Flow 事件顺序行为测试；全量 Go 测试、目标包 race、vet 与 diff 检查。
- [x] 2B ContextPolicy 统一应用于初始组装和每次 provider 调用。
- [x] 2B 工具 schema、输出预留、超预算、完整 tool pair 与原始消息顺序行为测试。
- [x] 2B 全量 Go 测试、Agent race、vet 与 diff 检查。
- [x] 2C MCP 动态 function 索引、名称碰撞拒绝、初始化资源清理与基础设施/远端业务错误分类。
- [x] 2C 全量 Go 测试、Agent race、vet 与 diff 检查。
- [x] 3A Run active/terminal 分类和唯一状态迁移矩阵。
- [x] 3A waiting_input 执行快照校验：当前步骤、问题消息、恢复模式和已完成步骤摘要。
- [x] 3A 纯恢复转换器：初始输入、完成步骤摘要、问题和回答的固定 role/顺序/附件契约。
- [x] 3A 测试先失败；全量 Go 测试、目标包 race、vet 与 diff 检查。
- [x] 3B `runs/messages` migration、Run/Message Repository、Run aggregate 窄事务入口与真实 PostgreSQL 组件测试。
- [x] 3B 创建幂等、每 Session 单 active Run、waiting_input 快照/问题原子写入、恢复输入幂等与事务回滚测试。
- [x] 3B 将 `TestRunRepo_` 纳入 `make test-component` 门禁；重复执行 migration 后组件测试通过。
- [x] 3C 纯 RunService：创建、启动、waiting_input、恢复输入、取消请求和取消收敛用例。
- [x] 3C 服务层拒绝 cancelling/终态输入，重复恢复输入幂等，损坏快照在持久化变更前失败。
