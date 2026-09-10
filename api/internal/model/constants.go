package model

// AppConfigType 标识应用配置所属领域。
type AppConfigType string

const (
	AppConfigTypeLLM   AppConfigType = "llm"
	AppConfigTypeAgent AppConfigType = "agent"
	AppConfigTypeMCP   AppConfigType = "mcp"
	AppConfigTypeA2A   AppConfigType = "a2a"
)

const AppConfigKeyDefault = "default" // 默认配置项的 key

// HealthState 表示组件或应用的健康状态。
type HealthState string

const (
	HealthStateHealthy   HealthState = "healthy"   // 完全健康，所有服务正常
	HealthStateDegraded  HealthState = "degraded"  // 部分降级，非核心功能异常但可降级使用
	HealthStateUnhealthy HealthState = "unhealthy" // 不可用，核心功能异常
	HealthStateSkipped   HealthState = "skipped"   // 跳过检查，服务未启用或配置为免检
)

// ServiceName 标识健康检查中的基础设施服务。
type ServiceName string

const (
	ServiceNamePostgres ServiceName = "postgres"
	ServiceNameRedis    ServiceName = "redis"
	ServiceNameOSS      ServiceName = "oss"
)
