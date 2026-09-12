# Manus 沙箱服务

基于 Ubuntu 22.04 构建的学习型沙箱环境，提供代码执行、浏览器自动化和远程桌面访问能力。

## 技术栈

- Ubuntu 22.04
- Python 3.10 + FastAPI
- Node.js 24 (LTS)
- Chromium (浏览器自动化)
- Xvfb + x11vnc + websockify (虚拟显示 + VNC)
- Supervisor (进程管理)

## 架构

沙箱通过 Supervisor 管理多个进程：

| 进程 | 端口 | 说明 |
|------|------|------|
| FastAPI | 8080 | REST API（文件操作、Shell 执行） |
| Chrome | 8222 (内部) | 浏览器实例 |
| socat | 9222 | Chrome DevTools Protocol 代理 |
| Xvfb | - | 虚拟显示器 (:1) |
| x11vnc | 5900 | VNC 服务 |
| websockify | 5901 | WebSocket VNC 代理 |

## API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/file/read-file` | 读取文件 |
| POST | `/api/file/write-file` | 写入文件 |
| POST | `/api/file/upload-file` | 上传文件 |
| GET | `/api/file/download-file` | 下载文件 |
| POST | `/api/shell/exec-command` | 执行命令 |
| POST | `/api/shell/read-shell-output` | 读取 Shell 输出 |
| GET | `/api/supervisor/status` | 获取进程状态 |

## 本地开发

推荐使用根目录 Compose 启动 sandbox。需要进入运行中的容器调试时：

```bash
docker compose up -d sandbox
docker exec -it go-manus-sandbox bash
```

本地直接运行时使用锁定依赖：

```bash
uv sync --frozen
uv run uvicorn app.main:app --host 0.0.0.0 --port 8080 --reload
```

## Docker 部署

沙箱服务通过根目录的 `docker-compose.yml` 统一部署。API 服务通过容器网络中的
`SANDBOX_ADDRESS=http://sandbox:8080` 连接。

### 端口说明

根目录 Compose 默认保留 `${SANDBOX_PORT:-8090}:8080` 宿主机映射，方便学习和调试；
生产环境建议删除 `ports` 映射，仅保留容器网络访问：

- `8080` - FastAPI REST API
- `9222` - Chrome DevTools Protocol
- `5900` - VNC RFB
- `5901` - WebSocket VNC（API 服务通过此端口代理 VNC 到前端）

### 信任模型与安全边界

沙箱提供 root 权限的 Shell 和文件操作，FastAPI 进程也以 root 运行；Chrome
启用了远程调试并关闭同源策略，VNC 当前使用 `-nopw`。这些设置用于本地学习和
调试，不应直接暴露到不可信网络。若部署到共享或生产环境，应移除宿主机端口映射、
限制网络访问，并为 VNC/CDP 和 API 增加认证与最小权限隔离。

Supervisor 的 FastAPI 进程默认不启用 `--reload`；本地开发需要显式设置
`UVI_ARGS=--reload`。
