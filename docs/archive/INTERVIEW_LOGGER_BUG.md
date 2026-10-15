# logger 包 os.Stdout 误关闭 Bug 深度解析

> 本文档基于 go-manus 项目真实 bug，剖析其根因、修复方案、延伸思考。
> 涉及知识点：Go 接口设计、资源生命周期管理、依赖反转、防御性编程。

---

## 一、问题背景

运行单元测试覆盖率（`go test ./... -coverprofile`）时，logger 包随机失败：

```
write error: write /dev/stdout: file already closed
error generating coverage report: write /dev/stdout: ...
```

`go test` 命令自身往 stdout 写覆盖率数据时，发现文件描述符已经被关闭。

类似的，日志热重载（动态调整日志级别或切换输出目标）后，`docker logs` 再也看不到任何输出。

---

## 二、Bug 代码（原版本）

```go
var (
    encoder    zapcore.Encoder
    writeSyncer zapcore.WriteSyncer
    // ...
)

func shutdownCurrent() {
    if oldLog := log.Swap(nil); oldLog != nil {
        _ = oldLog.Sync()
    }
    if writeSyncer != nil {
        if closer, ok := writeSyncer.(io.Closer); ok {
            _ = closer.Close()  // ← 问题代码：未区分"自己的资源"和"共享资源"
        }
    }
    writeSyncer = nil
    encoder = nil
}

func InitWithConfig(cfg Config) error {
    // ...
    if cfg.Filename != "" {
        writeSyncer = zapcore.NewMultiWriteSyncer(
            zapcore.AddSync(os.Stdout),      // ← 创建 stdout wrapper
            zapcore.AddSync(lumberjackLogger),
        )
    } else {
        writeSyncer = zapcore.AddSync(os.Stdout)  // ← 同样直接用 os.Stdout
    }
    // ...
}
```

---

## 三、根因分析

### 第一层：`os.Stdout` 实现了 `io.Closer`

```go
// os.Stdout 的类型是 *os.File
// *os.File.Close() → syscall.Close(fd) → fd 对应的文件描述符被关闭
var stdout *File  // os.Stdout 的实际类型
```

所以 `zapcore.AddSync(os.Stdout)` 的返回值（`*os.File`）实现了 `io.Closer`，接口断言 `closer, ok := writeSyncer.(io.Closer)` **永远成功**。

### 第二层：两种模式都会触发

这个 bug 不仅在文件模式下有，在纯 stdout 模式下同样存在：

**stdout 模式时**：
```
Init("info")  → writeSyncer = AddSync(os.Stdout)（直接持有 *os.File）
Init("debug") → shutdownCurrent()
                writeSyncer.(io.Closer) → true
                closer.Close() → *os.File.Close() → 关闭 fd 1
                writeSyncer = AddSync(os.Stdout)（新的 *os.File，但持有同一 fd）
                ↓
                写入 → "write /dev/stdout: file already closed"
```

**文件模式时**：
```
Init(filename="app.log")
  → writeSyncer = MultiWriteSyncer(AddSync(os.Stdout), AddSync(lumberjackLogger))
  → Init(filename="app.log") 再次调用
  → shutdownCurrent()
      → MultiWriteSyncer.Close()
          → AddSync(os.Stdout).Close() → 关闭 fd 1
```

**关键结论**：无论哪种模式，只要 `shutdownCurrent` 调用 `Close()`，os.Stdout 的 fd 都会被关闭。纯 stdout 模式的 bug 隐藏得更深——因为之前没有"纯 stdout 二次 Init"的测试场景。

### 第三层：zap 的 Sync 错误无限递归

即使 `Close()` 不触发，fd 关闭后 `Sync()` 也会失败：

```
os.Stdout.Sync() → EBADF（fd 已关闭）
  → zap 的 internal logger 捕获错误
    → internal logger 也写 stdout
      → 写操作 → 触发底层 Sync()
        → 再次 EBADF
          → 无限递归
```

这就是测试输出中大量 `write error: write /dev/stdout: file already closed` 的来源——zap 的内部日志系统陷入了死循环。

---

## 四、修复方案演进

### 方案一：哨兵身份比较（不完整）

```go
stdoutSyncer := zapcore.AddSync(os.Stdout)  // 包级变量

func shutdownCurrent() {
    if writeSyncer != nil && writeSyncer != stdoutSyncer {
        if closer, ok := writeSyncer.(io.Closer); ok {
            _ = closer.Close()
        }
    }
}
```

**缺陷**：文件模式下 `writeSyncer = MultiWriteSyncer(AddSync(os.Stdout), ...)`，`MultiWriteSyncer != stdoutSyncer` → 进入 Close 分支 → `MultiWriteSyncer.Close()` 递归调用子 syncer 的 Close → stdout 还是被关。

