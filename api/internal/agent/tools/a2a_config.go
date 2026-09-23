package tools

// A2AConfig A2A 配置
type A2AConfig struct {
	// Agents A2A Agent 列表
	Agents []A2AAgent `json:"agents"`
}

// A2AAgent A2A Agent 配置
type A2AAgent struct {
	// Name Agent 名称
	Name string `json:"name"`
	// URL Agent 服务地址
	URL string `json:"url"`
}
