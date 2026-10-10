import logging

from fastapi import Request

from app.interfaces.service_dependencies import get_supervisor_service

logger = logging.getLogger(__name__)


async def auto_extend_timeout_middleware(request: Request, call_next):
    """使用中间件延长每次API请求的超时销毁时间"""
    # 1.获取supervisor服务
    supervisor_service = get_supervisor_service()

    # 2.判断逻辑，仅在符合条件时延长超时销毁时间3分钟
    ignore_paths = (
        "/api/supervisor/activate-timeout",
        "/api/supervisor/extend-timeout",
        "/api/supervisor/cancel-timeout",
        "/api/supervisor/timeout-status",
    )
    # server_timeout_minutes 带默认值、类型非 Optional，判断其是否为 None 恒为真，
    # 是否启用自动延长由 supervisor_service.expand_enabled 决定。
    if (
            supervisor_service.timeout_active
            and request.url.path.startswith("/api/")
            and request.url.path not in ignore_paths
            and supervisor_service.expand_enabled
    ):
        try:
            await supervisor_service.extend_timeout(3)
            logger.debug("调用API请求而自动延长超时销毁时长: %s", request.url.path)
        except Exception as e:
            logger.warning("自动延长超时失败: %s", str(e))

    response = await call_next(request)
    return response
