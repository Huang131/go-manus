from functools import lru_cache

from fastapi import FastAPI, Request

from app.services.browser import BrowserService
from app.services.file import FileService
from app.services.shell import ShellService
from app.services.supervisor import SupervisorService

# app.state 上的属性名统一定义在这里，写入端（setup_services/shutdown_services）
# 与读取端（provider / 中间件）共用同一份常量，避免改名时漏改一处导致运行时 AttributeError。
SUPERVISOR_STATE_KEY = "supervisor_service"
SHELL_STATE_KEY = "shell_service"
BROWSER_STATE_KEY = "browser_service"


def setup_services(app: FastAPI) -> None:
    """在 lifespan 中构造状态型服务单例并挂到 app.state。

    Shell / Browser / Supervisor 都持有子进程、定时任务或事件循环内创建的锁，
    需要明确的构造时机与回收时机：
    1.在 lifespan 内构造，保证一定有运行中的事件循环，超时任务可安全调度；
    2.中间件与端点经 app.state 解析到同一实例；
    3.关闭时由 shutdown_services 回收同一实例，不存在"关错对象"的可能。

    无状态且无需回收的 FileService 不走 app.state，保持 lru_cache 惰性单例。
    """
    setattr(app.state, SUPERVISOR_STATE_KEY, SupervisorService())
    setattr(app.state, SHELL_STATE_KEY, ShellService())
    setattr(app.state, BROWSER_STATE_KEY, BrowserService())


async def shutdown_services(app: FastAPI) -> None:
    """回收 setup_services 创建的资源，与构造一一对应。"""
    # 关闭所有 Shell 会话进程与输出读取器，避免应用重启遗留子进程
    await getattr(app.state, SHELL_STATE_KEY).shutdown()
    # 取消待执行的超时销毁任务（只取消本地任务，不会关闭 supervisord）
    await getattr(app.state, SUPERVISOR_STATE_KEY).aclose()


def get_shell_service(request: Request) -> ShellService:
    return getattr(request.app.state, SHELL_STATE_KEY)


def get_browser_service(request: Request) -> BrowserService:
    return getattr(request.app.state, BROWSER_STATE_KEY)


def get_supervisor_service(request: Request) -> SupervisorService:
    """从应用状态中取出 Supervisor 单例。

    不用 lru_cache：实例改在 lifespan 中创建（见 setup_services），好处是
    1.构造时一定有运行中的事件循环，超时任务可安全调度；
    2.中间件与端点走同一条解析路径，测试可直接替换 app.state.supervisor_service。
    """
    return getattr(request.app.state, SUPERVISOR_STATE_KEY)


@lru_cache(maxsize=None)
def get_file_service() -> FileService:
    """FileService 无子进程与定时任务，构造不依赖事件循环，保持进程级惰性单例。"""
    return FileService()
