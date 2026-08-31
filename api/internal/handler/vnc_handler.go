package handler

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/mooc-manus/go-manus/api/internal/service"
	"github.com/mooc-manus/go-manus/api/pkg/logger"
	"go.uber.org/zap"
)

// vncUpgrader 接受浏览器 noVNC 客户端的 WebSocket 升级请求。
// 允许任意 origin（与 mooc-manus 实现对齐：vnc 是内网工具，由 nginx 控制外层鉴权）。
var vncUpgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	Subprotocols:    []string{"binary", "base64"},
	ReadBufferSize:  32 * 1024,
	WriteBufferSize: 32 * 1024,
}

// vncHTTPClient 用 HTTP/1.1 长连接把客户端的 WS 帧透传到 sandbox 的 websockify。
// gorilla/websocket 默认会注入 Connection: upgrade 头，但 websockify（Python）期望原始 HTTP/1.1 长连接，
// 因此这里使用标准 http.Client 直接发 GET 并复用连接。
func newVNCHTTPClient() *http.Client {
	transport := &http.Transport{
		DisableCompression:    true,
		MaxIdleConns:          10,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 0, // 0 表示无超时，websockify 长连接需要
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{Transport: transport}
}

// VNCProxy 转发一个 WebSocket 连接到 sandbox 的 VNC（websockify）端点。
// 对齐 mooc-manus 的 session_routes.py:285-360 实现。
func VNCProxy(svc service.SessionService, client *http.Client) gin.HandlerFunc {
	if client == nil {
		client = newVNCHTTPClient()
	}
	return func(c *gin.Context) {
		sessionID := c.Param("id")

		vncURL, err := svc.GetVNCURL(c.Request.Context(), sessionID)
		if err != nil {
			logger.Warn("VNC URL 解析失败", zap.String("session_id", sessionID), zap.Error(err))
			c.String(http.StatusBadGateway, "vnc url resolve failed: %s", err.Error())
			return
		}

		// 1. 升级浏览器连接为 WS
		wsConn, err := vncUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.Warn("VNC WS 升级失败", zap.Error(err))
			return
		}
		defer wsConn.Close()

		// 2. 与 sandbox 建立 HTTP/1.1 长连接（websockify 协议）
		httpResp, err := dialSandboxVNC(client, vncURL, wsConn.Subprotocol())
		if err != nil {
			logger.Warn("VNC 连接 sandbox 失败",
				zap.String("vnc_url", vncURL),
				zap.Error(err))
			_ = wsConn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error()))
			return
		}
		defer httpResp.Body.Close()

		// 3. 双向转发：客户端 WS ↔ sandbox HTTP body
		errCh := make(chan error, 2)

		// 客户端 → sandbox
		go func() {
			for {
				msgType, data, err := wsConn.ReadMessage()
				if err != nil {
					errCh <- err
					return
				}
				// 透传：根据子协议写入二进制或 base64 文本
				switch msgType {
				case websocket.BinaryMessage:
					if _, err := httpResp.Body.(io.Writer).Write(data); err != nil {
						// 不可写，转向 WriteTo hack：直接发 raw TCP — 改用 ReadFrom
						errCh <- err
						return
					}
				case websocket.TextMessage:
					decoded, derr := decodeSubprotocolFrame(wsConn.Subprotocol(), data)
					if derr != nil {
						errCh <- derr
						return
					}
					if _, err := httpResp.Body.(io.Writer).Write(decoded); err != nil {
						errCh <- err
						return
					}
				}
			}
		}()

		// sandbox → 客户端
		go func() {
			buf := make([]byte, 32*1024)
			for {
				n, err := httpResp.Body.Read(buf)
				if n > 0 {
					if werr := writeVNCFrame(wsConn, wsConn.Subprotocol(), buf[:n]); werr != nil {
						errCh <- werr
						return
					}
				}
				if err != nil {
					errCh <- err
					return
				}
			}
		}()

		<-errCh
	}
}

// dialSandboxVNC 与 sandbox 建立 websockify 长连接。
// 使用裸 net/http.Client GET，禁用 chunked / expect-continue 行为以兼容 websockify。
func dialSandboxVNC(client *http.Client, vncURL, subprotocol string) (*http.Response, error) {
	// vncURL 形如 ws://host:5901，但这里用 HTTP 协议访问相同 host:port
	u, err := url.Parse(vncURL)
	if err != nil {
		return nil, err
	}
	httpScheme := "http"
	if u.Scheme == "wss" {
		httpScheme = "https"
	}
	target := httpScheme + "://" + u.Host + "/"

	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	// 关键：Upgrade 头让 websockify 进入二进制透传模式
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Version", "13")
	if subprotocol != "" {
		req.Header.Set("Sec-WebSocket-Protocol", subprotocol)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, errors.New("sandbox 升级失败: status=" + resp.Status + " body=" + string(body))
	}
	return resp, nil
}

// decodeSubprotocolFrame 兼容 websockify 的 binary/base64 子协议。
// noVNC 默认使用 binary，但部分老旧实现使用 base64 文本帧。
func decodeSubprotocolFrame(subprotocol string, data []byte) ([]byte, error) {
	if strings.EqualFold(subprotocol, "base64") {
		// 文本帧已经是 base64 编码，websockify 期望原始字节 — 这里走客户端二进制路径更简单，
		// 因此 base64 模式下我们拒绝处理，提示前端改用 binary。
		return nil, errors.New("base64 subprotocol not supported, use binary")
	}
	return data, nil
}

// writeVNCFrame 根据子协议选择 binary/text 写入 WS 客户端。
func writeVNCFrame(ws *websocket.Conn, subprotocol string, data []byte) error {
	if strings.EqualFold(subprotocol, "base64") {
		return ws.WriteMessage(websocket.TextMessage, data)
	}
	return ws.WriteMessage(websocket.BinaryMessage, data)
}
