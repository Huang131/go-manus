package search

import "time"

// 搜索 provider 协议值：Tavily / Bocha 的厂商枚举值。
const (
	tavilySearchDepthBasic = "basic"
	tavilyTopicGeneral     = "general"
	tavilyTimeRangeDay     = "day"
	tavilyTimeRangeWeek    = "week"
	tavilyTimeRangeMonth   = "month"
	tavilyTimeRangeYear    = "year"

	bochaFreshnessOneDay   = "oneDay"
	bochaFreshnessOneWeek  = "oneWeek"
	bochaFreshnessOneMonth = "oneMonth"
	bochaFreshnessOneYear  = "oneYear"
)

// 搜索客户端通用运行参数
const (
	// 是搜索客户端缺省的 HTTP 请求超时。
	defaultSearchHTTPTimeout = 30 * time.Second

	// 限制各搜索 API 响应体大小，防止上游返回超大 payload 耗尽内存。
	MaxResponseBodyBytes = 1 << 20 // 1 MiB
)
