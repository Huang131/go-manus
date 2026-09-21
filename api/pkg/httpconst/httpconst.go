// Package httpconst 集中存放跨领域共用的 HTTP 协议常量，
// 避免各业务子包（llm/a2a/search/sandbox 等）互相反向依赖导致循环导入。
package httpconst

// 常用 HTTP MIME 类型（Content-Type / Accept 头取值）。
const (
	ContentTypeJSON = "application/json"
	ContentTypeSSE  = "text/event-stream"
)

// AuthBearerPrefix 是 Authorization 头 Bearer 认证方案的固定前缀。
const AuthBearerPrefix = "Bearer "
