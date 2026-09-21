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

	googleDateRestrictDay   = "d1"
	googleDateRestrictWeek  = "w1"
	googleDateRestrictMonth = "m1"
	googleDateRestrictYear  = "y1"
)

// 搜索客户端通用运行参数。
const (
	// defaultSearchHTTPTimeout 是搜索客户端缺省的 HTTP 请求超时。
	defaultSearchHTTPTimeout = 30 * time.Second
)
