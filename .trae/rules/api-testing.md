---
alwaysApply: true
description: API 测试编写、重构和评审规则
---

# API 测试规则

## 测试先绑定契约

- 每个测试必须说明它守护的业务、存储或协议契约。
- 不以测试数量或总覆盖率作为质量目标。
- 不为 getter、常量、接口实现编译断言或 mock 自身固定返回值编写独立测试。
- 宽松断言（例如“非 500”“返回任意非空值”）必须有明确理由，否则改为精确状态、错误码和响应数据断言。

## 衰减风险评审

评审测试时逐项检查以下风险：Test Obscurity、Test Brittleness、Test Duplication、Mock Abuse、Coverage Illusion、Architecture Mismatch 和 Layer Boundary Violation。

每个问题都按以下顺序记录：`Symptom -> Evidence -> Consequence -> Remedy`。证据必须能定位到源码、测试输出或可复现实验；没有证据的猜测不升级为整改项。

## 分层边界

- Unit 测试不得连接 PostgreSQL、Redis、MinIO、真实 LLM 或第三方网络。
- Service fake 只验证调用参数、调用顺序和错误传播，不模拟 SQL 分页、事务、唯一索引或并发。
- Repository 的 SQL、JSONB、事务、约束和并发必须使用真实 PostgreSQL component 测试。
- Redis Stream 的 cursor、blocking read、cancel、retention 和续读必须使用真实 Redis component 测试。
- MinIO/S3 adapter 使用真实对象存储验证协议；业务补偿和错误映射使用可控 fake。
- HTTP/API 测试使用真实 Gin、service、repository 和测试基础设施；外部 LLM 使用 deterministic fake。
- 真实 SenseNova、Search、Sandbox 测试属于 external smoke，必须显式启用，不能成为普通单测门禁。

## 测试替身

- Handler stub 只返回预设结果并记录调用，不维护完整业务 map 或状态机。
- 不要在内存 fake 中重写 Redis、SQL 或对象存储语义，然后把它命名为 integration test。
- 并发 mock 的 channel 关闭权只能归发送方；cancel 只发送停止信号。
- goroutine 必须有明确退出信号；测试优先等待 channel/barrier，禁止用固定 Sleep 等待状态变化。
- 生产对象支持依赖注入时，每个测试使用独立 registry、key prefix 或 fixture，避免包级全局状态互相污染。

## 外部环境

- `go test ./...` 必须无外部依赖、无固定端口、无真实网络调用。
- 集成配置通过集中式 `TestEnv/IntegrationConfig` 读取环境变量，不在测试函数中散落读取。
- 测试配置必须校验数据库、Redis db、bucket 和 key prefix 属于测试资源。
- 外部 smoke 未配置凭证时可以明确 skip；限流、服务错误和协议错误不能无条件转换为通过。

## 变更要求

- 新增生产接口、路由或状态迁移时，至少增加参数校验和主流程契约测试。
- 重命名 JSON/API 字段时，必须在同一改动中同步后端模型、Handler/Service、integration fixture、前端类型和实际消费者；开发阶段不保留新旧双字段兼容。
- 新增或重命名核心 HTTP 契约测试时，必须同步维护 `Makefile` 的分层测试选择器，避免测试存在但不进入默认门禁。
- `make test-api` 必须覆盖当前 Chat 成功链路、SSE 事件顺序和取消终态契约。
- 删除测试前确认生产调用、替代测试和业务语义，不能只因测试数量多而删除。
- 重构测试时保持每一步可编译、可运行、可回滚，不长期保留新旧两套业务语义测试。
- 完成测试修改后至少运行相关 unit、component 或 HTTP 层测试，并在需要时运行 `go vet` 和 `-race`。
