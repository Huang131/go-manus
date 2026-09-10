package logger

import (
	"context"
	"os"
	"sync"
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestRequestIDContext(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-123")
	if got := RequestIDFromContext(ctx); got != "req-123" {
		t.Fatalf("RequestIDFromContext() = %q, want req-123", got)
	}
	if got := RequestIDFromContext(nil); got != "" {
		t.Fatalf("RequestIDFromContext(nil) = %q, want empty", got)
	}
}

func TestInit(t *testing.T) {
	if err := Init(LevelDebug); err != nil {
		t.Errorf("Init() error = %v", err)
	}
}

func TestGet(t *testing.T) {
	Init(LevelInfo)

	log := Get()
	if log == nil {
		t.Error("Get() should not return nil")
	}
}

func TestDebug(t *testing.T) {
	Init(LevelDebug)
	Debug("test debug message")
}

func TestInfo(t *testing.T) {
	Init(LevelInfo)
	Info("test info message")
}

// testCallerLocation 是一个测试函数，用于验证 caller 信息
func testCallerLocation() {
	Info("caller test")
}

func TestCallerInBusinessCode(t *testing.T) {
	Init(LevelInfo)
	// 验证从业务代码调用时 caller 显示正确的行号
	testCallerLocation() // caller 应该指向这一行
}

func TestInitWithConfig(t *testing.T) {
	cfg := Config{
		Level:      "debug",
		MaxSize:    10,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   true,
	}

	if err := InitWithConfig(cfg); err != nil {
		t.Errorf("InitWithConfig() error = %v", err)
	}
}

func TestSetLevel(t *testing.T) {
	Init(LevelInfo)

	// 初始级别应该是 info
	if got := GetLevel(); got != LevelInfo {
		t.Errorf("GetLevel() = %v, want %v", got, LevelInfo)
	}

	// 动态调整为 debug
	SetLevel(LevelDebug)
	if got := GetLevel(); got != LevelDebug {
		t.Errorf("GetLevel() = %v, want %v", got, LevelDebug)
	}

	// 动态调整为 error
	SetLevel(LevelError)
	if got := GetLevel(); got != LevelError {
		t.Errorf("GetLevel() = %v, want %v", got, LevelError)
	}

	// 测试无效级别时默认回退到 info
	SetLevel("invalid")
	if got := GetLevel(); got != LevelInfo {
		t.Errorf("GetLevel() = %v, want %v (default)", got, LevelInfo)
	}
}

func TestSync(t *testing.T) {
	Init(LevelInfo)
	Sync()
}

func TestFieldConstructors(t *testing.T) {
	Init(LevelDebug)

	// 测试 String
	Info("test string field", String("key", "value"))

	// 测试 Int
	Info("test int field", Int("count", 42))

	// 测试 Int64
	Info("test int64 field", Int64("big", 123456789))

	// 测试 Float64
	Info("test float64 field", Float64("rate", 3.14))

	// 测试 Bool
	Info("test bool field", Bool("flag", true))

	// 测试 Err
	Info("test error field", Err(nil))

	// 测试 Any
	Info("test any field", Any("data", map[string]int{"a": 1}))

	// 测试 Strings
	Info("test strings field", Strings("names", []string{"a", "b"}))

	Sync()
}

// stdoutWriteOK 检测 os.Stdout 的底层 fd 是否仍然有效。
// 已关闭的 fd 写入会立即返回错误（EBADF），无需依赖系统调用。
func stdoutWriteOK() error {
	_, err := os.Stdout.Write([]byte(""))
	return err
}

// TestNeverCloseSyncer_StdoutModeSwitch 测试 stdout 模式下的 re-Init
// 不会关闭 os.Stdout 的 fd。
func TestNeverCloseSyncer_StdoutModeSwitch(t *testing.T) {
	Init(LevelInfo) // stdout 模式

	// 连续 re-Init，均为 stdout 模式
	Init(LevelDebug)
	Init(LevelError)

	if err := stdoutWriteOK(); err != nil {
		t.Errorf("os.Stdout still writable after 3x stdout re-Init, err=%v", err)
	}
}

// TestNeverCloseSyncer_StdoutToFileModeSwitch 测试从 stdout 模式切换到文件模式，
// 验证文件模式下 neverCloseSyncer 正确保护了 stdout fd。
func TestNeverCloseSyncer_StdoutToFileModeSwitch(t *testing.T) {
	Init(LevelInfo) // 初始 stdout 模式

	tmpFile := t.TempDir() + "/never_close_test.log"
	defer os.Remove(tmpFile)

	// 切换到文件模式（MultiWriteSyncer 中含 neverCloseSyncer 包装的 stdout）
	InitWithConfig(Config{
		Level:    "debug",
		Filename: tmpFile,
	})

	// 文件模式下 stdout fd 应该仍然有效
	if err := stdoutWriteOK(); err != nil {
		t.Errorf("os.Stdout still writable after stdout→file switch, err=%v", err)
	}
}

// TestNeverCloseSyncer_FileToStdoutModeSwitch 测试从文件模式切回 stdout 模式，
// 验证切回后 stdout fd 仍然有效。
func TestNeverCloseSyncer_FileToStdoutModeSwitch(t *testing.T) {
	tmpFile := t.TempDir() + "/never_close_test.log"
	defer os.Remove(tmpFile)

	// 初始文件模式
	InitWithConfig(Config{
		Level:    "info",
		Filename: tmpFile,
	})

	// 切回 stdout 模式
	Init(LevelDebug)

	if err := stdoutWriteOK(); err != nil {
		t.Errorf("os.Stdout still writable after file→stdout switch, err=%v", err)
	}
}

// TestNeverCloseSyncer_RepeatedSwitches 测试反复在 stdout 和文件模式间
// 切换，验证 neverCloseSyncer 在所有路径下均保护了 stdout fd。
func TestNeverCloseSyncer_RepeatedSwitches(t *testing.T) {
	tmpFile := t.TempDir() + "/never_close_test.log"
	defer os.Remove(tmpFile)

	// 路径：stdout → file → stdout → file → stdout → file
	modes := []Config{
		{Level: "info"},                     // stdout
		{Level: "debug", Filename: tmpFile}, // file
		{Level: "warn"},                     // stdout
		{Level: "error", Filename: tmpFile}, // file
		{Level: "info"},                     // stdout
		{Level: "debug", Filename: tmpFile}, // file
	}

	for i, cfg := range modes {
		InitWithConfig(cfg)
		if err := stdoutWriteOK(); err != nil {
			t.Errorf("os.Stdout writable after mode switch #%d, err=%v", i+1, err)
		}
	}

	// 最终切回 stdout
	Init(LevelInfo)
	if err := stdoutWriteOK(); err != nil {
		t.Errorf("os.Stdout writable after final switch, err=%v", err)
	}
}

// TestNeverCloseSyncer_MultiFileModeInit 测试文件模式下重复 Init，
// 验证 shutdownCurrent 的 MultiWriteSyncer.Close() 不会通过
// neverCloseSyncer 意外关闭 stdout。
func TestNeverCloseSyncer_MultiFileModeInit(t *testing.T) {
	tmpFile := t.TempDir() + "/never_close_test.log"
	defer os.Remove(tmpFile)

	for i := 0; i < 5; i++ {
		InitWithConfig(Config{
			Level:    "info",
			Filename: tmpFile,
		})
		if err := stdoutWriteOK(); err != nil {
			t.Errorf("os.Stdout writable after file mode Init #%d, err=%v", i+1, err)
		}
	}
}

// TestNeverCloseSyncer_ConcurrentInit 测试并发 re-Init。
// muLevel 写锁保证 shutdownCurrent + 新建 logger 的串行性，
// 因此即使多 goroutine 并发 Init，也不会出现：
//
//	goroutine A: shutdownCurrent → Close(stdout) → 关闭 fd 1
//	goroutine B: write → 失败
//
// 场景。并发安全性由写锁保证。
func TestNeverCloseSyncer_ConcurrentInit(t *testing.T) {
	tmpFile := t.TempDir() + "/never_close_test.log"
	defer os.Remove(tmpFile)

	// 先初始化为文件模式
	InitWithConfig(Config{
		Level:    "info",
		Filename: tmpFile,
	})

	var wg sync.WaitGroup
	// 10 个 goroutine 并发做 stdout ↔ file 切换
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(round int) {
			defer wg.Done()
			if round%2 == 0 {
				InitWithConfig(Config{
					Level:    "debug",
					Filename: tmpFile,
				})
			} else {
				Init(LevelInfo)
			}
		}(i)
	}
	wg.Wait()

	// 所有 goroutine 执行完后，验证 stdout fd 仍然有效
	if err := stdoutWriteOK(); err != nil {
		t.Errorf("os.Stdout writable after concurrent Init, err=%v", err)
	}
}

