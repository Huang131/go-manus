package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/bytedance/sonic"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/pkg/httpconst"
)

// envelope 沙箱统一响应信封 {code,msg,data}。
// data 保持原始 JSON，由各端点决定解析成什么结构。
type envelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// requireAddress 校验沙箱地址，避免把空地址拼成非法 URL 后报出难懂的传输错误
func (c *SandboxClient) requireAddress() error {
	if c.address == "" {
		return fmt.Errorf("sandbox address not configured")
	}
	return nil
}

// endpoint 拼接沙箱端点地址
func (c *SandboxClient) endpoint(action string) string {
	return fmt.Sprintf("%s/api/%s", c.address, action)
}

// postJSON 发送 JSON 请求并把响应 data 解码到 out，返回沙箱提示消息。
//
// 这是所有 JSON 端点的唯一出口：状态码判断、信封解码、错误构造都收敛在这里，
// 各业务方法只负责"组装请求 + 决定 data 解析成什么"。
func (c *SandboxClient) postJSON(ctx context.Context, action string, body any, out any) (string, error) {
	if err := c.requireAddress(); err != nil {
		return "", err
	}

	payload, err := sonic.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal %s request: %w", action, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(action), bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create %s request: %w", action, err)
	}
	httpReq.Header.Set("Content-Type", httpconst.ContentTypeJSON)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("send %s request: %w", action, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read %s response: %w", action, err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", newAPIError(resp.StatusCode, raw)
	}

	var env envelope
	if err := sonic.Unmarshal(raw, &env); err != nil {
		return "", fmt.Errorf("unmarshal %s response: %w", action, err)
	}
	// 沙箱以 HTTP 200 承载业务失败的情况（code>=300）同样按错误处理
	if env.Code >= http.StatusMultipleChoices {
		return "", &SandboxAPIError{StatusCode: resp.StatusCode, Code: env.Code, Message: env.Msg}
	}
	if err := decodeData(env.Data, out); err != nil {
		return "", fmt.Errorf("decode %s data: %w", action, err)
	}
	return env.Msg, nil
}

// postTool 调用 JSON 端点并把 data 原样装进 ToolResult，供结果结构不固定的端点复用
func (c *SandboxClient) postTool(ctx context.Context, action string, body any) (*model.ToolResult, error) {
	var data map[string]interface{}
	msg, err := c.postJSON(ctx, action, body, &data)
	if err != nil {
		return toolResultErr(err)
	}
	return model.NewToolResultWithMessage(msg, data), nil
}

// downloadBytes 下载沙箱文件原始字节（非 JSON 信封）。
//
// 产物、截图等二进制内容必须走这条通道：JSON 信封需要 base64 编码，
// 会把体积放大 33% 且容易在中间环节被截断。
func (c *SandboxClient) downloadBytes(ctx context.Context, filepath string) ([]byte, error) {
	if err := c.requireAddress(); err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/api/file/download-file?filepath=%s", c.address, url.QueryEscape(filepath))
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", filepath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return nil, newAPIError(resp.StatusCode, raw)
	}

	// 多读 1 字节用于判断是否超限，避免把超大文件整个读进内存
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxDownloadBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read download body: %w", err)
	}
	if len(data) > maxDownloadBytes {
		return nil, fmt.Errorf("downloaded %s exceeds %d bytes limit", filepath, maxDownloadBytes)
	}
	return data, nil
}

// uploadFile 以 multipart 表单上传文件到沙箱
func (c *SandboxClient) uploadFile(ctx context.Context, filepath, filename string, data []byte) (*model.ToolResult, error) {
	if err := c.requireAddress(); err != nil {
		return toolResultErr(err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return toolResultErr(err)
	}
	if _, err := part.Write(data); err != nil {
		return toolResultErr(err)
	}
	if err := writer.WriteField("filepath", filepath); err != nil {
		return toolResultErr(err)
	}
	if err := writer.Close(); err != nil {
		return toolResultErr(err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint("file/upload-file"), body)
	if err != nil {
		return toolResultErr(err)
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return toolResultErr(err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return toolResultErr(err)
	}
	if resp.StatusCode != http.StatusOK {
		return toolResultErr(newAPIError(resp.StatusCode, raw))
	}

	var env envelope
	if err := sonic.Unmarshal(raw, &env); err != nil {
		return toolResultErr(err)
	}
	if env.Code >= http.StatusMultipleChoices {
		return toolResultErr(&SandboxAPIError{StatusCode: resp.StatusCode, Code: env.Code, Message: env.Msg})
	}
	return model.NewToolResultWithMessage(env.Msg, decodeMapQuietly(env.Data)), nil
}

// decodeData 把信封里的 data 解码到目标结构；目标为空或 data 缺失时跳过
func decodeData(raw json.RawMessage, out any) error {
	if out == nil || len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return sonic.Unmarshal(raw, out)
}

// decodeMapQuietly 尽力解码为 map，失败时返回 nil 而不打断调用链
func decodeMapQuietly(raw json.RawMessage) map[string]interface{} {
	var data map[string]interface{}
	if err := decodeData(raw, &data); err != nil {
		return nil
	}
	return data
}

// newAPIError 由非 200 响应构造统一错误：优先取信封里的 msg 与 code
func newAPIError(statusCode int, raw []byte) error {
	message := ""
	code := statusCode
	var env envelope
	if err := sonic.Unmarshal(raw, &env); err == nil && env.Msg != "" {
		message = env.Msg
		if env.Code != 0 {
			code = env.Code
		}
	} else {
		message = string(raw)
	}
	return &SandboxAPIError{StatusCode: statusCode, Code: code, Message: message}
}

// SandboxErrorStatus 从错误中提取 HTTP 状态码，便于 handler 映射响应
func SandboxErrorStatus(err error) int {
	var apiErr *SandboxAPIError
	if errors.As(err, &apiErr) && apiErr != nil {
		if apiErr.StatusCode != 0 {
			return apiErr.StatusCode
		}
		return apiErr.Code
	}
	return 0
}
