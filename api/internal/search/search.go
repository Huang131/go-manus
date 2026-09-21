package search

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/bytedance/sonic"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/pkg/httpconst"
)

// SearchEngine 搜索引擎接口
type SearchEngine interface {
	// Invoke 调用搜索引擎
	Invoke(ctx context.Context, query string, dateRange *string, limit int) (*model.ToolResult, error)
}

const defaultSearchResultLimit = 10

func normalizeSearchLimit(limit int) int {
	if limit <= 0 {
		return defaultSearchResultLimit
	}
	return limit
}

// resolveSearchTimeout 解析搜索客户端超时：非正值时回退到默认超时。
func resolveSearchTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return defaultSearchHTTPTimeout
	}
	return timeout
}

// toolResultErr 将底层错误映射为 (ToolResult, error)，供搜索客户端统一返回失败结果。
func toolResultErr(err error) (*model.ToolResult, error) {
	return model.NewToolError(err.Error()), err
}

// GoogleSearchClient Google 搜索客户端。
// Google Custom Search 要求同时提供 apiKey 与 searchEngineID（即请求参数 cx），
// 缺一不可；cx 为空时 Google 会以 400 拒绝请求。
type GoogleSearchClient struct {
	apiKey         string
	searchEngineID string
	baseURL        string
	httpClient     *http.Client
}

// NewGoogleSearchClient 创建 Google 搜索客户端
func NewGoogleSearchClient(apiKey, searchEngineID string) *GoogleSearchClient {
	return NewGoogleSearchClientWithTimeout(apiKey, searchEngineID, defaultSearchHTTPTimeout)
}

