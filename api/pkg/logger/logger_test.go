package logger

import (
	"testing"
)

func TestInit(t *testing.T) {
	// 测试日志初始化
	if err := Init("debug"); err != nil {
		t.Errorf("Init() error = %v", err)
	}
}

func TestGet(t *testing.T) {
	// 先初始化
	Init("info")

	log := Get()
	if log == nil {
		t.Error("Get() should not return nil")
	}
}

func TestDebug(t *testing.T) {
	Init("debug")
	// 不应 panic
	Debug("test debug message")
}

func TestInfo(t *testing.T) {
	Init("info")
	// 不应 panic
	Info("test info message")
}
