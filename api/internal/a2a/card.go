package a2a

import "fmt"

// A2AAgentCard A2A Agent 卡片信息。
//
// 兼容两版 Agent Card 结构：
//   - v0.3：调用端点位于根层 `url` 字段；
//   - v1.0：调用端点位于 `supportedInterfaces[]`，每个 interface 通过 `protocolBinding`
//     声明支持的协议绑定（JSONRPC / HTTP+JSON）。
//
// 解析端点统一走 ResolveEndpoint，不要在别处直接读 URL / SupportedInterfaces。
type A2AAgentCard struct {
	Name                string                 `json:"name"`                          // Agent 名称
	Description         string                 `json:"description,omitempty"`         // Agent 描述
	URL                 string                 `json:"url,omitempty"`                 // v0.3 调用端点（兜底）
	Version             string                 `json:"version,omitempty"`             // Agent 版本
	ProtocolVersion     string                 `json:"protocolVersion,omitempty"`     // v0.3 协议版本，如 "0.3.0"
	SupportedInterfaces []A2AInterface         `json:"supportedInterfaces,omitempty"` // v1.0 支持的协议接口
	Capabilities        *A2AAgentCapabilities  `json:"capabilities,omitempty"`        // Agent 能力
	Skills              []A2AAgentSkill        `json:"skills,omitempty"`              // Agent 技能列表
	Metadata            map[string]interface{} `json:"metadata,omitempty"`            // 元数据
	Enabled             bool                   `json:"enabled,omitempty"`             // 是否启用（go-manus 本地字段，非协议字段）
}

// A2AInterface A2A 支持的协议接口（v1.0）。
type A2AInterface struct {
	URL             string `json:"url"`                       // 该接口的调用端点
	ProtocolBinding string `json:"protocolBinding,omitempty"` // "JSONRPC" | "HTTP+JSON"
	ProtocolVersion string `json:"protocolVersion,omitempty"` // 接口协议版本，如 "1.0"
}

// A2AAgentCapabilities Agent 能力。
//
// 兼容两版命名：v1.0 使用 `streaming`，v0.3 使用 `supportsStreaming`。
// 调用方优先读 SupportsStreaming() 聚合方法，不要直接读字段。
type A2AAgentCapabilities struct {
	Streaming                 bool `json:"streaming,omitempty"`                 // v1.0 流式
	PushNotifications         bool `json:"pushNotifications,omitempty"`         // v1.0 推送通知
	SupportsStreaming         bool `json:"supportsStreaming,omitempty"`         // v0.3 流式
	SupportsPushNotifications bool `json:"supportsPushNotifications,omitempty"` // v0.3 推送通知
}

// CanStream 返回是否支持流式，兼容 v1.0（streaming）/ v0.3（supportsStreaming）两版字段名。
func (c *A2AAgentCapabilities) CanStream() bool {
	return c != nil && (c.Streaming || c.SupportsStreaming)
}

// A2AAgentSkill Agent 技能。
type A2AAgentSkill struct {
	ID          string `json:"id,omitempty"`          // 技能 ID
	Name        string `json:"name,omitempty"`        // 技能名称
	Description string `json:"description,omitempty"` // 技能描述
}

// ResolveEndpoint 解析用于 JSON-RPC 调用的端点和协议版本。
//
// 优先级：v1.0 的 supportedInterfaces[]（ProtocolBinding == "JSONRPC"）→ v0.3 的根层 URL。
// 都缺失时返回错误，让调用方在初始化阶段即感知，而不是到 Invoke 才发现端点为空。
func (c *A2AAgentCard) ResolveEndpoint() (string, string, error) {
	if c == nil {
		return "", "", fmt.Errorf("Agent Card 为空")
	}

	// 1. v1.0：从 supportedInterfaces 找 JSONRPC 绑定。
	for _, iface := range c.SupportedInterfaces {
		if iface.URL == "" {
			continue
		}
		// 未声明 protocolBinding 的老实现默认视为 JSONRPC。
		if iface.ProtocolBinding == "" || iface.ProtocolBinding == a2aProtocolBindingJSONRPC {
			return iface.URL, iface.ProtocolVersion, nil
		}
	}

	// 2. v0.3：回退根层 URL。
	if c.URL != "" {
		return c.URL, c.ProtocolVersion, nil
	}

	return "", "", fmt.Errorf("Agent Card 缺少 JSON-RPC 调用端点")
}
