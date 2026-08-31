package model

// SearchResultItem 搜索结果条目
type SearchResultItem struct {
	URL     string `json:"url"`     // 搜索条目 URL 地址
	Title   string `json:"title"`   // 搜索条目标题
	Snippet string `json:"snippet"` // 搜索条目简介
}

// SearchResults 搜索结果数据模型
type SearchResults struct {
	Query        string             `json:"query"`         // 用户的搜索词
	DateRange    string             `json:"date_range"`    // 日期检索范围
	TotalResults int                `json:"total_results"` // 搜索结果总条数
	Results      []SearchResultItem `json:"results"`       // 搜索结果列表
}
