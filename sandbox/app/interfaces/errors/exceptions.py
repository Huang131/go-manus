#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
@Time    : 2025/05/10 23:56
@Author  : thezehui@gmail.com
@File    : exceptions.py
"""
import logging
from typing import Any

from fastapi import status

logger = logging.getLogger(__name__)


class AppException(Exception):
    """应用基础异常"""

    # 预期内异常（如繁忙背压）不按 error 记日志，避免误导运维
    expected = False

    def __init__(
            self,
            msg: str = "应用发生错误请稍后尝试",
            status_code: int = status.HTTP_500_INTERNAL_SERVER_ERROR,
            data: Any = None
    ) -> None:
        """构造函数，完成异常的初始化"""
        # 1.完成数据初始化
        self.msg = msg
        self.status_code = status_code
        self.data = data

        # 2.记录日志并调用父类构造函数。
        # 4xx 属预期业务失败（如 wait-process 轮询超时被高频触发），
        # 以及显式标记 expected 的异常（如 503 繁忙背压）降为 WARNING，
        # 避免 api 侧 1.5s 一次的 shell 输出轮询刷出 ERROR 日志洪水。
        if 400 <= status_code < 500 or self.expected:
            log_fn = logger.warning
        else:
            log_fn = logger.error
        log_fn(f"沙箱发生错误: {msg} (code: {status_code})")
        super().__init__(self.msg)


class BusyException(AppException):
    """沙箱繁忙：浏览器动作全局串行，锁等待耗尽预算时触发，可稍后重试。

    与 504「动作超时」的语义区分：
    - 503 繁忙：动作根本没开始，锁被占着，模型可择机重试；
    - 504 超时：动作已开始但页面迟迟不反馈，模型应调整策略而非立即重试。
    """

    expected = True

    def __init__(self, msg: str = "沙箱浏览器正忙，请稍后重试", retry_after: int = 3) -> None:
        super().__init__(msg=msg, status_code=status.HTTP_503_SERVICE_UNAVAILABLE)
        self.retry_after = retry_after


class NotFoundException(AppException):
    """资源未找到异常"""

    def __init__(self, msg: str = "资源未找到，请核实后尝试") -> None:
        super().__init__(msg=msg, status_code=status.HTTP_404_NOT_FOUND)


class BadRequestException(AppException):
    """错误请求异常"""

    def __init__(self, msg: str = "客户端请求错误，请检查后重试") -> None:
        super().__init__(msg=msg, status_code=status.HTTP_400_BAD_REQUEST)
