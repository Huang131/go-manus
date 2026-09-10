package external

import (
	"context"
	"fmt"
	"github.com/bytedance/sonic"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/Huang131/go-manus/api/internal/model"
)

// SearchEngine 搜索引擎接口
type SearchEngine interface {
	// Invoke 调用搜索引擎
	Invoke(ctx context.Context, query string, dateRange *string) (*model.ToolResult, error)
}

// BingSearchClient Bing 搜索客户端
type BingSearchClient struct {
	apiKey     string
	httpClient *http.Client
}

// NewBingSearchClient 创建 Bing 搜索客户端
func NewBingSearchClient(apiKey string) *BingSearchClient {
	return NewBingSearchClientWithTimeout(apiKey, 30*time.Second)
}

// NewBingSearchClientWithTimeout 创建带请求超时的 Bing 客户端。
func NewBingSearchClientWithTimeout(apiKey string, timeout time.Duration) *BingSearchClient {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &BingSearchClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Invoke 调用 Bing 搜索
func (c *BingSearchClient) Invoke(ctx context.Context, query string, dateRange *string) (*model.ToolResult, error) {
	if c.apiKey == "" {
		return model.NewToolError("Bing API key not configured"), nil
	}

	// 构建请求 URL
	baseURL := "https://api.bing.microsoft.com/v7.0/search"
	params := url.Values{}
	params.Set("q", query)
	params.Set("mkt", "zh-CN")
	params.Set("safesearch", "Moderate")
	if dateRange != nil {
		params.Set("freshness", *dateRange)
	}

	reqURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return model.NewToolError(err.Error()), err
	}

	// 设置请求头
	req.Header.Set("Ocp-Apim-Subscription-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.NewToolError(err.Error()), err
	}

	if resp.StatusCode != http.StatusOK {
		return model.NewToolError(fmt.Sprintf("Bing API error: status=%d, body=%s", resp.StatusCode, string(body))), nil
	}

	// 解析响应
	var bingResp bingSearchResponse
	if err := sonic.Unmarshal(body, &bingResp); err != nil {
		return model.NewToolError(err.Error()), err
	}

	// 转换为标准搜索结果
	results := &model.SearchResults{
		Results: make([]model.SearchResultItem, 0, len(bingResp.WebPages.Value)),
	}

	for _, item := range bingResp.WebPages.Value {
		results.Results = append(results.Results, model.SearchResultItem{
			Title:   item.Name,
			URL:     item.URL,
			Snippet: item.Snippet,
		})
	}

	return model.NewToolResultWithMessage("", map[string]interface{}{
		"results":        results,
		"total_matches":  bingResp.WebPages.TotalMatches,
		"web_search_url": bingResp.WebPages.WebSearchURL,
	}), nil
}

// bingSearchResponse Bing 搜索响应
type bingSearchResponse struct {
	Type         string `json:"_type"`
	QueryContext struct {
		OriginalQuery string `json:"originalQuery"`
	} `json:"queryContext"`
	WebPages struct {
		WebSearchURL string `json:"webSearchUrl"`
		TotalMatches int    `json:"totalEstimatedMatches"`
		Value        []struct {
			ID              string `json:"id"`
			Name            string `json:"name"`
			URL             string `json:"url"`
			DisplayURL      string `json:"displayUrl"`
			Snippet         string `json:"snippet"`
			DateLastCrawled string `json:"dateLastCrawled"`
			Language        string `json:"language"`
			IsNavigational  bool   `json:"isNavigational"`
			Thumbnail       struct {
				SourceURL string `json:"sourceUrl"`
			} `json:"thumbnail"`
		} `json:"value"`
	} `json:"webPages"`
	Images struct {
		Value []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			URL    string `json:"url"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"value"`
	} `json:"images"`
	News struct {
		Value []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			URL           string `json:"url"`
			Description   string `json:"description"`
			DatePublished string `json:"datePublished"`
			Provider      []struct {
				Type string `json:"_type"`
				Name string `json:"name"`
			} `json:"provider"`
		} `json:"value"`
	} `json:"news"`
	RelatedSearches struct {
		Value []struct {
			ID    string `json:"id"`
			Value string `json:"value"`
		} `json:"value"`
	} `json:"relatedSearches"`
	RankingResponse struct {
		MainLine struct {
			Items []struct {
				AnswerType  string `json:"answerType"`
				ResultIndex int    `json:"resultIndex"`
			} `json:"items"`
		} `json:"mainLine"`
	} `json:"rankingResponse"`
}

// GoogleSearchClient Google 搜索客户端 (模拟实现，实际需要 Google API)
type GoogleSearchClient struct {
	apiKey     string
	httpClient *http.Client
}

// NewGoogleSearchClient 创建 Google 搜索客户端
func NewGoogleSearchClient(apiKey string) *GoogleSearchClient {
	return NewGoogleSearchClientWithTimeout(apiKey, 30*time.Second)
}

// NewGoogleSearchClientWithTimeout 创建带请求超时的 Google 客户端。
func NewGoogleSearchClientWithTimeout(apiKey string, timeout time.Duration) *GoogleSearchClient {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &GoogleSearchClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Invoke 调用 Google 搜索
func (c *GoogleSearchClient) Invoke(ctx context.Context, query string, dateRange *string) (*model.ToolResult, error) {
	if c.apiKey == "" {
		return model.NewToolError("Google API key not configured"), nil
	}

	// 构建请求 URL
	baseURL := "https://www.googleapis.com/customsearch/v1"
	params := url.Values{}
	params.Set("key", c.apiKey)
	params.Set("q", query)
	params.Set("cx", "") // 需要设置 Search Engine ID
	params.Set("hl", "zh-CN")
	if dateRange != nil {
		params.Set("dateRestrict", *dateRange)
	}

	reqURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return model.NewToolError(err.Error()), err
	}

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return model.NewToolError(err.Error()), err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.NewToolError(err.Error()), err
	}

	if resp.StatusCode != http.StatusOK {
		return model.NewToolError(fmt.Sprintf("Google API error: status=%d, body=%s", resp.StatusCode, string(body))), nil
	}

	// 解析响应
	var googleResp googleSearchResponse
	if err := sonic.Unmarshal(body, &googleResp); err != nil {
		return model.NewToolError(err.Error()), err
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

// SearchConfig 搜索配置
type SearchConfig struct {
	BingAPIKey     string `mapstructure:"bing_api_key"`
	GoogleAPIKey   string `mapstructure:"google_api_key"`
	SearchEngineID string `mapstructure:"search_engine_id"`
	Provider       string `mapstructure:"provider"` // "bing" or "google"
	HTTPTimeout    int    `mapstructure:"http_timeout"`
}

// NewSearchEngine 根据配置创建搜索引擎客户端
func NewSearchEngine(cfg *SearchConfig) SearchEngine {
	timeout := time.Duration(cfg.HTTPTimeout) * time.Second
	switch cfg.Provider {
	case "google":
		return NewGoogleSearchClientWithTimeout(cfg.GoogleAPIKey, timeout)
	default:
		return NewBingSearchClientWithTimeout(cfg.BingAPIKey, timeout)
	}
}
