# API 测试规则

这份规则是 `api/` 测试代码的项目级约束。测试的目标是守护业务、存储或协议契约，不是堆积函数数量或追求覆盖率数字。

## 分层

- Unit 测试不得连接 PostgreSQL、Redis、MinIO、真实 LLM 或第三方网络。
- Service fake 只验证调用参数、调用顺序和错误传播，不模拟 SQL 分页、事务、唯一索引或并发。
- Repository 的 SQL、JSONB、事务、约束和并发必须使用真实 PostgreSQL component 测试。
- Redis Stream 的 cursor、blocking read、cancel、retention 和续读必须使用真实 Redis component 测试。
- MinIO/S3 adapter 使用真实对象存储验证协议；业务补偿使用可控 fake。
- HTTP/API 测试使用真实 Gin、service、repository 和测试基础设施；外部 LLM 使用 deterministic fake。
- SenseNova、Search、Sandbox 等真实第三方测试属于 external smoke，不作为普通单测门禁。

## 测试替身

- Handler stub 只返回预设结果并记录调用，不维护完整业务 map 或状态机。
- 不要把重写 Redis、SQL 或对象存储语义的内存 fake 命名为 integration test。
- 并发 mock 的 channel 关闭权只能归发送方；cancel 只发送停止信号。
- goroutine 必须有明确退出信号，测试优先等待 channel/barrier，禁止用固定 `Sleep` 等待状态变化。
- 支持依赖注入时，每个测试使用独立 registry、key prefix 或 fixture，避免包级全局状态污染。

## 断言和维护

- 每个测试必须能说明自己守护的契约。
- 不为 getter、常量、接口实现编译断言或 mock 自身固定返回值编写独立测试。
- 宽松断言（例如“非 500”）必须有明确理由，否则断言精确 HTTP 状态、错误码和响应数据。
- 删除测试前确认生产调用、替代测试和业务语义；不能只因测试数量多而删除。
- 新增接口、路由或状态迁移时，至少覆盖参数校验和主流程。

## 环境

- `go test ./...` 不得依赖外部数据库、Redis、MinIO 或第三方服务，不得调用外部网络或占用固定端口。允许 `httptest.NewServer` 和 `127.0.0.1:0` 这类回环临时端口。
- 集成配置集中由 `internal/testsupport.IntegrationEnv` 读取环境变量，测试函数不散落读取。
- 测试配置必须校验数据库、Redis db 和 bucket 属于测试资源；未接入的 key prefix 不得作为伪配置暴露。
- 外部 smoke 未配置凭证时可以明确 skip；限流、服务错误和协议错误不能无条件转为通过。

## 评审格式

使用 Test Decay Risk Reference 的七类风险检查：

1. Test Obscurity
2. Test Brittleness
3. Test Duplication
4. Mock Abuse
5. Coverage Illusion
6. Architecture Mismatch
7. Layer Boundary Violation

每个 finding 按 `Symptom -> Evidence -> Consequence -> Remedy` 描述。没有源码、测试输出或可复现实验支持的猜测，不升级为整改项。

## 变更门禁

测试重构每一步都必须：

- 能编译；
- 能运行对应层测试；
- 有独立回滚边界；
- 不长期保留新旧两套业务语义；
- 必要时运行 `go vet` 和 `go test -race`。

## 当前范围

- 当前规则只约束现有 Session/RedisStreamTask/Agent 生产链路。
- Run 生产代码尚未实现，不提前编写猜测性的 Run 测试。
- Run 重构开始时，先冻结 Run spec，再增加状态、存储、执行和 HTTP/SSE 契约测试。
