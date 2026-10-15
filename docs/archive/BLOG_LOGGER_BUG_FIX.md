# 记一次 Go 日志库 os.Stdout 被误关闭的 Bug 修复

## 背景

在 go-manus 项目中，logger 包使用 zap 作为底层日志库，包装了一层简化 API，支持：

- 全局日志实例（通过 `Init` / `InitWithConfig` 初始化）
- 热重载（运行时动态调整日志级别或切换输出目标）
- 同时输出到 stdout 和日志文件（通过 lumberjack 实现日志轮转）

在一次运行单元测试覆盖率时，发现报错：

```
write error: write /dev/stdout: file already closed
error generating coverage report: write /dev/stdout: ...
```

`go test` 命令自身会往 stdout 写覆盖率数据，此时发现文件描述符已被关闭。更隐蔽的是，在服务运行期间动态切换日志配置后，`docker logs` 看不到任何输出——stdout 的 fd 已经被关了。

---

## 问题代码

```go
package logger

var (
    encoder    zapcore.Encoder
    writeSyncer zapcore.WriteSyncer
    log        atomic.Pointer[zap.Logger]
)

func shutdownCurrent() {
    if oldLog := log.Swap(nil); oldLog != nil {
        _ = oldLog.Sync()
    }
    if writeSyncer != nil {
        if closer, ok := writeSyncer.(io.Closer); ok {
            _ = closer.Close()  // ← 问题代码
        }
    }
    writeSyncer = nil
    encoder = nil
}

func InitWithConfig(cfg Config) error {
    // ...
    if cfg.Filename != "" {
        writeSyncer = zapcore.NewMultiWriteSyncer(
            zapcore.AddSync(os.Stdout),      // ← *os.File
            zapcore.AddSync(lumberjackLogger),
        )
    } else {
        writeSyncer = zapcore.AddSync(os.Stdout)  // ← 也是 *os.File
    }
    // ...
}
```

---

## 根因分析：三层面递进

### 第一层：`*os.File` 实现了 `io.Closer`

Go 的 `os.Stdout` 类型是 `*os.File`，它实现了 `Close()` 方法：

```go
var stdout *os.File  // os.Stdout 的实际类型

// *os.File.Close() → syscall.Close(fd) → 关闭进程级 fd 1
```

所以 `zapcore.AddSync(os.Stdout)` 返回的实际上是 `*os.File`，它实现了 `io.Closer` 接口。`shutdownCurrent` 中的接口断言 `closer, ok := writeSyncer.(io.Closer)` **永远成功**，调用 `closer.Close()` 就会关闭进程级 fd。

### 第二层：两种模式都触发

**纯 stdout 模式**（之前没发现是因为测试场景不完整）：

```
Init("info")
  → writeSyncer = AddSync(os.Stdout)（直接持有 *os.File，fd=1）

Init("debug")        // 热重载触发
  → shutdownCurrent()
      → writeSyncer.(io.Closer) → true
      → closer.Close() → 关闭 fd 1

os.Stdout 已死 → 所有后续 stdout 写入 → EBADF
```

**文件模式**（通过 MultiWriteSyncer 递归关闭）：

```
Init(filename="app.log")
  → writeSyncer = MultiWriteSyncer(AddSync(os.Stdout), AddSync(lumberjackLogger))

Init(filename="app.log")  // 再次调用
  → shutdownCurrent()
      → MultiWriteSyncer.Close()
          → 递归调用各子 syncer 的 Close()
              → AddSync(os.Stdout).Close() → 关闭 fd 1
```

### 第三层：Sync 失败引发无限递归

即使 `Close()` 不触发，如果 fd 已被其他途径关闭，`Sync()` 会返回 `EBADF`。zap 的 `Sync()` 失败后会调用内部错误日志记录器，而内部 logger 尝试写入 stdout，又触发 `Sync()` 失败——**无限递归**，在终端上表现为海量刷屏的错误信息。

---

## 修复方案演进

### 方案一：哨兵身份比较（不完整）

```go
stdoutSyncer := zapcore.AddSync(os.Stdout)  // 包级变量

func shutdownCurrent() {
    if writeSyncer != nil && writeSyncer != stdoutSyncer {  // ← 哨兵比较
        if closer, ok := writeSyncer.(io.Closer); ok {
            _ = closer.Close()
        }
    }
}
```