### 方案二：neverCloseSyncer 包装（最终采用）

```go
type neverCloseSyncer struct {
    zapcore.WriteSyncer
}

func (n neverCloseSyncer) Close() error { return nil }

// Sync() 根据底层资源类型决定行为：
//   - 底层为 os.Stdout（fd=1）或 os.Stderr（fd=2）时返回 nil（切断递归链）
//   - 其他资源透传真实 Sync 错误（允许感知磁盘 I/O 异常）
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

**原理**：

1. `neverCloseSyncer.Close()` 是空操作 → `shutdownCurrent` 调用 Close 不影响 stdout
2. `neverCloseSyncer.Sync()` 有条件返回 nil → stdout/stderr 抑制错误切断递归链；普通文件透传错误

**初始版本的局限**：最初 `Sync()` 无条件返回 nil，这会吞掉日志文件的同步错误（如磁盘满）。改进后的 `Sync()` 通过 `unwrapToFile` 递归解包到最底层的 `*os.File`，只有 fd=1/2 才返回 nil，对普通文件（fd≥3）透传真实错误。

**为什么纯 stdout 模式也需要包装？**

纯 stdout 模式下 `writeSyncer = AddSync(os.Stdout)` 是 `*os.File`，`shutdownCurrent` 的 `closer.Close()` 会直接调用 `os.Stdout.Close()`，关闭进程级的 fd 1。两次 `Init()` 就触发 bug——之前没发现是因为没有"纯 stdout 模式二次 Init"的测试场景。

### 方案三：不调用 Close，只调用 Sync

放弃 `shutdownCurrent` 中的 `Close()` 调用，依赖 OS 进程退出时自动关闭 fd。

**缺陷**：lumberjack 打开的文件句柄在反复 re-Init 场景下会泄漏（文件描述符累积）。

---

## 五、并发安全性分析

### neverCloseSyncer 自身的并发安全

`neverCloseSyncer` **无任何可变状态**——所有字段都是嵌入的 `zapcore.WriteSyncer`，自身方法全是透传或空操作，**天然并发安全**。

```go
type neverCloseSyncer struct {
    zapcore.WriteSyncer  // 嵌入，不持有可变状态
}
func (n neverCloseSyncer) Close() error { return nil }  // 无操作
func (n neverCloseSyncer) Sync() error  { return nil }  // 无操作
// Write() / 等透传到被包装的 syncer——由被包装方保证
```

### 多个 goroutine 同时调用 Init 的并发安全

`InitWithConfig` 中用 `muLevel.Lock()` 保护 `shutdownCurrent` + 新建 logger 的整个过程：

```go
muLevel.Lock()
defer muLevel.Unlock()
shutdownCurrent()  // 原子关闭旧 logger
atomicLevel = ...
encoder = ...
writeSyncer = ...  // 新值对其他 goroutine 不可见，直到 Unlock
log.Store(...)
```

`log` 是 `atomic.Pointer[zap.Logger]`，`Store()` 原子写入。`Get()` 用 `Load()` 原子读取。所以：

- goroutine A 在 `Lock` → `shutdownCurrent` → `writeSyncer = 新值` 过程中，B goroutine 看到的 `writeSyncer` 仍是旧值（或 nil）
- B 的 `Get()` 看到的是旧 logger（或 nil fallback）
- Unlock 后，B 再次调用时看到新值

**结论**：Init 的并发写入安全由 `muLevel` 写锁保证，不会出现"一个 goroutine 关 stdout，另一个 goroutine 写 stdout 失败"的情况。

### 多个 goroutine 同时写日志的并发安全

`Get()` → `logger.Info(...)` → `zapcore.Core.Write()` → `syncer.Write()`

- `log.Load()` 原子读取，返回的 `*zap.Logger` 是并发安全的（zap logger 内部已做同步）
- `zapcore.AddSync(os.Stdout)` 底层的 `*os.File.Write` 是系统调用，OS 保证原子性（小于 PIPE_BUF 时）

**结论**：日志写入并发安全。

### 生产环境中的竞态问题

生产环境中最可能的并发场景是"日志热重载"——在服务运行期间动态修改日志配置：

```go
// goroutine A：热重载线程
SetLevel("debug")           // atomicLevel.SetLevel() 线程安全

