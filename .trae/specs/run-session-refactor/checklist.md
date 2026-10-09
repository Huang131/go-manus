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