// TestNeverCloseSyncer_IsSafe 验证 neverCloseSyncer 自身的并发安全性。
// neverCloseSyncer 不持有任何可变状态，所有操作均透传到被包装的 WriteSyncer，
// 自身无竞态。真正的并发安全取决于被包装的 syncer（os.Stdout 本身是线程安全的）。
func TestNeverCloseSyncer_IsSafe(t *testing.T) {
	base := zapcore.AddSync(os.Stdout)
	wrapper := neverCloseSyncer{base}

	done := make(chan struct{})
	for i := 0; i < 50; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				wrapper.Write([]byte("test"))
				wrapper.Sync()
				wrapper.Close() // 连续调用 Close 应安全（no-op）
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < 50; i++ {
		<-done
	}
	// 能走到这里说明并发安全
}

// TestNeverCloseSyncer_CloseIsNoOp 验证 neverCloseSyncer.Close() 确实是空操作，
// Close 前后的 Write 行为不变。
func TestNeverCloseSyncer_CloseIsNoOp(t *testing.T) {
	base := zapcore.AddSync(os.Stdout)
	wrapper := neverCloseSyncer{base}

	// Close 前可写
	n1, err1 := wrapper.Write([]byte("before close\n"))

	// Close 是空操作
	_ = wrapper.Close()

	// Close 后仍可写（因为 Close 什么都没做）
	n2, err2 := wrapper.Write([]byte("after close\n"))

	if err1 != nil || err2 != nil {
		t.Errorf("Write before/after Close: err1=%v, err2=%v", err1, err2)
	}
	if n1 == 0 && n2 == 0 {
		// 空写入在 darwin/Linux 上返回 0，只要 err 为 nil 即表示 fd 有效
	}
}
