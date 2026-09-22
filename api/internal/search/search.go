package search

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bytedance/sonic"

	"github.com/Huang131/go-manus/api/config"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/pkg/httpconst"
	"github.com/Huang131/go-manus/api/pkg/logger"
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

// Tavily 面向 LLM agent 设计，返回已提取的正文内容而非裸链接。
type TavilySearchClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// 创建带请求超时的 Tavily 客户端。
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
		return model.NewToolError(err.Error()), err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	req.Header.Set("Authorization", httpconst.AuthBearerPrefix+c.apiKey)
	req.Header.Set("Content-Type", httpconst.ContentTypeJSON)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBodyBytes))
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	if resp.StatusCode != http.StatusOK {
		return model.NewToolError(fmt.Sprintf("Tavily API error: status=%d, body=%s", resp.StatusCode, string(respBody))), nil
	}

	var tavilyResp tavilySearchResponse
	if err := sonic.Unmarshal(respBody, &tavilyResp); err != nil {
		return model.NewToolError(err.Error()), err
	}

	results := &model.SearchResults{
		Query:          query,
		TotalResults:   len(tavilyResp.Results),
		ResponseTime:   tavilyResp.ResponseTime,
		TotalEstimated: 0, // Tavily 不返回总数估计
		Results:        make([]model.SearchResultItem, 0, len(tavilyResp.Results)),
	}
	for _, item := range tavilyResp.Results {
		results.Results = append(results.Results, model.SearchResultItem{
			Title:   item.Title,
			URL:     item.URL,
			Snippet: item.Content,
			Score:   item.Score,
		})
	}

	return model.NewToolResultWithMessage("", map[string]interface{}{
		"results": results,
	}), nil
}

type tavilySearchRequest struct {
	Query       string `json:"query"`
	SearchDepth string `json:"search_depth"` // 搜索深度
	MaxResults  int    `json:"max_results"`
	Topic       string `json:"topic"`
	TimeRange   string `json:"time_range,omitempty"`
}

type tavilySearchResponse struct {
	Results []struct {
		Title   string  `json:"title"`
		URL     string  `json:"url"`
		Content string  `json:"content"`
		Score   float64 `json:"score"`
	} `json:"results"`
	ResponseTime float64 `json:"response_time"`
}

// 国内直连、中文场景最优，作为 Tavily 直连不稳时的付费兜底。
type BochaSearchClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// 创建带请求超时的博查客户端。
func NewBochaSearchClientWithTimeout(apiKey string, timeout time.Duration) *BochaSearchClient {
	timeout = resolveSearchTimeout(timeout)
	return &BochaSearchClient{
		apiKey:  apiKey,
		baseURL: "https://api.bocha.cn/v1/web-search",
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
		Summary:   true, // 开启 AI 摘要
	})
	if err != nil {
		return model.NewToolError(err.Error()), err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	req.Header.Set("Authorization", httpconst.AuthBearerPrefix+c.apiKey)
	req.Header.Set("Content-Type", httpconst.ContentTypeJSON)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBodyBytes))
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	if resp.StatusCode != http.StatusOK {
		return model.NewToolError(fmt.Sprintf("Bocha API error: status=%d, body=%s", resp.StatusCode, string(respBody))), nil
	}

	var bochaResp bochaSearchResponse
	if err := sonic.Unmarshal(respBody, &bochaResp); err != nil {
		return model.NewToolError(err.Error()), err
	}
	if bochaResp.Code != 200 {
		return model.NewToolError(fmt.Sprintf("Bocha API error: code=%d, msg=%s", bochaResp.Code, bochaResp.Msg)), nil
	}

	values := bochaResp.Data.WebPages.Value
	results := &model.SearchResults{
		Query:          query,
		TotalResults:   len(values),
		TotalEstimated: bochaResp.Data.WebPages.TotalEstimatedMatches,
		ResponseTime:   0, // Bocha 不返回响应时间
		Results:        make([]model.SearchResultItem, 0, len(values)),
	}
	for _, item := range values {
		results.Results = append(results.Results, model.SearchResultItem{
			Title:         item.Name,
			URL:           item.URL,
			Snippet:       item.Snippet,
			Summary:       item.Summary,
			SiteName:      item.SiteName,
			SiteIcon:      item.SiteIcon,
			DatePublished: item.DatePublished,
		})
	}

	return model.NewToolResultWithMessage("", map[string]interface{}{
		"results": results,
	}), nil
}

