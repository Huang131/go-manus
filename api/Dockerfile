# ============================================================
# Stage 1: Builder - 编译 Go 代码
# ============================================================
FROM golang:1.23-alpine AS builder

# 安装构建工具
RUN apk add --no-cache git make

# 设置 Go 工具链自动下载
ENV GOTOOLCHAIN=auto

WORKDIR /build

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖 (利用 Docker 缓存层)
RUN go mod download

# 复制源代码
COPY . .

# 编译二进制文件
# CGO_ENABLED=0 表示静态编译，不依赖 libc
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o go-manus \
    ./cmd/server

# ============================================================
# Stage 2: Runner - 运行镜像
# ============================================================
FROM alpine:3.19 AS runner

# 安装 ca-certificates (用于 HTTPS 请求) 和 tzdata
RUN apk add --no-cache ca-certificates tzdata

# 创建非 root 用户
RUN adduser -D -g '' appuser

WORKDIR /app

# 从 builder 阶段复制二进制文件
COPY --from=builder /build/go-manus .
COPY --from=builder /build/config.yaml .
COPY --from=builder /build/.env.example .env

# 复制数据库迁移脚本
COPY --from=builder /build/migrations ./migrations

# 复制 pkg 目录 (如果需要)
COPY --from=builder /build/pkg ./pkg

# 设置文件权限
RUN chown -R appuser:appuser /app

# 切换到非 root 用户
USER appuser

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 启动命令
ENTRYPOINT ["./go-manus"]