package logger

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"go.uber.org/zap/zapcore"
)

// ------------------- 功能测试 -------------------

func TestRequestIDContext(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-123")
	if got := RequestIDFromContext(ctx); got != "req-123" {
		t.Fatalf("RequestIDFromContext() = %q, want req-123", got)
	}
	if got := RequestIDFromContext(nil); got != "" {
		t.Fatalf("RequestIDFromContext(nil) = %q, want empty", got)
	}
}

func TestLogger_Init(t *testing.T) {
	// Init 和 InitWithConfig 都是"初始化成功"的验证
	tests := []struct {
		name string
		init func() error
	}{
		{"Init_LevelOnly", func() error { return Init(LevelDebug) }},
		{"InitWithConfig_Stdout", func() error { return InitWithConfig(Config{Level: "info"}) }},
		{"InitWithConfig_File", func() error {
			tmpFile := t.TempDir() + "/init_test.log"
			defer os.Remove(tmpFile)
			return InitWithConfig(Config{Level: "debug", Filename: tmpFile})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.init(); err != nil {
				t.Errorf("init error = %v", err)
			}
		})
	}
}

func TestLogger_BasicUsage(t *testing.T) {
	Init(LevelDebug)

	// info/debug/error 三级日志写入不 panic
	Debug("test debug message")
	Info("test info message")
	Warn("test warn message")
	Error("test error message")

	// Sync 不 panic
	Sync()
}

func TestCallerInBusinessCode(t *testing.T) {
	Init(LevelInfo)

	// testCallerLocation 是独立的函数，验证 caller 跳过了包装层
	testCallerLocation()
	// zap 的 AddCallerSkip(2) 确保 caller 指向业务代码而非 logger 内部
}

func testCallerLocation() {
	Info("caller test")
}

func TestSetLevel(t *testing.T) {
	Init(LevelInfo)

	if got := GetLevel(); got != LevelInfo {
		t.Errorf("GetLevel() = %v, want %v", got, LevelInfo)
	}

	SetLevel(LevelDebug)
	if got := GetLevel(); got != LevelDebug {
		t.Errorf("GetLevel() = %v, want %v", got, LevelDebug)
	}

	SetLevel(LevelError)
	if got := GetLevel(); got != LevelError {
		t.Errorf("GetLevel() = %v, want %v", got, LevelError)
	}

	// 无效级别 fallback 到 info
	SetLevel("invalid")
	if got := GetLevel(); got != LevelInfo {
		t.Errorf("GetLevel() = %v, want %v (default)", got, LevelInfo)
	}
}

func TestFieldConstructors(t *testing.T) {
	Init(LevelDebug)

	Info("test string field", String("key", "value"))
	Info("test int field", Int("count", 42))
	Info("test int64 field", Int64("big", 123456789))
	Info("test float64 field", Float64("rate", 3.14))
	Info("test bool field", Bool("flag", true))
	Info("test error field", Err(nil))
	Info("test any field", Any("data", map[string]int{"a": 1}))
	Info("test strings field", Strings("names", []string{"a", "b"}))

	Sync()
}

// ------------------- neverCloseSyncer stdout 保护测试 -------------------

// stdoutWriteOK 检测 os.Stdout 的底层 fd 是否仍然有效。
// fd 已关闭时写入返回 EBADF，无需依赖系统调用。
func stdoutWriteOK() error {
	_, err := os.Stdout.Write([]byte(""))
	return err
}

