import logging

from fastapi import Request

logger = logging.getLogger(__name__)


async def auto_extend_timeout_middleware(request: Request, call_next):
    """使用中间件延长每次API请求的超时销毁时间"""
    # 1.从应用状态取 supervisor 单例，与端点共用同一实例
    supervisor_service = request.app.state.supervisor_service

    # 2.判断逻辑收口在 should_auto_extend，命中则触发一次活动保活；
    #    keep_alive 只顺延销毁时间，不改动用户的保活开关（与显式续期 extend_timeout 区分）。
    if supervisor_service.should_auto_extend(request.url.path):
        try:
            await supervisor_service.keep_alive()
            logger.debug("调用API请求而自动延长超时销毁时长: %s", request.url.path)
        except Exception as e:
            logger.warning("自动延长超时失败: %s", str(e))

    return await call_next(request)
