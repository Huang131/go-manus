package mcp

import (
	"testing"
)

// TestMCPClientManager_New 测试创建 MCP 客户端管理器
func TestMCPClientManager_New(t *testing.T) {
	manager := NewMCPClientManager(nil)
	if manager == nil {
		t.Error("NewMCPClientManager should return non-nil manager")
	}
}

// TestMCPClientManager_GetClient 测试获取不存在的客户端
func TestMCPClientManager_GetClient(t *testing.T) {
	manager := NewMCPClientManager(nil)
	_, ok := manager.GetClient("nonexistent")
	if ok {
		t.Error("GetClient should return false for nonexistent client")
	}
}

// TestMCPClientManager_ListAllTools_Empty 测试空管理器获取工具列表
func TestMCPClientManager_ListAllTools_Empty(t *testing.T) {
	manager := NewMCPClientManager(nil)
	tools, err := manager.ListAllTools(nil)
	if err != nil {
		t.Errorf("ListAllTools should not return error for nil config: %v", err)
	}
	if len(tools) != 0 {
		t.Errorf("ListAllTools should return empty map, got %d", len(tools))
	}
}

// TestMCPClientManager_Close 测试关闭空管理器
func TestMCPClientManager_Close(t *testing.T) {
	manager := NewMCPClientManager(nil)
	err := manager.Close()
	if err != nil {
		t.Errorf("Close should not return error for nil config: %v", err)
	}
}