// NewGoogleSearchClientWithTimeout 创建带请求超时的 Google 客户端。
// baseURL 预置为 Google Custom Search 端点，测试中可覆盖为 httptest 地址。
func NewGoogleSearchClientWithTimeout(apiKey, searchEngineID string, timeout time.Duration) *GoogleSearchClient {
	timeout = resolveSearchTimeout(timeout)
	return &GoogleSearchClient{
		apiKey:         apiKey,
		searchEngineID: searchEngineID,
		baseURL:        "https://www.googleapis.com/customsearch/v1",
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Invoke 调用 Google 搜索
func (c *GoogleSearchClient) Invoke(ctx context.Context, query string, dateRange *string, limit int) (*model.ToolResult, error) {
	if c.apiKey == "" {
		return model.NewToolError("Google API key not configured"), nil
	}
	if c.searchEngineID == "" {
		return model.NewToolError("Google Search Engine ID not configured"), nil
	}

	// 构建请求 URL
	params := url.Values{}
	params.Set("key", c.apiKey)
	params.Set("q", query)
	params.Set("cx", c.searchEngineID)
	params.Set("hl", "zh-CN")
	if limit = normalizeSearchLimit(limit); limit > 10 {
		limit = 10
	}
	params.Set("num", fmt.Sprintf("%d", limit))
	if dateRange != nil {
		params.Set("dateRestrict", *dateRange)
	}

	reqURL := fmt.Sprintf("%s?%s", c.baseURL, params.Encode())

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return toolResultErr(err)
	}

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return toolResultErr(err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return toolResultErr(err)
	}

	if resp.StatusCode != http.StatusOK {
		return model.NewToolError(fmt.Sprintf("Google API error: status=%d, body=%s", resp.StatusCode, string(body))), nil
	}

	// 解析响应
	var googleResp googleSearchResponse
	if err := sonic.Unmarshal(body, &googleResp); err != nil {
		return toolResultErr(err)
	}

	// 转换为标准搜索结果
	results := &model.SearchResults{
		Results: make([]model.SearchResultItem, 0, len(googleResp.Items)),
	}

	for _, item := range googleResp.Items {
		results.Results = append(results.Results, model.SearchResultItem{
			Title:   item.Title,
			URL:     item.Link,
			Snippet: item.Snippet,
		})
	}

	return model.NewToolResultWithMessage("", map[string]interface{}{
		"results": results,
		"total":   googleResp.SearchInformation.TotalResults,
	}), nil
}

// googleSearchResponse Google 搜索响应
type googleSearchResponse struct {
	Kind string `json:"kind"`
	URLs struct {
		Type     string `json:"type"`
		Template string `json:"template"`
	} `json:"url"`
	Queries struct {
		Request []struct {
			Title        string `json:"title"`
			TotalResults string `json:"totalResults"`
			SearchTerms  string `json:"searchTerms"`
			Count        int    `json:"count"`
			StartIndex   int    `json:"startIndex"`
		} `json:"request"`
	} `json:"queries"`
	SearchInformation struct {
		SearchTime            float64 `json:"searchTime"`
		FormattedSearchTime   string  `json:"formattedSearchTime"`
		TotalResults          string  `json:"totalResults"`
		FormattedTotalResults string  `json:"formattedTotalResults"`
	} `json:"searchInformation"`
	Items []struct {
		Kind             string                              `json:"kind"`
		Title            string                              `json:"title"`
		HTMLTitle        string                              `json:"htmlTitle"`
		Link             string                              `json:"link"`
		DisplayLink      string                              `json:"displayLink"`
		Snippet          string                              `json:"snippet"`
		HTMLSnippet      string                              `json:"htmlSnippet"`
		CacheID          string                              `json:"cacheId"`
		FormattedURL     string                              `json:"formattedUrl"`
		HTMLFormattedURL string                              `json:"htmlFormattedUrl"`
		Pagemap          map[string][]map[string]interface{} `json:"pagemap"`
	} `json:"items"`
}

// TavilySearchClient Tavily 搜索客户端。
// Tavily 面向 LLM agent 设计，直连国内可访问，返回已提取的正文内容而非裸链接。
type TavilySearchClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewTavilySearchClient 创建 Tavily 搜索客户端。
func NewTavilySearchClient(apiKey string) *TavilySearchClient {
	return NewTavilySearchClientWithTimeout(apiKey, defaultSearchHTTPTimeout)
}

// NewTavilySearchClientWithTimeout 创建带请求超时的 Tavily 客户端。
func NewTavilySearchClientWithTimeout(apiKey string, timeout time.Duration) *TavilySearchClient {
	timeout = resolveSearchTimeout(timeout)
	return &TavilySearchClient{
		apiKey:  apiKey,
		baseURL: "https://api.tavily.com/search",
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Invoke 调用 Tavily 搜索。dateRange 取值 d/w/m/y，映射为 day/week/month/year。
func (c *TavilySearchClient) Invoke(ctx context.Context, query string, dateRange *string, limit int) (*model.ToolResult, error) {
	if c.apiKey == "" {
		return model.NewToolError("Tavily API key not configured"), nil
	}
	if query == "" {
		return model.NewToolError("search query is empty"), nil
	}

	body, err := sonic.Marshal(tavilySearchRequest{
		Query:       query,
		SearchDepth: tavilySearchDepthBasic,
		MaxResults:  normalizeSearchLimit(limit),
		Topic:       tavilyTopicGeneral,
		TimeRange:   tavilyTimeRange(dateRange),
	})
	if err != nil {
		return toolResultErr(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return toolResultErr(err)
	}
	req.Header.Set("Authorization", httpconst.AuthBearerPrefix+c.apiKey)
	req.Header.Set("Content-Type", httpconst.ContentTypeJSON)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return toolResultErr(err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return toolResultErr(err)
	}
	if resp.StatusCode != http.StatusOK {
		return model.NewToolError(fmt.Sprintf("Tavily API error: status=%d, body=%s", resp.StatusCode, string(respBody))), nil
	}

	var tavilyResp tavilySearchResponse
	if err := sonic.Unmarshal(respBody, &tavilyResp); err != nil {
		return toolResultErr(err)
	}

	results := &model.SearchResults{
		Query:        query,
		TotalResults: len(tavilyResp.Results),
		Results:      make([]model.SearchResultItem, 0, len(tavilyResp.Results)),
	}
	for _, item := range tavilyResp.Results {
		results.Results = append(results.Results, model.SearchResultItem{
			Title:   item.Title,
			URL:     item.URL,
			Snippet: item.Content,
		})
	}

	return model.NewToolResultWithMessage("", map[string]interface{}{
		"results": results,
	}), nil
}

// tavilySearchRequest Tavily 搜索请求体。
type tavilySearchRequest struct {
	Query       string `json:"query"`
	SearchDepth string `json:"search_depth"`
	MaxResults  int    `json:"max_results"`
	Topic       string `json:"topic"`
	TimeRange   string `json:"time_range,omitempty"`
}

// tavilySearchResponse Tavily 搜索响应体。
type tavilySearchResponse struct {
	Results []struct {
		Title   string  `json:"title"`
		URL     string  `json:"url"`
		Content string  `json:"content"`
		Score   float64 `json:"score"`
	} `json:"results"`
}

// BochaSearchClient 博查搜索客户端。
// 国内直连、中文场景最优，作为 Tavily 直连不稳时的付费兜底。
type BochaSearchClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewBochaSearchClient 创建博查搜索客户端。
func NewBochaSearchClient(apiKey string) *BochaSearchClient {
	return NewBochaSearchClientWithTimeout(apiKey, defaultSearchHTTPTimeout)
}

// NewBochaSearchClientWithTimeout 创建带请求超时的博查客户端。
func NewBochaSearchClientWithTimeout(apiKey string, timeout time.Duration) *BochaSearchClient {
	timeout = resolveSearchTimeout(timeout)
	return &BochaSearchClient{
		apiKey:  apiKey,
		baseURL: "https://api.bochaai.com/v1/web-search",
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Invoke 调用博查搜索。dateRange 取值 d/w/m/y，映射为 oneDay/oneWeek/oneMonth/oneYear。
func (c *BochaSearchClient) Invoke(ctx context.Context, query string, dateRange *string, limit int) (*model.ToolResult, error) {
	if c.apiKey == "" {
		return model.NewToolError("Bocha API key not configured"), nil
	}
	if query == "" {
		return model.NewToolError("search query is empty"), nil
	}

	body, err := sonic.Marshal(bochaSearchRequest{
		Query:     query,
		Count:     normalizeSearchLimit(limit),
		Freshness: bochaFreshness(dateRange),
	})
	if err != nil {
		return toolResultErr(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return toolResultErr(err)
	}
	req.Header.Set("Authorization", httpconst.AuthBearerPrefix+c.apiKey)
	req.Header.Set("Content-Type", httpconst.ContentTypeJSON)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return toolResultErr(err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return toolResultErr(err)
	}
	if resp.StatusCode != http.StatusOK {
		return model.NewToolError(fmt.Sprintf("Bocha API error: status=%d, body=%s", resp.StatusCode, string(respBody))), nil
	}

	var bochaResp bochaSearchResponse
	if err := sonic.Unmarshal(respBody, &bochaResp); err != nil {
		return toolResultErr(err)
	}
	if bochaResp.Code != 0 {
		return model.NewToolError(fmt.Sprintf("Bocha API error: code=%d, message=%s", bochaResp.Code, bochaResp.Message)), nil
	}

	values := bochaResp.Data.WebPages.Value
	results := &model.SearchResults{
		Query:        query,
		TotalResults: len(values),
		Results:      make([]model.SearchResultItem, 0, len(values)),
	}
	for _, item := range values {
		results.Results = append(results.Results, model.SearchResultItem{
			Title:   item.Name,
			URL:     item.URL,
			Snippet: item.Snippet,
		})
	}

	return model.NewToolResultWithMessage("", map[string]interface{}{
		"results": results,
	}), nil
}

// bochaSearchRequest 博查搜索请求体。freshness 为空时服务端按 noLimit 处理。
type bochaSearchRequest struct {
	Query     string `json:"query"`
	Count     int    `json:"count"`
	Freshness string `json:"freshness,omitempty"`
}

// bochaSearchResponse 博查搜索响应体。
type bochaSearchResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		WebPages struct {
			Value []struct {
				Name    string `json:"name"`
				URL     string `json:"url"`
				Snippet string `json:"snippet"`
			} `json:"value"`
		} `json:"webPages"`
	} `json:"data"`
}

// tavilyTimeRange 将工具层日期范围缩写（d/w/m/y）映射为 Tavily time_range 取值。
func tavilyTimeRange(r *string) string {
	if r == nil {
		return ""
	}
	switch *r {
	case "d":
		return tavilyTimeRangeDay
	case "w":
		return tavilyTimeRangeWeek
	case "m":
		return tavilyTimeRangeMonth
	case "y":
		return tavilyTimeRangeYear
	default:
		return ""
	}
}

// bochaFreshness 将工具层日期范围缩写（d/w/m/y）映射为博查 freshness 取值。
func bochaFreshness(r *string) string {
	if r == nil {
		return ""
	}
	switch *r {
	case "d":
		return bochaFreshnessOneDay
	case "w":
		return bochaFreshnessOneWeek
	case "m":
		return bochaFreshnessOneMonth
	case "y":
		return bochaFreshnessOneYear
	default:
		return ""
	}
}

// SearchConfig 搜索配置
type SearchConfig struct {
	GoogleAPIKey   string `mapstructure:"google_api_key"`
	TavilyAPIKey   string `mapstructure:"tavily_api_key"`
	BochaAPIKey    string `mapstructure:"bocha_api_key"`
	SearchEngineID string `mapstructure:"search_engine_id"`
	Provider       string `mapstructure:"provider"` // "tavily" | "bocha" | "google"
	HTTPTimeout    int    `mapstructure:"http_timeout"`
}

// NewSearchEngine 根据配置创建搜索引擎客户端
func NewSearchEngine(cfg *SearchConfig) SearchEngine {
	timeout := time.Duration(cfg.HTTPTimeout) * time.Second
	switch cfg.Provider {
	case "bocha":
		return NewBochaSearchClientWithTimeout(cfg.BochaAPIKey, timeout)
	case "google":
		return NewGoogleSearchClientWithTimeout(cfg.GoogleAPIKey, cfg.SearchEngineID, timeout)
	default:
		return NewTavilySearchClientWithTimeout(cfg.TavilyAPIKey, timeout)
	}
}