**缺陷**：文件模式下 `writeSyncer = MultiWriteSyncer(...)` 与 `stdoutSyncer` 不等，进入 Close 分支，`MultiWriteSyncer.Close()` 递归调用子 syncer 的 Close，stdout 还是被关。只解决了表象。

### 方案二：neverCloseSyncer 包装（最终采用）

```go
type neverCloseSyncer struct {
    zapcore.WriteSyncer
}

func (n neverCloseSyncer) Close() error { return nil }

// Sync() 根据底层资源类型决定行为：
//   - 底层为 os.Stdout（fd=1）或 os.Stderr（fd=2）时返回 nil（切断递归链）
//   - 其他资源（文件、内存等）透传真实 Sync 错误
func (n neverCloseSyncer) Sync() error {
    if n.WriteSyncer == nil {
        return nil
    }
    if underlying, ok := n.getUnderlyingFile(); ok {
        if underlying.Fd() == 1 || underlying.Fd() == 2 {
            return nil
        }
    }
    return n.WriteSyncer.Sync()
}
```

**为什么需要覆盖 `Sync()` 而不只是 `Close()`？**

因为 fd 关闭后，`Sync()` 返回 `EBADF`，触发 zap 内部错误日志的无限递归。覆盖 `Sync()` 返回 nil 切断递归链。

**进一步改进：区分 stdout/stderr 和普通文件**

最初的实现对所有底层资源都返回 nil，这意味着日志文件的同步错误（如磁盘满）也会被吞掉。改进后的实现通过 `unwrapToFile` 递归解包底层 `*os.File`，**只有 fd=1（stdout）或 fd=2（stderr）时才返回 nil**，对普通文件（fd≥3）透传真实错误。这使得日志文件的同步失败可以被上层感知，同时保护 stdout/stderr 不触发无限递归。

### 方案三：不调用 Close，只调用 Sync

放弃 `shutdownCurrent` 中的 `Close()` 调用，依赖 OS 进程退出时自动关闭 fd。

**缺陷**：lumberjack 打开的文件句柄在反复 re-Init 时会泄漏（fd 累积），内存和文件句柄都会慢慢耗尽。

### 最终代码

