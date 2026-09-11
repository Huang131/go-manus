# ============================================================
# Stage 1: Builder - 编译 Go 代码
# ============================================================
FROM golang:1.24-alpine AS builder

# 支持多平台构建 (Apple Silicon / amd64)
ARG TARGETARCH

ENV GOTOOLCHAIN=auto
WORKDIR /build

# 先复制 go mod 文件，利用缓存
COPY go.mod go.sum ./

# 复制源代码
COPY . .

# 编译二进制文件
# CGO_ENABLED=0 表示静态编译，不依赖 libc
# 合并两个 cache mount 到一个 RUN，减少层数并加速增量构建
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} go build \
    -ldflags="-s -w" \
    -o go-manus \
    ./cmd/server

# ============================================================
# Stage 2: Runner - 运行镜像
# ============================================================
FROM alpine:3.19 AS runner

# 安装 ca-certificates (用于 HTTPS 请求) 和 tzdata
RUN apk add --no-cache ca-certificates tzdata && \
    # 创建非 root 用户
    adduser -D -g '' appuser

WORKDIR /app

# 从 builder 阶段复制文件
# 注意：不再拷贝 /build/pkg，静态编译不需要源码
COPY --from=builder /build/go-manus ./
COPY --from=builder /build/config.yaml ./
COPY --from=builder /build/migrations ./migrations/

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