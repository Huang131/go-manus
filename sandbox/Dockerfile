# ============================================================
# go-manus sandbox - Python + Node.js + Chrome 环境
# ============================================================
FROM ubuntu:22.04

# 合并所有 ENV 到一个 RUN 减少层数
ENV DEBIAN_FRONTEND=noninteractive \
    HOSTNAME=sandbox \
    LANG=zh_CN.UTF-8 \
    LANGUAGE=zh_CN:zh \
    LC_ALL=zh_CN.UTF-8 \
    CHROME_ARGS="" \
    UVI_ARGS=""

# ============================================================
# 阶段1: 配置 apt 源 + 基础依赖 + 语言支持
# 利用缓存: 源变化时才重新执行
# ============================================================
RUN sed -i 's|http://archive.ubuntu.com/ubuntu/|http://mirrors.aliyun.com/ubuntu/|g' /etc/apt/sources.list && \
    sed -i 's|http://security.ubuntu.com/ubuntu/|http://mirrors.aliyun.com/ubuntu/|g' /etc/apt/sources.list && \
    sed -i 's|http://ports.ubuntu.com/ubuntu-ports/|http://mirrors.aliyun.com/ubuntu-ports/|g' /etc/apt/sources.list && \
    # 基础依赖 + 语言支持 一起安装，减少 apt-get update 调用
    apt-get update && \
    apt-get install -y --no-install-recommends \
        sudo bc curl wget gnupg software-properties-common supervisor \
        xterm socat xvfb x11vnc websockify \
        fonts-noto-cjk fonts-noto-color-emoji language-pack-zh-hans locales && \
    locale-gen zh_CN.UTF-8 && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# 创建用户
RUN useradd -m -d /home/ubuntu -s /bin/bash ubuntu && \
    echo "ubuntu ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/ubuntu

# ============================================================
# 阶段2: Python 3.10
# 利用缓存: Python 版本不常变化
# ============================================================
RUN add-apt-repository ppa:deadsnakes/ppa -y && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
        python3.10 python3.10-venv python3.10-dev python3-pip && \
    update-alternatives --install /usr/bin/python3 python3 /usr/bin/python3.10 1 && \
    # 配置 pip 镜像源（仅作为 fallback）
    pip3 config set global.index-url https://mirrors.aliyun.com/pypi/simple/ && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# ============================================================
# 阶段3: uv 包管理器 + Node.js 24 + Chrome
# 使用 uv sync 安装依赖，比 pip 更快
# ============================================================
# 安装 uv
COPY --from=ghcr.io/astral-sh/uv:latest /uv /uvx /bin/

RUN mkdir -p /etc/apt/keyrings && \
    curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg && \
    echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_24.x nodistro main" | tee /etc/apt/sources.list.d/nodesource.list && \
    add-apt-repository ppa:xtradeb/apps -y && \
    apt-get update && \
    apt-get install -y --no-install-recommends chromium nodejs && \
    # 配置 npm 镜像源
    npm config set registry https://registry.npmmirror.com && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

WORKDIR /sandbox

# ============================================================
# 阶段4: Python 依赖 (使用 uv sync)
# 利用缓存: pyproject.toml/uv.lock 变化时才重新安装
# ============================================================
# 设置 uv 环境变量
ENV UV_PROJECT_ENVIRONMENT="/venv"

# 复制依赖配置文件（放在代码之前，利用缓存）
COPY pyproject.toml uv.lock ./

# 使用 uv sync 安装依赖 (--frozen 使用锁文件，--no-dev 不安装开发依赖)
RUN uv sync --frozen --no-dev

# ============================================================
# 阶段5: 复制代码和配置
# 代码放最后，减少缓存失效
# ============================================================
COPY . .

COPY supervisord.conf /etc/supervisor/conf.d/sandbox.conf

EXPOSE 8080 9222 5900 5901

# 健康检查：检查 /health 端点
HEALTHCHECK --interval=15s --timeout=10s --start-period=10s --retries=3 \
    CMD python3 -c "import urllib.request; urllib.request.urlopen('http://localhost:8080/health')" || exit 1

# 启动 supervisor 管理所有服务
CMD ["supervisord", "-n", "-c", "/sandbox/supervisord.conf"]