[pkg/logger/logger.go](file:///Users/huanghao2/GolandProjects/study/imooc-mas/go-manus/api/pkg/logger/logger.go)（行 68-95，164-172，255-266）：

```go
// neverCloseSyncer 包装一个 WriteSyncer，将其 Close() 和 Sync() 方法变成无操作。
// 保护进程级共享资源（如 os.Stdout）不被 MultiWriteSyncer.Close() 递归关闭。
type neverCloseSyncer struct {
    zapcore.WriteSyncer
}

func (n neverCloseSyncer) Close() error { return nil }
func (n neverCloseSyncer) Sync() error  { return nil }
```

---

## 并发安全性分析

### neverCloseSyncer 自身

无任何可变状态，所有方法均是透传或空操作，**天然并发安全**。

### 多个 goroutine 同时调用 Init

`InitWithConfig` 中用 `muLevel.Lock()` 保护 shutdown + 新建 logger 的整个过程，**写锁保证串行性**，不会出现"A 关 stdout，B 写 stdout 失败"的交错。

### 读端（Get / SetLevel）

`log atomic.Pointer[zap.Logger]` 保证无锁并发读。`SetLevel` 只修改 `atomicLevel`，与 `writeSyncer` 读写分离，无竞争。

### 生产环境热重载

低频操作，可以接受写锁短暂阻塞。`SetLevel`（原子级别修改）完全无锁，可高频调用。

---

## 测试设计

覆盖 9 个场景，验证 neverCloseSyncer 的所有行为路径：

| 测试 | 验证内容 |
|------|----------|
| `TestNeverCloseSyncer_StdoutModeSwitch` | 纯 stdout 模式连续 3 次 re-Init |
| `TestNeverCloseSyncer_StdoutToFileModeSwitch` | stdout → 文件模式切换 |
| `TestNeverCloseSyncer_FileToStdoutModeSwitch` | 文件 → stdout 模式切换 |
| `TestNeverCloseSyncer_RepeatedSwitches` | 6 轮反复切换（stdout↔file） |
| `TestNeverCloseSyncer_MultiFileModeInit` | 文件模式 5 次 re-Init |
| `TestNeverCloseSyncer_ConcurrentInit` | 10 goroutine 并发切换 |
| `TestNeverCloseSyncer_IsSafe` | 50 goroutine × 100 次 Write/Sync/Close |
| `TestNeverCloseSyncer_CloseIsNoOp` | Close 前后 Write 行为不变 |

所有测试均验证 `stdoutWriteOK()` ——通过向 os.Stdout 写入空字节检测底层 fd 是否仍有效：

```go
func stdoutWriteOK() error {
    _, err := os.Stdout.Write([]byte(""))
    return err  // fd 已关闭时返回 EBADF
}
```

---

## 业界方案对比

| 方案 | 代表框架 | 思路 | 适用场景 |
|------|---------|------|----------|
| 依赖注入 | slog.Logger | 不使用全局变量，由调用方管理生命周期 | 新项目 / 可重构场景 |
| 写时复制（COW） | logrus `New()` | 创建新 logger 后原子替换，读端无锁 | 已有全局 logger 改造 |
| 写锁 + 原子指针 + no-op 包装 | 本项目 | 保留全局 logger 架构，最小侵入修复 | 存量项目 |
| 只用 Sync 不用 Close | 部分轻量库 | 放弃主动关闭，依赖 OS | fd 泄漏可接受场景 |

slog 的依赖注入方案从架构上根本规避了这个问题——因为全局状态变成了参数传递，资源生命周期由显式的代码路径管理，不存在"谁该关闭"的歧义。但迁移成本高，需要改动所有调用点。

本项目的 `neverCloseSyncer` 方案在**不改变现有架构（全局 logger + 热重载）**的前提下，用最小改动（30 行代码）解决了问题。

---

## 核心设计原则

**1. 谁创建，谁负责关闭**

`shutdownCurrent` 的职责是释放"本次 Init 创建的资源"。`os.Stdout` 是进程启动时 OS 打开的，不归 logger 管理。

**2. 进程级共享资源的保护契约**

`os.Stdout`、`os.Stderr`、`os.Stdin` 是进程级共享资源，库代码无权关闭。这是通过类型层面保证的，而不是运行时检查。

**3. 接口组合的意外暴露**

`zapcore.WriteSyncer` 嵌入了 `io.Writer` 和 `io.Syncer`，但某些 syncer（如 `*os.File`）同时实现了 `io.Closer`。使用接口断言时需要明确：这个 Close 是不是应该调用的。

**4. no-op 包装模式**

当需要"告诉某个系统不要对某个资源做某操作"时，空操作包装是最优雅的方式：

```go
// 禁止关闭：数据库连接池归还而非关闭
// 禁止同步：测试时 mock 掉不需要验证的方法
// 本场景：保护 stdout 的 Close/Sync 不被调用
type neverCloseSyncer struct { zapcore.WriteSyncer }
func (n neverCloseSyncer) Close() error { return nil }
func (n neverCloseSyncer) Sync() error  { return nil }
```

---

## 延伸思考：这个 Bug 的本质

表面上是"调用了不该调用的方法"，实际上是**资源所有权不清晰**。

`shutdownCurrent` 作为清理函数，它拥有对 `writeSyncer` 的引用，但并不清楚这个引用指向的资源中哪些是自己的、哪些是共享的。`neverCloseSyncer` 的本质是一个**所有权声明**——"stdout 这个 fd 归进程，不归你，别动它"。

这与 C++ 的 RAII、Go 的 `defer file.Close()` 逻辑一致：谁打开谁关闭。但全局状态使得"打开"和"关闭"不在同一个作用域，需要显式声明。

另一个思路是在 API 设计层面解决：如果 logger 的 `Output` 字段由调用方持有（类似 logrus 的 `l.SetOutput(w)`），清理责任就完全在调用方，logger 只负责写入。但当前项目已大量使用全局 logger，迁移成本过高。

**没有银弹，只有当前约束下的最优解。** 对于已有全局 logger 的存量项目，`neverCloseSyncer` 是最干净的方案。