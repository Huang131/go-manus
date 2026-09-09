package handler

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Huang131/go-manus/api/internal/service"
	"github.com/Huang131/go-manus/api/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// vncUpgrader 接受浏览器 noVNC 客户端的 WebSocket 升级请求。
// VNC 连接由 nginx / 内网鉴权层控制，这里允许任意 Origin。
var vncUpgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	Subprotocols:    []string{"binary", "base64"},
	ReadBufferSize:  32 * 1024,
	WriteBufferSize: 32 * 1024,
}

// VNCProxy 将前端的 WebSocket 连接代理到 sandbox 的 websockify 端点。
// 这里使用标准的 WS -> WS 双向转发，而不是把 HTTP body 当成可写通道。
func VNCProxy(svc service.SessionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")

		vncURL, err := svc.GetVNCURL(c.Request.Context(), sessionID)
		if err != nil {
			logger.Warn("VNC URL 解析失败", logger.String("session_id", sessionID), logger.Err(err))
			c.String(http.StatusBadGateway, "vnc url resolve failed: %s", err.Error())
			return
		}

		clientConn, err := vncUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.Warn("VNC WS 升级失败", logger.Err(err))
			return
		}
		defer clientConn.Close()

		serverConn, err := dialSandboxVNC(c.Request.Context(), vncURL, clientConn.Subprotocol())
		if err != nil {
			logger.Warn("VNC 连接 sandbox 失败",
				logger.String("vnc_url", vncURL),
				logger.Err(err))
			_ = clientConn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error()))
			return
		}
		defer serverConn.Close()

		errCh := make(chan error, 2)

		go func() {
			for {
				msgType, data, err := clientConn.ReadMessage()
				if err != nil {
					errCh <- err
					return
				}
				if err := serverConn.WriteMessage(msgType, data); err != nil {
					errCh <- err
					return
				}
			}
		}()

		go func() {
			for {
				msgType, data, err := serverConn.ReadMessage()
				if err != nil {
					errCh <- err
					return
				}
				if err := clientConn.WriteMessage(msgType, data); err != nil {
					errCh <- err
					return
				}
			}
		}()

		<-errCh
	}
}

// dialSandboxVNC 直接连接 sandbox 的 websockify WebSocket 地址。
func dialSandboxVNC(ctx context.Context, vncURL, subprotocol string) (*websocket.Conn, error) {
	u, err := url.Parse(vncURL)
	if err != nil {
		return nil, err
	}

	dialer := websocket.Dialer{
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  10 * time.Second,
		EnableCompression: false,
	}

	header := http.Header{}
	if strings.TrimSpace(subprotocol) != "" {
		dialer.Subprotocols = []string{subprotocol}
		header.Set("Sec-WebSocket-Protocol", subprotocol)
	}

	conn, _, err := dialer.DialContext(ctx, u.String(), header)
	return conn, err
}
