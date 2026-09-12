# Manus 前端 UI

基于 Next.js 构建的前端用户界面，提供会话管理、AI 对话、远程桌面（VNC）等交互功能。

## 技术栈

- Next.js 16 (React 19)
- TypeScript
- Tailwind CSS 4
- Radix UI (组件库)
- noVNC (远程桌面)

## 项目结构

```
ui/
├── src/
│   ├── app/               # 页面路由
│   │   ├── page.tsx       # 首页
│   │   └── sessions/      # 会话页面
│   ├── components/        # 组件
│   │   ├── ui/            # 基础 UI 组件
│   │   └── tool-use/      # 工具使用相关组件
│   ├── lib/
│   │   ├── api/           # API 客户端（REST + SSE 封装）
│   │   ├── session-events.ts  # 任务事件 → 时间线归并
│   │   └── utils.ts       # 通用工具
│   ├── hooks/             # 自定义 Hooks（use-session-detail 等）
│   └── providers/         # Context Providers
│       ├── sessions-provider.tsx  # 会话列表（SSE 流式刷新）
│       └── models-provider.tsx    # 模型列表全局共享
├── public/                # 静态资源
├── next.config.ts         # Next.js 配置
├── Dockerfile
├── package.json
└── tsconfig.json
```

## API 调用

项目通过环境变量 `NEXT_PUBLIC_API_BASE_URL` 配置 API 地址：

- **开发环境**：默认 `http://localhost:8000/api`（直连 API 服务）
- **生产环境**：构建时设置为 `/api`（通过 Nginx 反向代理）

## 模型选择（Auto）

聊天输入框的模型选择器提供「Auto · 跟随系统」选项（默认选中）：

- **Auto**：不携带 `model_id`，由后端按模型可用性路由；DB 模型全部失败时
  自动降级到部署配置文件（env）中的兜底模型
- **选定具体模型**：粘性路由——该模型失败时直接报错，不静默切换；
  界面会提供「切换 Auto 重试」的快速恢复入口

模型列表在「设置 → 模型提供商」中维护，保存后聊天选择器即时生效。

## 本地开发

### 环境准备

- Node.js >= 22
- npm >= 10

### 安装与启动

```bash
# 安装依赖
npm install

# 启动开发服务器
npm run dev
```

开发服务器默认运行在 `http://localhost:3000`，API 请求自动转发到 `http://localhost:8000/api`。

### 构建

```bash
npm run build
npm run start
```

## Docker 部署

UI 服务通过根目录的 `docker-compose.yml` 统一部署。Dockerfile 采用多阶段构建：

1. **deps** - 安装 npm 依赖
2. **builder** - 构建 Next.js 应用（standalone 模式）
3. **runner** - 最小化生产镜像

构建时通过 `NEXT_PUBLIC_API_BASE_URL=/api` 参数将 API 地址设置为相对路径，由 Nginx 统一代理。
