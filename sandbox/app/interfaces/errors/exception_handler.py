import logging

from fastapi import FastAPI, Request, status
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse
from starlette.exceptions import HTTPException

from app.interfaces.schemas.base import Response
from .exceptions import AppException, BusyException

logger = logging.getLogger(__name__)


def register_exception_handlers(app: FastAPI) -> None:
    @app.exception_handler(RequestValidationError)
    async def request_validation_error_handler(req: Request, e: RequestValidationError) -> JSONResponse:
        """参数校验失败统一包装成 {code,msg,data} 信封，避免走 FastAPI 原生 422 格式"""
        errors = "; ".join(f"{'.'.join(str(loc) for loc in err.get('loc', []))}: {err.get('msg', '')}" for err in e.errors()[:3])
        logger.warning(f"RequestValidationError: {errors}")
        return JSONResponse(
            status_code=422,
            content=Response.fail(422, msg=f"请求参数校验失败: {errors}").model_dump(),
        )

    @app.exception_handler(BusyException)
    async def busy_exception_handler(req: Request, e: BusyException) -> JSONResponse:
        """沙箱繁忙（锁等待耗尽）：返回 503 并携带 Retry-After 提示重试时机。

        必须优先于 AppException 注册，FastAPI 按异常继承链匹配最接近的 handler。
        """
        # 日志出口统一收敛到 handler，按异常自带的 log_level 分级；
        # BusyException 属预期背压（expected=True），落 WARNING 而非 ERROR。
        level = e.log_level
        logger.log(level, "BusyException: %s (retry_after=%ss)", e.msg, e.retry_after)
        return JSONResponse(
            status_code=e.status_code,
            content=Response(code=e.status_code, msg=e.msg, data={}).model_dump(),
            headers={"Retry-After": str(e.retry_after)},
        )

    @app.exception_handler(AppException)
    async def app_exception_handler(req: Request, e: AppException) -> JSONResponse:
        """处理Manus沙箱自定义业务异常，将所有状态统一响应结构"""
        level = e.log_level
        # 5xx 才带堆栈：4xx 是预期业务失败，堆栈只会污染日志
        logger.log(level, "AppException: %s (code=%s)", e.msg, e.status_code, exc_info=level >= logging.ERROR)
        return JSONResponse(
            status_code=e.status_code,
            content=Response(
                code=e.status_code,
                msg=e.msg,
                data=e.data if e.data is not None else {}
            ).model_dump(),
        )

    @app.exception_handler(HTTPException)
    async def http_exception_handler(req: Request, e: HTTPException) -> JSONResponse:
        """处理FastAPI抛出的http异常，将所有状态统一响应结构"""
        # FastAPI 内建 HTTPException 也按 4xx/5xx 分级，避免 404 被记成 ERROR
        level = logging.WARNING if 400 <= e.status_code < 500 else logging.ERROR
        logger.log(level, "HttpException: %s (code=%s)", e.detail, e.status_code, exc_info=level >= logging.ERROR)
        return JSONResponse(
            status_code=e.status_code,
            content=Response(
                code=e.status_code,
                msg=e.detail,
                data={}
            ).model_dump(),
        )

    @app.exception_handler(Exception)
    async def exception_handler(req: Request, e: Exception) -> JSONResponse:
        """处理Manus沙箱服务中抛出的任意未定义异常，将所有状态码统一响应结构"""
        # 未捕获异常必定是 bug，用 exception() 带上完整堆栈与请求上下文，
        # 否则线上只能看到一句"Exception: xxx"却无法定位来源。
        logger.exception("未捕获异常: %s %s", req.method, req.url.path)
        return JSONResponse(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            content=Response(
                code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                msg="服务器出现异常请稍后尝试",
                data={}
            ).model_dump(),
        )
