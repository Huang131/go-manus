package logger

import (
	"testing"
)

func TestInit(t *testing.T) {
	if err := Init("debug"); err != nil {
		t.Errorf("Init() error = %v", err)
	}
}

func TestGet(t *testing.T) {
	Init("info")

	log := Get()
	if log == nil {
		t.Error("Get() should not return nil")
	}
}

func TestDebug(t *testing.T) {
	Init("debug")
	Debug("test debug message")
}

func TestInfo(t *testing.T) {
	Init("info")
	Info("test info message")
}

// testCallerLocation 是一个测试函数，用于验证 caller 信息
func testCallerLocation() {
	Info("caller test")
}

func TestCallerInBusinessCode(t *testing.T) {
	Init("info")
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
	Init("info")

	// 初始级别应该是 info
	if got := GetLevel(); got != "info" {
		t.Errorf("GetLevel() = %v, want info", got)
	}

	// 动态调整为 debug
	SetLevel("debug")
	if got := GetLevel(); got != "debug" {
		t.Errorf("GetLevel() = %v, want debug", got)
	}

	// 动态调整为 error
	SetLevel("error")
	if got := GetLevel(); got != "error" {
		t.Errorf("GetLevel() = %v, want error", got)
	}

	// 测试无效级别时默认回退到 info
	SetLevel("invalid")
	if got := GetLevel(); got != "info" {
		t.Errorf("GetLevel() = %v, want info (default)", got)
	}
}

func TestSync(t *testing.T) {
	Init("info")
	Sync()
}

func TestFieldConstructors(t *testing.T) {
	Init("debug")

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
