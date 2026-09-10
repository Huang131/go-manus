package model

// SearchResultItem 单条搜索结果
type SearchResultItem struct {
	URL     string `json:"url"`     // 搜索条目 URL 地址
	Title   string `json:"title"`   // 搜索条目标题
	Snippet string `json:"snippet"` // 搜索结果摘要，通常是包含关键词的页面片段
}

// SearchResults 搜索工具的返回结果
type SearchResults struct {
	Query        string             `json:"query"`         // 实际发送的搜索词（可能与用户输入略有不同）
	DateRange    string             `json:"date_range"`    // 时间范围限制，如 "r:7d"（最近7天）
	TotalResults int                `json:"total_results"` // 搜索引擎返回的匹配总数（非实际返回条数）
	Results      []SearchResultItem `json:"results"`       // 实际返回的搜索结果列表
}
