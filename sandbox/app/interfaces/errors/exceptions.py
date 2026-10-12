import logging
from typing import Any

from fastapi import status


def log_level_for(status_code: int, expected: bool = False) -> int:
    """按状态码与"是否预期失败"决定日志级别，异常与 handler 共用同一规则。

    4xx 属预期业务失败（如 wait-process 轮询超时被高频触发），
    以及显式标记 expected 的异常（如 503 繁忙背压）降为 WARNING，
    避免 api 侧 1.5s 一次的 shell 输出轮询刷出 ERROR 日志洪水。
    """
    if 400 <= status_code < 500 or expected:
        return logging.WARNING
    return logging.ERROR


class AppException(Exception):
    """应用基础异常。

    只负责携带"错误语义"（msg / status_code / data），打日志的职责交给
    exception_handler 单一出口：异常可能在构造后被吞掉、也可能被复用，
    在 __init__ 里记日志会留下与实际响应不符的假日志。
    """

    # 预期内异常（如繁忙背压）不按 error 记日志，避免误导运维
    expected = False

    def __init__(
            self,
            msg: str = "应用发生错误请稍后尝试",
            status_code: int = status.HTTP_500_INTERNAL_SERVER_ERROR,
            data: Any = None,
    ) -> None:
        self.msg = msg
        self.status_code = status_code
        self.data = data
        super().__init__(self.msg)

    @property
    def log_level(self) -> int:
        """日志级别由异常自身语义决定，handler 只负责按此级别出口。"""
        return log_level_for(self.status_code, self.expected)


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


class TimeoutException(AppException):
    """动作超时：动作已开始但迟迟不返回，模型应调整策略而非立即重试。

    与 503「繁忙」的语义区分见 BusyException 文档：这里不携带 Retry-After，
    避免上层把"动作已开始"的超时误当成可重试的背压。
    """

    def __init__(self, msg: str = "沙箱动作执行超时，请调整策略后重试") -> None:
        super().__init__(msg=msg, status_code=status.HTTP_504_GATEWAY_TIMEOUT)


class NotFoundException(AppException):
    """资源未找到异常"""

    def __init__(self, msg: str = "资源未找到，请核实后尝试") -> None:
        super().__init__(msg=msg, status_code=status.HTTP_404_NOT_FOUND)


class BadRequestException(AppException):
    """错误请求异常"""

    def __init__(self, msg: str = "客户端请求错误，请检查后重试") -> None:
        super().__init__(msg=msg, status_code=status.HTTP_400_BAD_REQUEST)
