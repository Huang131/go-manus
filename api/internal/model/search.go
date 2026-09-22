package model

// SearchResultItem 单条搜索结果
type SearchResultItem struct {
	URL     string `json:"url"`     // 搜索条目 URL 地址
	Title   string `json:"title"`   // 搜索条目标题
	Snippet string `json:"snippet"` // 搜索结果摘要，通常是包含关键词的页面片段

	// 扩展字段
	Score         float64 `json:"score,omitempty"`          // Tavily 相关性评分 (0-1)
	Summary       string  `json:"summary,omitempty"`        // Bocha AI 生成的摘要
	SiteName      string  `json:"site_name,omitempty"`      // Bocha 网站名称
	SiteIcon      string  `json:"site_icon,omitempty"`      // Bocha 网站图标 URL
	DatePublished string  `json:"date_published,omitempty"` // Bocha 发布时间
}

// SearchResults 搜索工具的返回结果
type SearchResults struct {
	Query          string             `json:"query"`           // 实际发送的搜索词（可能与用户输入略有不同）
	DateRange      string             `json:"date_range"`      // 时间范围限制，如 "r:7d"（最近7天）
	TotalResults   int                `json:"total_results"`   // 实际返回的搜索结果条数（非引擎侧匹配总数）
	TotalEstimated int64              `json:"total_estimated"` // 搜索引擎估计的总匹配数（Bocha）
	ResponseTime   float64            `json:"response_time"`   // Tavily 响应时间（秒）
	Results        []SearchResultItem `json:"results"`         // 实际返回的搜索结果列表
}
