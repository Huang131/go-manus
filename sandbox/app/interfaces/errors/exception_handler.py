#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
@Time    : 2025/05/11 0:01
@Author  : thezehui@gmail.com
@File    : exception_handler.py
"""
import logging

from fastapi import FastAPI, Request, status
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse
from starlette.exceptions import HTTPException

from app.interfaces.schemas.base import Response
from .exceptions import AppException

logger = logging.getLogger(__name__)


def register_exception_handlers(app: FastAPI) -> None:
    @app.exception_handler(RequestValidationError)
    async def request_validation_error_handler(req: Request, e: RequestValidationError) -> JSONResponse:
        """参数校验失败统一包装成 {code,msg,data} 信封，避免走 FastAPI 原生 422 格式"""
        errors = "; ".join(f"{'.'.join(str(loc) for loc in err.get('loc', []))}: {err.get('msg', '')}" for err in e.errors()[:3])
        logger.warning(f"RequestValidationError: {errors}")
        return JSONResponse(
            status_code=422,
            content=Response.error(msg=f"请求参数校验失败: {errors}"),
        )

    @app.exception_handler(AppException)
    async def app_exception_handler(req: Request, e: AppException) -> JSONResponse:
        """处理Manus沙箱自定义业务异常，将所有状态统一响应结构"""
        logger.error(f"AppException: {e.msg}")
        return JSONResponse(
            status_code=e.status_code,
            content=Response(
                code=e.status_code,
                msg=e.msg,
                data={}
            ).model_dump(),
        )

    @app.exception_handler(HTTPException)
    async def http_exception_handler(req: Request, e: HTTPException) -> JSONResponse:
        """处理FastAPI抛出的http异常，将所有状态统一响应结构"""
        logger.error(f"HttpException: {e.detail}")
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
        logger.error(f"Exception: {str(e)}")
        return JSONResponse(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            content=Response(
                code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                msg="服务器出现异常请稍后尝试",
                data={}
            ).model_dump(),
        )
