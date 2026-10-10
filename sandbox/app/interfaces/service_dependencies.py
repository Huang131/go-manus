from functools import lru_cache

from fastapi import Request

from app.services.browser import BrowserService
from app.services.file import FileService
from app.services.shell import ShellService
from app.services.supervisor import SupervisorService


@lru_cache()
def get_shell_service() -> ShellService:
    return ShellService()


@lru_cache()
def get_browser_service() -> BrowserService:
    return BrowserService()


@lru_cache()
def get_file_service() -> FileService:
    return FileService()


def get_supervisor_service(request: Request) -> SupervisorService:
    """从应用状态中取出 Supervisor 单例。

    不再用 lru_cache：实例改在 lifespan 中创建，好处是
    1.构造时一定有运行中的事件循环，超时任务可安全调度；
    2.中间件与端点走同一条解析路径，测试可直接替换 app.state.supervisor_service。
    """
    return request.app.state.supervisor_service
