import logging
import sys
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.core.config import get_settings
from app.core.middleware import auto_extend_timeout_middleware
from app.interfaces.endpoints.routes import router
from app.interfaces.errors.exception_handler import register_exception_handlers
from app.interfaces.service_dependencies import get_shell_service

# 控制台处理器的唯一标识，用于 setup_logging 的幂等判断。
CONSOLE_HANDLER_NAME = "sandbox-console"


def setup_logging() -> None:
    """设置沙箱API应用日志"""
    # 获取项目配置
    settings = get_settings()

    # 获取根日志处理器
    root_logger = logging.getLogger()

    # 设置根日志处理器等级（settings.log_level 由 Literal 约束，取值一定合法）
    log_level = getattr(logging, settings.log_level)
    root_logger.setLevel(log_level)

    # 日志输出格式定义
    formatter = logging.Formatter(
        '%(asctime)s - %(name)s - %(levelname)s - %(message)s',
        datefmt='%Y-%m-%d %H:%M:%S'
    )

    # 创建控制台日志输出处理器
    console_handler = logging.StreamHandler(sys.stdout)
    console_handler.setFormatter(formatter)
    console_handler.setLevel(log_level)
    console_handler.set_name(CONSOLE_HANDLER_NAME)

    # 将控制台日志处理器添加到根日志处理器中
    # reload 或测试环境可能重复初始化模块，避免重复输出同一条日志。
    # 只按自定义名称判断：若改成判断「是否已有 StreamHandler」，第三方注册的
    # 处理器会让条件为假，导致本处理器静默不生效。
    if not any(handler.get_name() == CONSOLE_HANDLER_NAME for handler in root_logger.handlers):
        root_logger.addHandler(console_handler)

    root_logger.info("沙箱系统日志模块初始化完成")


@asynccontextmanager
async def lifespan(app: FastAPI):
    """FastAPI生命周期上下文管理器"""
    # yield 之前：应用启动阶段，此时还未开始接收请求
    logger.info("Manus沙箱正在初始化")

    try:
        # yield 是分界线：在此把控制权交回 FastAPI，之后才开始接收请求
        yield
    finally:
        # yield 之后：应用关闭阶段，finally 保证异常退出时也会回收资源
        await get_shell_service().shutdown()
        logger.info("Manus沙箱关闭成功")


# 1.初始化日志系统
setup_logging()
logger = logging.getLogger(__name__)

# 2.定义FastAPI路由tags标签
openapi_tags = [
    {
        "name": "文件模块",
        "description": "包含 **文件增删改查** 等 API 接口，用于实现对沙箱文件的操作。",
    },
    {
        "name": "Shell模块",
        "description": "包含 **执行/查看Shell** 等 API 接口，用于实现操控沙箱内部的 Shell 命令。",
    },
    {
        "name": "Supervisor模块",
        "description": "使用接口+Supervisor实现管理沙箱系统的程序逻辑",
    },
    {
        "name": "浏览器模块",
        "description": "包含 **导航/快照/截图/点击/输入/滚动/控制台** 等 API 接口，用于操控沙箱内的 Chrome。",
    },
]

# 3.实例化FastAPI项目实例
app = FastAPI(
    title="Manus沙箱系统",
    description="该沙箱系统中预装了Chrome、Python、Node.js，支持运行 Shell 命令、文件管理等功能",
    openapi_tags=openapi_tags,
    lifespan=lifespan,
    version="0.1.0",  # 与 pyproject.toml 的 version 保持一致
)

# 4.添加中间件
# 顺序说明：Starlette 的中间件是洋葱模型，后注册的位于更外层、更先执行。
# 因此 CORSMiddleware 在 auto_extend 之外，浏览器的 OPTIONS 预检请求会在外层
# 被处理并返回，不会触发沙箱超时延长。调整这两行的顺序会改变该行为。
app.middleware("http")(auto_extend_timeout_middleware)
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    # 学习环境允许跨域，但不携带 Cookie，避免通配 origin 与 credentials 冲突。
    allow_credentials=False,
    allow_methods=["*"],
    allow_headers=["*"],
)

# 5.注册全局异常处理器
register_exception_handlers(app)

# 6.健康检查端点
# 故意不套 {code, msg, data} 响应信封：探针（含 Dockerfile HEALTHCHECK）
# 只判断 HTTP 状态码，保持响应体最小。
@app.get("/health", tags=["健康检查"])
async def health_check():
    """健康检查端点，供 API 服务检测沙箱状态"""
    return {"status": "healthy"}

# 7.集成路由
app.include_router(router, prefix="/api")
