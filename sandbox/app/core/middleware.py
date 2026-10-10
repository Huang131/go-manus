import logging

from fastapi import Request

logger = logging.getLogger(__name__)


async def auto_extend_timeout_middleware(request: Request, call_next):
    """使用中间件延长每次API请求的超时销毁时间"""
    # 1.从应用状态取 supervisor 单例，与端点共用同一实例
    supervisor_service = request.app.state.supervisor_service

    # 2.判断逻辑收口在 should_auto_extend，仅在符合条件时延长超时销毁时间3分钟
    if supervisor_service.should_auto_extend(request.url.path):
        try:
            await supervisor_service.extend_timeout(3)
            logger.debug("调用API请求而自动延长超时销毁时长: %s", request.url.path)
        except Exception as e:
            logger.warning("自动延长超时失败: %s", str(e))

    return await call_next(request)
