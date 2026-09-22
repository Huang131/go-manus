//go:build external

package integration

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/config"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/sandbox"
)

// 沙箱是独立部署的 Python 服务，按 docs/API_TEST_RULES.md 属于 external smoke：
// 它验证 go-manus 客户端与部署环境之间的协议契约，不进入 go test ./... 门禁。
//
// 运行前先启动根目录 Compose 里的 sandbox（宿主机 8090 → 容器 8080）：
//
//	docker compose -f ../docker-compose.yml up -d sandbox
//	cd api && make test-sandbox
const (
	sandboxAddress     = "http://127.0.0.1:8090"
	sandboxHTTPTimeout = 120 // 浏览器操作最慢（Navigate 90s），需大于 Python 侧超时
)

// requireSandbox 建立真实沙箱客户端。服务不可达时明确 skip，
// 但请求一旦发出，协议错误必须让测试失败，不能降级为通过。
func requireSandbox(t *testing.T) *sandbox.SandboxClient {
	t.Helper()
	client := sandbox.NewSandboxClient(&config.SandboxConfig{
		Address:     sandboxAddress,
		HTTPTimeout: sandboxHTTPTimeout,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.HealthCheck(ctx); err != nil {
		t.Skipf("沙箱服务不可达（%s），跳过 external smoke：%v", sandboxAddress, err)
	}
	return client
}

func sandboxContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// sandboxData 取出信封 data。沙箱失败走 error 返回，这里拦的是"业务失败被当成成功"。
func sandboxData(t *testing.T, res *model.ToolResult) map[string]any {
	t.Helper()
	if res == nil {
		t.Fatal("工具结果为 nil")
	}
	if !res.Success {
		t.Fatalf("工具结果标记失败：%s", res.Message)
	}
	data, ok := res.Data.(map[string]any)
	if !ok {
		t.Fatalf("data 期望 map[string]any，实际 %T", res.Data)
	}
	return data
}

// uniquePath 生成每次运行唯一的沙箱路径，避免重复运行或并发时相互影响。
func uniquePath(suffix string) string {
	return fmt.Sprintf("/tmp/go-manus-contract-%d%s", time.Now().UnixNano(), suffix)
}

// maxTruncatedMarkerRunes 是服务端可能追加的截断提示长度上限。
// 早期沙箱版本会把 "(truncated)" 拼进 content，当前源码改为只走 msg 提示。
// 客户端契约只保证有效内容被限制在 max_length 内，因此这里既锁住"截断生效"，
// 又不把服务端提示策略的变化误判成客户端回归。
const maxTruncatedMarkerRunes = 16

// assertContentLimitedTo 断言 content 的前 maxLength 个字符就是文件内容，
// 且总长度不超过 maxLength 加少量提示。截断失效时会返回完整文件长度，在这里暴露。
func assertContentLimitedTo(t *testing.T, content, filler string, maxLength int) {
	t.Helper()
	if !strings.HasPrefix(content, strings.Repeat(filler, maxLength)) {
		t.Fatalf("content 未按 max_length=%d 截断，实际 %d 字符：%q",
			maxLength, len([]rune(content)), content)
	}
	if n := len([]rune(content)); n > maxLength+maxTruncatedMarkerRunes {
		t.Fatalf("content 字符数 = %d，超出 max_length+提示上限 %d", n, maxLength+maxTruncatedMarkerRunes)
	}
}

// TestSandbox_HealthEndpointIsReachable 验证部署连通性：
// API 启动期的健康检查契约就是 GET /health 返回 200。
func TestSandbox_HealthEndpointIsReachable(t *testing.T) {
	client := requireSandbox(t)
	if err := client.HealthCheck(sandboxContext(t)); err != nil {
		t.Fatalf("HealthCheck() error = %v", err)
	}
}

// TestSandbox_ShellExecReturnsCommandOutput 验证 shell 执行契约：
// session_id 为空时由沙箱分配会话；命令在同步等待窗口内返回 output，
// 若沙箱先返回 running，则必须能通过 read-shell-output 读到同一份输出。
func TestSandbox_ShellExecReturnsCommandOutput(t *testing.T) {
	client := requireSandbox(t)
	ctx := sandboxContext(t)

	nonce := fmt.Sprintf("go-manus-%d", time.Now().UnixNano())
	res, err := client.ExecCommand(ctx, "", "/tmp", "echo "+nonce)
	if err != nil {
		t.Fatalf("ExecCommand() error = %v", err)
	}
	data := sandboxData(t, res)

	sessionID, _ := data["session_id"].(string)
	if sessionID == "" {
		t.Fatalf("exec-command 未返回 session_id：%#v", data)
	}
	t.Cleanup(func() {
		// 清理沙箱内的交互式会话，避免容器累积残留 shell。
		_, _ = client.KillProcess(context.Background(), sessionID)
	})

	if output, _ := data["output"].(string); strings.Contains(output, nonce) {
		return
	}
	if got := waitShellOutput(t, client, sessionID, nonce); !strings.Contains(got, nonce) {
		t.Fatalf("shell 输出未包含命令标记 %q：%q", nonce, got)
	}
}

// waitShellOutput 轮询 read-shell-output 直到出现期望内容。
// 外部服务没有事件通知，只能按 ticker 拉取；超时由断言给出，不用固定 Sleep 掩盖状态。
func waitShellOutput(t *testing.T, client *sandbox.SandboxClient, sessionID, want string) string {
	t.Helper()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(30 * time.Second)

	last := ""
	for {
		select {
		case <-timeout:
			t.Fatalf("等待 shell 输出超时，最后输出 %q", last)
		case <-ticker.C:
		}

		res, err := client.ReadShellOutput(context.Background(), sessionID, false)
		if err != nil {
			t.Fatalf("ReadShellOutput() error = %v", err)
		}
		last, _ = sandboxData(t, res)["output"].(string)
		if strings.Contains(last, want) {
			return last
		}
	}
}

// TestSandbox_FileLifecycle 验证文件端点契约：写入内容必须与读回内容一致，
// 且 check-file-exists/find-files/delete-file 对同一路径的判断前后自洽。
func TestSandbox_FileLifecycle(t *testing.T) {
	client := requireSandbox(t)
	ctx := sandboxContext(t)

	remote := uniquePath(".txt")
	content := "alpha\nbeta"
	if _, err := client.WriteFile(ctx, remote, content, false, false, false, false); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Cleanup(func() {
		_, _ = client.DeleteFile(context.Background(), remote)
	})

	readRes, err := client.ReadFile(ctx, remote, nil, nil, false, 0)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	readData := sandboxData(t, readRes)
	if got, _ := readData["content"].(string); got != content {
		t.Fatalf("read-file content = %q, want %q", got, content)
	}
	if truncated, _ := readData["truncated"].(bool); truncated {
		t.Error("小文件被标记为截断")
	}

	existsRes, err := client.CheckFileExists(ctx, remote)
	if err != nil {
		t.Fatalf("CheckFileExists() error = %v", err)
	}
	if exists, _ := sandboxData(t, existsRes)["exists"].(bool); !exists {
		t.Fatalf("写入后 check-file-exists = false，路径 %s", remote)
	}

	findRes, err := client.FindFiles(ctx, "/tmp", "go-manus-contract-*.txt")
	if err != nil {
		t.Fatalf("FindFiles() error = %v", err)
	}
	files, _ := sandboxData(t, findRes)["files"].([]any)
	if !containsPath(files, remote) {
		t.Fatalf("find-files 未返回 %s，实际 files=%v", remote, files)
	}

	deleteRes, err := client.DeleteFile(ctx, remote)
	if err != nil {
		t.Fatalf("DeleteFile() error = %v", err)
	}
	if deleted, _ := sandboxData(t, deleteRes)["deleted"].(bool); !deleted {
		t.Error("delete-file deleted = false")
	}

	goneRes, err := client.CheckFileExists(ctx, remote)
	if err != nil {
		t.Fatalf("CheckFileExists() after delete error = %v", err)
	}
	if exists, _ := sandboxData(t, goneRes)["exists"].(bool); exists {
		t.Errorf("删除后 check-file-exists = true，路径 %s", remote)
	}
}

func containsPath(files []any, want string) bool {
	for _, file := range files {
		if path, ok := file.(string); ok && path == want {
			return true
		}
	}
	return false
}

// TestSandbox_ReadFileHonorsExplicitMaxLength 验证截断契约：
// max_length 按字符生效，同时必须通过 truncated 与 msg 双通道告诉调用方内容不完整。
func TestSandbox_ReadFileHonorsExplicitMaxLength(t *testing.T) {
	client := requireSandbox(t)
	ctx := sandboxContext(t)

	remote := uniquePath("-long.txt")
	if _, err := client.WriteFile(ctx, remote, strings.Repeat("a", 300), false, false, false, false); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Cleanup(func() {
		_, _ = client.DeleteFile(context.Background(), remote)
	})

	res, err := client.ReadFile(ctx, remote, nil, nil, false, 50)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	data := sandboxData(t, res)

	got, _ := data["content"].(string)
	assertContentLimitedTo(t, got, "a", 50)
	if truncated, _ := data["truncated"].(bool); !truncated {
		t.Error("truncated = false, want true")
	}
	// 截断提示走 msg 还是 content 由服务端版本决定，客户端只负责透传，不锁具体文案。
	if strings.TrimSpace(res.Message) == "" {
		t.Error("Message 为空，服务端提示未透传到 ToolResult")
	}
}

// TestSandbox_ReadFileUsesDefaultLimitWhenMaxLengthUnset 验证两端默认值契约：
// Go 客户端在 maxLength<=0 时不发送 max_length，沙箱侧必须回落到 10000 字符默认上限。
// 这一条锁住的是"客户端省略字段"与"服务端默认值"的一致性，任一侧改动都会在这里暴露。
func TestSandbox_ReadFileUsesDefaultLimitWhenMaxLengthUnset(t *testing.T) {
	client := requireSandbox(t)
	ctx := sandboxContext(t)

	remote := uniquePath("-default.txt")
	if _, err := client.WriteFile(ctx, remote, strings.Repeat("b", 12000), false, false, false, false); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Cleanup(func() {
		_, _ = client.DeleteFile(context.Background(), remote)
	})

	res, err := client.ReadFile(ctx, remote, nil, nil, false, 0)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	data := sandboxData(t, res)

	got, _ := data["content"].(string)
	assertContentLimitedTo(t, got, "b", 10000)
	if truncated, _ := data["truncated"].(bool); !truncated {
		t.Error("truncated = false, want true")
	}
}

// TestSandbox_BusinessErrorMapsToAPIError 验证失败映射契约：
// 沙箱异常以真实 HTTP 状态码 + {code,msg} 信封返回，客户端必须还原成 SandboxAPIError，
// 并且 SandboxErrorStatus 能供 handler 直接映射响应码。
// 这条契约跨了 Python 状态码与 Go 解析两端，mock 测试无法覆盖。
func TestSandbox_BusinessErrorMapsToAPIError(t *testing.T) {
	client := requireSandbox(t)
	ctx := sandboxContext(t)

	_, err := client.ReadFile(ctx, uniquePath("-missing.txt"), nil, nil, false, 0)
	if err == nil {
		t.Fatal("读取不存在的文件应返回错误")
	}

	var apiErr *sandbox.SandboxAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *sandbox.SandboxAPIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
	if strings.TrimSpace(apiErr.Message) == "" {
		t.Error("SandboxAPIError.Message 为空，信封 msg 未传递")
	}
	if got := sandbox.SandboxErrorStatus(err); got != http.StatusNotFound {
		t.Errorf("SandboxErrorStatus() = %d, want 404", got)
	}
}

// TestSandbox_BrowserScreenshotReturnsPNG 验证二进制通道契约：
// 截图先由沙箱落盘，再由客户端 GET download-file 取回原始字节。
// 断言 PNG 魔数是为了确认拿到的是真实图片，而不是被截断的错误信封。
func TestSandbox_BrowserScreenshotReturnsPNG(t *testing.T) {
	client := requireSandbox(t)
	ctx := sandboxContext(t)
	browser := sandbox.NewBrowserClient(client)

	if _, err := browser.Navigate(ctx, "about:blank"); err != nil {
		// browser 路由缺失（镜像未重建）或沙箱侧浏览器环境不可用（Chrome/CDP 起不来）时，
		// 沙箱分别返回 404 与 503/504。这属于部署环境问题而非 Go 客户端协议错误，
		// 按 external smoke 的边界跳过并给出诊断命令，不作为门禁失败。
		var apiErr *sandbox.SandboxAPIError
		if errors.As(err, &apiErr) && (apiErr.StatusCode == http.StatusNotFound || apiErr.StatusCode == 503 || apiErr.StatusCode == 504) {
			t.Skipf("沙箱 browser 能力不可用（%s），请检查镜像是否已重建、"+
				"Chrome/CDP 是否正常：docker exec go-manus-sandbox supervisorctl status（%v）",
				http.StatusText(apiErr.StatusCode), err)
		}
		t.Fatalf("Navigate() error = %v", err)
	}
	png, err := browser.Screenshot(ctx, false)
	if err != nil {
		t.Fatalf("Screenshot() error = %v", err)
	}

	pngMagic := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	if len(png) < len(pngMagic) || !bytes.HasPrefix(png, pngMagic) {
		t.Fatalf("截图不是 PNG：长度 %d，前缀 %x", len(png), png[:min(8, len(png))])
	}
}