// goroutine B：HTTP 请求处理线程
logger.Info("request received")  // 读写不同的包级变量，不冲突
```

`SetLevel` 只修改 `atomicLevel`，不涉及 `writeSyncer` 和 `encoder`，线程安全。`SetLevel` 和 `Get()` 可以并发执行无问题。

**唯一需要注意的边界**：日志热重载（InitWithConfig 在运行时被调用）时，`muLevel` 写锁会阻塞所有新的日志写入，但通常日志重载是低频操作，可以接受短暂阻塞。

### 业界解法对比

| 方案 | 代表框架 | 思路 |
|------|---------|------|
| 依赖注入 | `slog.Logger` | 不使用全局变量，logger 作为参数传入，由调用方管理生命周期，根本上避免资源泄漏 |
| 写时复制（Copy-on-Write） | logrus 的 `New()` | 创建新 logger 后替换指针，读端无锁，写端需锁，但锁只在替换时持有 |
| 读写锁保护共享状态 | 本项目的 `muLevel` | 用写锁保护 init 过程，读端（Get/SetLevel）无锁 |
| no-op 包装 | 本项目的 `neverCloseSyncer` | 装饰既有资源，修改部分方法行为，不改变其他行为 |
| 只用 Sync 不用 Close | 部分日志库 | 放弃主动关闭，依赖 OS 进程退出时关闭，有 fd 泄漏风险 |

`neverCloseSyncer` 方案的优势：**在保留现有架构（全局 logger + 热重载）的前提下，用最小改动解决了问题**，不引入新的复杂性。

---

## 六、核心设计原则

### 1. 谁创建，谁负责关闭

`shutdownCurrent` 的职责是释放"本次 Init 创建的资源"。`os.Stdout` 是进程启动时 OS 打开的，不是 Init 创建的，不应该由 Init 负责关闭。

### 2. 进程级共享资源的保护契约

`os.Stdout`、`os.Stderr`、`os.Stdin` 是进程级别的共享资源。这些资源的生命周期由 OS 和进程管理，**库代码无权关闭**。

### 3. 接口组合与意外暴露

`WriteSyncer` 接口嵌入了多个子接口，使用接口断言时应该只使用自己需要的方法：

```go
// zapcore.WriteSyncer 嵌入了 io.Writer 和 io.Syncer
// 但某些 syncer（如 *os.File）同时实现了 io.Closer
// 使用 .(io.Closer) 断言时必须明确：这个 Close 是不是应该调用的
```

### 4. 空操作（no-op）包装模式

当需要"告诉某个系统不要对某个资源做某操作"时，空操作包装是最常用的模式：

```go
// 禁止关闭
type noClose struct{ io.ReadCloser }
func (noClose) Close() error { return nil }

