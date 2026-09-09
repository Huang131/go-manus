package model

// AppConfigType 标识应用配置所属领域。
type AppConfigType string

const (
	AppConfigTypeLLM   AppConfigType = "llm"
	AppConfigTypeAgent AppConfigType = "agent"
	AppConfigTypeMCP   AppConfigType = "mcp"
	AppConfigTypeA2A   AppConfigType = "a2a"
)

const AppConfigKeyDefault = "default"

// HealthState 表示组件或应用的健康状态。
type HealthState string

const (
	HealthStateHealthy   HealthState = "healthy"
	HealthStateDegraded  HealthState = "degraded"
	HealthStateUnhealthy HealthState = "unhealthy"
	HealthStateSkipped   HealthState = "skipped"
)

// ServiceName 标识健康检查中的基础设施服务。
type ServiceName string

const (
	ServiceNamePostgres ServiceName = "postgres"
	ServiceNameRedis    ServiceName = "redis"
	ServiceNameOSS      ServiceName = "oss"
)