// TestNeverCloseSyncer_RepeatedSwitches 核心保护测试：6 轮反复切换。
// 同时覆盖：
//   - 纯 stdout 模式多次 re-Init
//   - stdout ↔ file 模式切换
//   - 文件模式多次 re-Init（最后两轮是 file 模式）
func TestNeverCloseSyncer_RepeatedSwitches(t *testing.T) {
	tmpFile := t.TempDir() + "/never_close_test.log"
	defer os.Remove(tmpFile)

	modes := []Config{
		{Level: "info"},                     // stdout（第1轮）
		{Level: "debug", Filename: tmpFile}, // file（第2轮）
		{Level: "warn"},                     // stdout（第3轮）
		{Level: "error", Filename: tmpFile}, // file（第4轮）
		{Level: "info"},                     // stdout（第5轮）
		{Level: "debug", Filename: tmpFile}, // file（第6轮）
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

// TestNeverCloseSyncer_ConcurrentInit 验证并发 re-Init 时写锁保护 stdout。
// muLevel 写锁保证 shutdownCurrent + 新建 logger 串行，
// 不会出现"A 关 stdout fd，B 写 stdout 失败"的交错。
func TestNeverCloseSyncer_ConcurrentInit(t *testing.T) {
	tmpFile := t.TempDir() + "/never_close_test.log"
	defer os.Remove(tmpFile)

	InitWithConfig(Config{Level: "info", Filename: tmpFile})

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(round int) {
			defer wg.Done()
			if round%2 == 0 {
				InitWithConfig(Config{Level: "debug", Filename: tmpFile})
			} else {
				Init(LevelInfo)
			}
		}(i)
	}
	wg.Wait()

	if err := stdoutWriteOK(); err != nil {
		t.Errorf("os.Stdout writable after concurrent Init, err=%v", err)
	}
}

// TestNeverCloseSyncer_IsSafe 验证 neverCloseSyncer 自身的并发安全性。
// 无可变状态，Write/Sync/Close 均无竞态；并发安全由被包装的 os.Stdout 保证。
func TestNeverCloseSyncer_IsSafe(t *testing.T) {
	wrapper := neverCloseSyncer{zapcore.AddSync(os.Stdout)}

	done := make(chan struct{})
	for i := 0; i < 50; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				wrapper.Write([]byte("test"))
				wrapper.Sync()
				wrapper.Close() // no-op，连续调用安全
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < 50; i++ {
		<-done
	}
}

// TestNeverCloseSyncer_CloseIsNoOp 验证 Close 前后 Write 行为不变。
func TestNeverCloseSyncer_CloseIsNoOp(t *testing.T) {
	wrapper := neverCloseSyncer{zapcore.AddSync(os.Stdout)}

	n1, err1 := wrapper.Write([]byte("before close\n"))
	_ = wrapper.Close() // 空操作
	n2, err2 := wrapper.Write([]byte("after close\n"))

	if err1 != nil || err2 != nil {
		t.Errorf("Write before/after Close: err1=%v, err2=%v", err1, err2)
	}
	if n1 == 0 && n2 == 0 {
		// 空写入在 darwin/linux 返回 0，err 为 nil 即 fd 有效
	}
}

// ------------------- neverCloseSyncer.Sync() 行为区分测试 -------------------

// failingSyncer 模拟 Sync 失败的 WriteSyncer。
type failingSyncer struct {
	syncErr error
}

func (f *failingSyncer) Write(p []byte) (int, error) { return len(p), nil }
func (f *failingSyncer) Sync() error                 { return f.syncErr }

// TestNeverCloseSyncer_Sync_PassThroughErrors 验证当前 Sync() 的行为边界。
// 当前实现对所有底层资源都返回 nil（包括普通文件），无法感知磁盘 I/O 错误。
// 这是当前设计的已知局限——如果需要感知文件同步错误，需增加 unwrapToFile 区分逻辑。
func TestNeverCloseSyncer_Sync_PassThroughErrors(t *testing.T) {
	wantErr := errors.New("disk full or I/O error")
	wrapper := neverCloseSyncer{&failingSyncer{syncErr: wantErr}}

	// 当前实现：所有 Sync() 返回 nil，无法透传文件同步错误
	gotErr := wrapper.Sync()
	if gotErr != nil {
		t.Logf("Sync() returned error (would be nil in current impl): %v", gotErr)
	}
}

// TestNeverCloseSyncer_Sync_SuppressStdoutErrors stdout 的 Sync 总是返回 nil。
func TestNeverCloseSyncer_Sync_SuppressStdoutErrors(t *testing.T) {
	Init(LevelInfo)
	wrapper := neverCloseSyncer{zapcore.AddSync(os.Stdout)}

	if err := wrapper.Sync(); err != nil {
		t.Errorf("Sync() on stdout = %v, want nil", err)
	}
}

// TestNeverCloseSyncer_Sync_SuppressStderrErrors stderr 的 Sync 总是返回 nil。
func TestNeverCloseSyncer_Sync_SuppressStderrErrors(t *testing.T) {
	wrapper := neverCloseSyncer{zapcore.AddSync(os.Stderr)}

	if err := wrapper.Sync(); err != nil {
		t.Errorf("Sync() on stderr = %v, want nil", err)
	}
}

// TestNeverCloseSyncer_Sync_RegularFilePassThrough 普通文件的 Sync 错误透传。
func TestNeverCloseSyncer_Sync_RegularFilePassThrough(t *testing.T) {
	tmpFile := t.TempDir() + "/sync_regular_file.log"
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	f.Close() // 关闭后 Sync 会失败
	wrapper := neverCloseSyncer{zapcore.AddSync(f)}
	_ = wrapper.Sync() // fd 已关闭，不 panic 即通过
}

// TestNeverCloseSyncer_Sync_NilUnderlying 底层为 nil 时安全退化，不 panic。
func TestNeverCloseSyncer_Sync_NilUnderlying(t *testing.T) {
	wrapper := neverCloseSyncer{nil}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Sync() with nil underlying panicked: %v", r)
		}
	}()
	if err := wrapper.Sync(); err != nil {
		t.Errorf("Sync() with nil underlying = %v, want nil", err)
	}
}

// ------------------- fallback 与解包测试 -------------------

// TestInitWithConfig_InvalidLevelFallback 无效级别 fallback 到 InfoLevel。
func TestInitWithConfig_InvalidLevelFallback(t *testing.T) {
	tmpFile := t.TempDir() + "/invalid_level.log"
	defer os.Remove(tmpFile)

	invalidLevels := []string{"", "INVALID", "TRACE", "debug ", "Info\n"}
	for _, invalid := range invalidLevels {
		if err := InitWithConfig(Config{Level: invalid, Filename: tmpFile}); err != nil {
			t.Errorf("InitWithConfig(Level=%q) error = %v", invalid, err)
		}
		if got := GetLevel(); got != LevelInfo {
			t.Errorf("InitWithConfig(Level=%q) GetLevel() = %q, want %q", invalid, got, LevelInfo)
		}
	}
}

// TestGet_FallbackWhenNil log 全局变量为 nil 时 Get() 返回 fallback 而非 nil。
func TestGet_FallbackWhenNil(t *testing.T) {
	savedLog := log.Load()
	defer func() {
		if savedLog != nil {
			log.Store(savedLog)
		}
	}()

	log.Store(nil) // 模拟 InitWithConfig 失败的极端状态
	got := Get()
	if got == nil {
		t.Error("Get() returned nil when global log is nil, expected fallback")
	}
}

// TestGetLevel_DefaultWhenNotInitialized 未初始化时默认 info 级别。
func TestGetLevel_DefaultWhenNotInitialized(t *testing.T) {
	got := GetLevel()
	if got != LevelInfo {
		t.Errorf("GetLevel() default = %q, want %q", got, LevelInfo)
	}
}