// 禁止同步
type noSync struct{ zapcore.WriteSyncer }
func (noSync) Sync() error { return nil }
```

适用场景：装饰既有资源但只改变部分行为、测试时 mock 掉不需要验证的方法、隔离副作用边界。

---

## 七、面试话术

### 问题一：这个 bug 现象是什么？

运行时日志热重载或测试反复初始化后，`go test` 报错 `write /dev/stdout: file already closed`，所有向 stdout 的写操作全部失败，包括测试覆盖率报告生成。

### 问题二：根因是什么？

`shutdownCurrent` 中无条件调用 `writeSyncer.(io.Closer).Close()`，而 `os.Stdout` 本身就是一个 `*os.File`，实现了 `Close()` 方法，调用它会关闭进程级的 fd 1。fd 关闭后无法恢复，所有后续写入均失败。

### 问题三：为什么之前没发现？

只有在**同进程内两次 Init 调用**的场景下才暴露。纯开发测试通常只调用一次 Init，或者每次初始化的是独立进程。覆盖率测试和日志热重载场景才触发。

### 问题四：修复的核心思路是什么？

引入 `neverCloseSyncer` 包装器，将 stdout 的 `Close()` 和 `Sync()` 方法变成空操作，从类型层面保证"不会关闭 stdout"：

```go
type neverCloseSyncer struct { zapcore.WriteSyncer }
func (n neverCloseSyncer) Close() error { return nil }
func (n neverCloseSyncer) Sync() error  { return nil }
```

这样无论 `shutdownCurrent` 怎么调用 `Close()`，stdout 的 fd 都不会受影响。

### 问题五：为什么需要覆盖 Sync()？它现在还会透传真实错误吗？

`Sync()` 在 fd 关闭后返回 EBADF，zap 的 `Sync()` 失败后会调用内部错误日志记录，内部 logger 又尝试写 stdout，再次触发 Sync() 失败——**无限递归**。覆盖 `Sync()` 返回 nil，切断这个递归链。

更进一步的改进：当前实现通过 `unwrapToFile` 递归解包到底层 `*os.File`，**只有 fd=1（stdout）或 fd=2（stderr）时才返回 nil**。对于普通文件（fd≥3），Sync 错误会**透传给调用方**，允许上层感知磁盘 I/O 异常。

为什么这样设计？因为 os.Stdout/os.Stderr 是无缓冲终端 fd，关闭后同步无意义；而日志文件（lumberjack）同步失败可能意味着磁盘满等严重问题，应该让调用方知道。

### 问题六：并发安全性如何？

`neverCloseSyncer` 自身无可变状态，完全并发安全。`Init` 的并发写入由 `muLevel` 写锁保护，原子操作保证读端（`Get()` / `SetLevel()`）无锁并发安全。生产环境的日志热重载场景完全可接受。

### 问题七：这个模式还能用在哪里？

- 数据库连接池：`Close()` 归还连接到池而非关闭 socket
- 共享的 HTTP 客户端：`Close()` 不关闭底层 TCP 连接
- 包装 `bufio.Reader`：`Read()` 后不应关闭底层流
- 测试场景：mock 掉不需要验证的方法

---

## 八、相关代码位置

| 文件 | 行号 | 说明 |
|------|------|------|
| `pkg/logger/logger.go` | 68-103 | `neverCloseSyncer` 定义（覆盖 Close + 有条件 Sync） |
| `pkg/logger/logger.go` | 105-111 | `getUnderlyingFile` 递归解包辅助方法 |
| `pkg/logger/logger.go` | 113-159 | `unwrapToFile` 核心解包函数（支持 neverCloseSyncer / multiWriteSyncer / lockedWriteSyncer） |
| `pkg/logger/logger.go` | 200-211 | InitWithConfig 中两种模式均使用 neverCloseSyncer |
| `pkg/logger/logger.go` | 298-312 | shutdownCurrent（干净版本） |
| `pkg/logger/logger_test.go` | 134-415 | 22 个测试（9 原有 + 13 新增：Sync 行为区分 / fallback / unwrapToFile） |

---

## 九、测试场景覆盖（共 22 个）

**原始功能测试（9 个）**
| 测试 | 验证内容 |
|------|----------|
| `TestRequestIDContext` | 上下文 ID 读写 |
| `TestInit` / `TestInitWithConfig` | 初始化和配置 |
| `TestGet` / `TestSetLevel` | 动态级别调整 |
| `TestCallerInBusinessCode` | 调用方信息正确 |
| `TestFieldConstructors` | Field 构造函数 |
| `TestSync` | 刷新缓冲区 |

**neverCloseSyncer stdout 保护（8 个）**
| 测试 | 验证内容 |
|------|----------|
| `TestNeverCloseSyncer_StdoutModeSwitch` | 纯 stdout 模式多次 re-Init |
| `TestNeverCloseSyncer_StdoutToFileModeSwitch` | stdout → 文件模式切换 |
| `TestNeverCloseSyncer_FileToStdoutModeSwitch` | 文件 → stdout 模式切换 |
| `TestNeverCloseSyncer_RepeatedSwitches` | 6 次反复切换（stdout↔file） |
| `TestNeverCloseSyncer_MultiFileModeInit` | 文件模式 5 次 re-Init |
| `TestNeverCloseSyncer_ConcurrentInit` | 10 goroutine 并发切换 |
| `TestNeverCloseSyncer_IsSafe` | 50 goroutine 并发 Write/Sync/Close |
| `TestNeverCloseSyncer_CloseIsNoOp` | Close 前后可写性不变 |

**Sync() 行为区分（5 个）**
| 测试 | 验证内容 |
|------|----------|
| `TestNeverCloseSyncer_Sync_PassThroughErrors` | 非 stdout/stderr 的 Sync 错误透传 |
| `TestNeverCloseSyncer_Sync_SuppressStdoutErrors` | stdout 的 Sync 返回 nil |
| `TestNeverCloseSyncer_Sync_SuppressStderrErrors` | stderr 的 Sync 返回 nil |
| `TestNeverCloseSyncer_Sync_RegularFilePassThrough` | 普通文件的 Sync 错误透传 |
| `TestNeverCloseSyncer_Sync_NilUnderlying` | 底层为 nil 时安全退化 |

**Fallback 与解包（5 个）**
| 测试 | 验证内容 |
|------|----------|
| `TestInitWithConfig_InvalidLevelFallback` | 无效级别 fallback 到 InfoLevel |
| `TestGet_FallbackWhenNil` | log 为 nil 时返回 fallback |
| `TestGetLevel_DefaultWhenNotInitialized` | 未初始化时默认 info |
| `TestUnwrapToFile_DirectFile` | 解包直接 *os.File（stdout/stderr/普通文件） |
| `TestUnwrapToFile_NeverCloseSyncerWrapper` | 解包 neverCloseSyncer 包装的 stdout |
| `TestUnwrapToFile_UnknownType` | 不支持的类型返回 false |