// 博查搜索请求体。freshness 为空时服务端按 noLimit 处理。
type bochaSearchRequest struct {
	Query     string `json:"query"`
	Count     int    `json:"count"`
	Freshness string `json:"freshness,omitempty"`
	Summary   bool   `json:"summary,omitempty"`
}

// 博查搜索响应体。
type bochaSearchResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		WebPages struct {
			TotalEstimatedMatches int64 `json:"totalEstimatedMatches"`
			Value                 []struct {
				Name          string `json:"name"`
				URL           string `json:"url"`
				Snippet       string `json:"snippet"`
				Summary       string `json:"summary"`
				SiteName      string `json:"siteName"`
				SiteIcon      string `json:"siteIcon"`
				DatePublished string `json:"datePublished"`
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

// FallbackSearchClient Tavily + Bocha 自动切换客户端。
// 当主客户端（Tavily）调用失败时，自动切换到备选客户端（Bocha）。
type FallbackSearchClient struct {
	primary  *TavilySearchClient
	fallback *BochaSearchClient
}

// NewFallbackSearchClient 创建支持自动 fallback 的搜索客户端。
func NewFallbackSearchClient(tavilyKey, bochaKey string, timeout time.Duration) *FallbackSearchClient {
	return &FallbackSearchClient{
		primary:  NewTavilySearchClientWithTimeout(tavilyKey, timeout),
		fallback: NewBochaSearchClientWithTimeout(bochaKey, timeout),
	}
}

// Invoke 先调用 Tavily，失败后自动切换到 Bocha。
func (c *FallbackSearchClient) Invoke(ctx context.Context, query string, dateRange *string, limit int) (*model.ToolResult, error) {
	// 尝试主客户端（Tavily）
	result, err := c.primary.Invoke(ctx, query, dateRange, limit)
	if err == nil && result.Success {
		return result, nil
	}

	// 主客户端失败，记录日志并切换到 fallback（Bocha）
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	} else {
		errMsg = result.Message
	}
	logger.Warn("Tavily search failed, falling back to Bocha",
		logger.String("error", errMsg),
		logger.String("query", query))

	fallbackResult, fallbackErr := c.fallback.Invoke(ctx, query, dateRange, limit)
	if fallbackErr == nil && fallbackResult.Success {
		return fallbackResult, nil
	}

	// 两个都失败，返回 fallback 的错误结果和错误信息
	var fallbackErrMsg string
	if fallbackErr != nil {
		fallbackErrMsg = fallbackErr.Error()
	} else {
		fallbackErrMsg = fallbackResult.Message
	}
	logger.Error("Both search engines failed",
		logger.String("primary_error", errMsg),
		logger.String("fallback_error", fallbackErrMsg),
		logger.String("query", query))

	return fallbackResult, fmt.Errorf("search failed: primary Tavily failed (%s), fallback Bocha also failed (%s)", errMsg, fallbackErrMsg)
}

// NewSearchEngine 根据配置创建搜索引擎客户端
func NewSearchEngine(cfg *config.SearchConfig) SearchEngine {
	timeout := time.Duration(cfg.HTTPTimeout) * time.Second
	switch cfg.Provider {
	case "bocha":
		return NewBochaSearchClientWithTimeout(cfg.BochaAPIKey, timeout)
	case "fallback":
		return NewFallbackSearchClient(cfg.TavilyAPIKey, cfg.BochaAPIKey, timeout)
	default:
		if cfg.Provider != "" {
			logger.Warn("unknown search provider configured, falling back to tavily",
				logger.String("provider", cfg.Provider))
		}
		return NewTavilySearchClientWithTimeout(cfg.TavilyAPIKey, timeout)
	}
}
