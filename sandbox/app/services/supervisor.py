import asyncio
import http.client
import logging
import socket
import xmlrpc.client
import time
from datetime import datetime, timedelta
from typing import Any

from app.core.config import get_settings
from app.interfaces.errors.exceptions import BadRequestException, AppException
from app.models.supervisor import ProcessInfo, SupervisorActionResult, SupervisorTimeout

"""
1.Supervisor启动后，通过一个Unix套接字文件来实现通信(rpc协议)
2.连接这个通信文件，/tmp/supervisor.sock (xml-rpc连接)
3.使用某种方式来完整转换，让xml-rpc实现连接supervisor.sock
4.连接之后我们就可以调用rpc对应的方法，getAllProcessInfo()
"""

logger = logging.getLogger(__name__)

# 不触发自动保活的路径：这些端点是对超时的显式操作，不应再被中间件叠加一次延长。
AUTO_EXTEND_EXCLUDED_PATHS = frozenset({
    "/api/supervisor/activate-timeout",
    "/api/supervisor/extend-timeout",
    "/api/supervisor/cancel-timeout",
    "/api/supervisor/timeout-status",
})

# 每次 API 请求自动续期的时长（分钟）。活动保活与显式续期的默认值共用此常量，
# 避免"改了默认值忘了改调用点"导致行为不一致。
AUTO_EXTEND_MINUTES = 3


class UnixStreamHTTPConnection(http.client.HTTPConnection):
    """基于Unix流的HTTP连接处理器"""

    def __init__(self, host: str, socket_path: str, timeout=None) -> None:
        """构造函数，完成连接处理器初始化"""
        http.client.HTTPConnection.__init__(self, host, timeout)
        self.socket_path = socket_path
        self.timeout = timeout

    def connect(self) -> None:
        """重写连接方法，欺骗xml-rpc库让其觉得自己正在进行网络连接"""
        self.sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        if self.timeout is not None:
            self.sock.settimeout(self.timeout)
        self.sock.connect(self.socket_path)


class UnixStreamTransport(xmlrpc.client.Transport):
    """基于Unix流传输层的适配器/转换器"""

    def __init__(self, socket_path: str, timeout=None) -> None:
        """构造函数，完成传输适配器的初始化"""
        xmlrpc.client.Transport.__init__(self)
        self.socket_path = socket_path
        self.timeout = timeout

    def make_connection(self, host) -> http.client.HTTPConnection:
        return UnixStreamHTTPConnection(host, self.socket_path, self.timeout)


class SupervisorService:
    """Supervisor服务"""

    def __init__(self) -> None:
        """构造函数，完成supervisor服务链接"""
        # 1.连接supervisor配置
        self.rpc_url = "/tmp/supervisor.sock"
        self._connect_rpc()

        # 2.supervisor超时状态
        # _deadline 是超时状态的唯一真源：timeout_active、shutdown_time 均由它派生，
        # 避免"状态位"与"实际是否存在定时器"不同步。
        self._deadline: datetime | None = None
        self.shutdown_task = None
        self._timer_generation = 0
        self._auto_extend = True  # 是否自动保活(每调用一次接口就增加时间)
        self._state_lock = asyncio.Lock()
        self._rpc_lock = asyncio.Lock()

        # 3.检测是否配置了默认超时，有则预置销毁时间+定时器
        default_minutes = get_settings().server_timeout_minutes
        if default_minutes is not None:
            self._arm_timer(default_minutes)

    def _get_state_lock(self) -> asyncio.Lock:
        """懒加载状态锁，兼容测试中通过 __new__ 构造的服务实例。"""
        lock = getattr(self, "_state_lock", None)
        if lock is None:
            lock = asyncio.Lock()
            self._state_lock = lock
        return lock

    def _get_rpc_lock(self) -> asyncio.Lock:
        """复用 RPC 锁，避免多个线程并发使用同一 XML-RPC 连接。"""
        lock = getattr(self, "_rpc_lock", None)
        if lock is None:
            lock = asyncio.Lock()
            self._rpc_lock = lock
        return lock

    @property
    def timeout_active(self) -> bool:
        """是否存在生效中的销毁截止时间（派生自 _deadline，不单独存储）"""
        return self._deadline is not None

    @property
    def shutdown_time(self) -> datetime | None:
        """当前销毁截止时间；未激活超时时为 None"""
        return self._deadline

    def should_auto_extend(self, path: str) -> bool:
        """判断该请求是否需要自动延长销毁时间，供中间件调用。

        把路径白名单与策略判断收口在此，中间件只负责调用，
        避免白名单同时在中间件和路由表里各维护一份。

        三个条件缺一不可：存在生效中的销毁计划、路径属于 /api/ 且不是对超时的
        显式操作、保活开关未被用户关掉。timeout_active 是硬性前提——
        cancel-timeout 会清空销毁计划，此时已无计划可延长，
        沙箱保持存活，直到被显式重启或停止。
        """
        return (
            self.timeout_active
            and path.startswith("/api/")
            and path not in AUTO_EXTEND_EXCLUDED_PATHS
            and self._auto_extend
        )

    def _arm_timer(self, timeout_minutes: int) -> None:
        """写入截止时间并调度定时器。

        调用方需处于事件循环中，并自行保证状态互斥（构造期或已持有 _state_lock）。
        """
        self._deadline = datetime.now() + timedelta(minutes=timeout_minutes)
        self._setup_timer(timeout_minutes)

    def _setup_timer(self, minutes: int) -> None:
        """传递时间(分钟)并创建定时器，在时间结束之后关闭supervisord主进程"""
        self._timer_generation = getattr(self, "_timer_generation", 0) + 1
        generation = self._timer_generation
        # 1.检测当前是否存在销毁任务，如果存在则先取消
        previous_task = getattr(self, "shutdown_task", None)
        if previous_task is not None:
            previous_task.cancel()
            self.shutdown_task = None

        # 2.创建一个异步定时器任务函数
        async def shutdown_after_timeout():
            await asyncio.sleep(minutes * 60)
            # 旧任务即使在取消竞态中醒来，也不能关闭已被重新设置的服务。
            if generation != getattr(self, "_timer_generation", 0):
                return
            await self.shutdown()

        # 单例在 lifespan 中创建、其余调用都发生在请求上下文内，
        # 因此此处必然有运行中的事件循环；没有循环属于编程错误，让其直接抛出。
        task = asyncio.get_running_loop().create_task(shutdown_after_timeout())
        self.shutdown_task = task

        def on_timer_done(completed_task: asyncio.Task) -> None:
            if self.shutdown_task is completed_task:
                self.shutdown_task = None
            if completed_task.cancelled():
                return
            error = completed_task.exception()
            if error is not None:
                logger.error("Supervisor 自动关闭任务失败: %s", error)

        task.add_done_callback(on_timer_done)

    def _cancel_timeout_timer(self) -> None:
        """取消当前定时器并使已排队的旧回调失效。"""
        self._timer_generation = getattr(self, "_timer_generation", 0) + 1
        task = getattr(self, "shutdown_task", None)
        if task is not None:
            task.cancel()
            self.shutdown_task = None

    def _connect_rpc(self) -> None:
        """使用python的xml-rpc客户端连接一个本地sock文件文件实现连接rpc服务"""
        try:
            self.server = xmlrpc.client.ServerProxy(
                "http://localhost",
                transport=UnixStreamTransport(self.rpc_url, timeout=8),
            )
        except Exception as e:
            logger.error(f"连接Supervisor服务失败: {str(e)}")
            raise BadRequestException(f"连接Supervisor服务失败: {str(e)}")

    async def _call_rpc(self, method, *args) -> Any:
        """根据传递的方法+参数调用rpc方法"""
        try:
            async with self._get_rpc_lock():
                return await asyncio.to_thread(method, *args)
        except Exception as e:
            logger.error(f"RPC方法调用失败: {str(e)}")
            raise BadRequestException(f"RPC方法调用失败: {str(e)}")

    @staticmethod
    def _namespec(process: dict) -> str:
        """把 getAllProcessInfo 返回的进程信息还原成 supervisor 的 namespec。

        getAllProcessInfo 返回的是拆开的 name + group，而 stopProcess/startProcess
        接受的是 group:name 形式的 namespec。进程被收进 supervisord.conf 的
        [group:services] 后，只传短名会被当成同名的组去查找（名为 app 的组并不
        存在，真实的组是 services），直接抛 BAD_NAME；因此调用前必须拼回 namespec。
        """
        name = process.get("name")
        group = process.get("group")
        return name if group in (None, "", name) else f"{group}:{name}"

    async def get_all_processes(self) -> list[ProcessInfo]:
        """获取当前supervisor管理的所有进程信息"""
        try:
            processes = await self._call_rpc(self.server.supervisor.getAllProcessInfo)
            return [ProcessInfo(**process) for process in processes]
        except Exception as e:
            logger.error(f"获取进程信息失败: {str(e)}")
            raise AppException(f"获取进程信息失败: {str(e)}")

    async def stop_all_processes(self) -> SupervisorActionResult:
        """停止除 app 外的全部受管进程。

        supervisor 的 stopAllProcesses 会连本 HTTP 服务一起停（响应发不回去），
        因此逐个停止、排除 app 自身，与 restart 的排除逻辑对称。
        """
        try:
            processes = await self._call_rpc(self.server.supervisor.getAllProcessInfo)
            stopped = []
            for process in processes:
                name = process.get("name")
                if name == "app":
                    continue
                if process.get("statename", "RUNNING") == "RUNNING":
                    await self._call_rpc(
                        self.server.supervisor.stopProcess, self._namespec(process), True
                    )
                    stopped.append(name)
            return SupervisorActionResult(status="stopped", result={"stopped": stopped})
        except Exception as e:
            logger.error(f"停止supervisor所有进程服务失败: {str(e)}")
            raise AppException(f"停止supervisor所有进程服务失败: {str(e)}")

    async def shutdown(self) -> SupervisorActionResult:
        """关闭supervisord服务"""
        try:
            shutdown_result = await self._call_rpc(self.server.supervisor.shutdown)
            return SupervisorActionResult(status="shutdown", shutdown_result=shutdown_result)
        except Exception as e:
            logger.error(f"关闭supervisord服务失败: {str(e)}")
            raise AppException(f"关闭supervisord服务失败: {str(e)}")

    async def restart(self) -> SupervisorActionResult:
        """重启非 API 子进程，避免停止当前 HTTP 服务。"""
        try:
            processes = await self._call_rpc(self.server.supervisor.getAllProcessInfo)
            managed = [process for process in processes if process.get("name") != "app"]
            stopped = []
            started = []
            # 只停止确实运行中的进程；已停止/异常进程直接启动，避免 stopProcess 报错。
            deadline = time.monotonic() + 8  # 总超时：单进程卡住不拖死整个请求
            # 先统一算好 namespec 再执行 RPC，避免边遍历边调用时中途失败留下半停状态。
            running = [self._namespec(process) for process in managed
                       if process.get("statename", "RUNNING") == "RUNNING"]
            for namespec in reversed(running):
                if time.monotonic() > deadline:
                    logger.warning("restart 总超时，剩余进程跳过停止")
                    break
                stopped.append(
                    await self._call_rpc(self.server.supervisor.stopProcess, namespec, True)
                )
            for process in managed:
                if time.monotonic() > deadline:
                    logger.warning("restart 总超时，剩余进程跳过启动")
                    break
                started.append(
                    await self._call_rpc(
                        self.server.supervisor.startProcess, self._namespec(process), True
                    )
                )
            return SupervisorActionResult(status="restarted", stop_result=stopped, start_result=started)
        except Exception as e:
            logger.error(f"重启Supervisor子进程失败: {e}")
            raise AppException(f"重启Supervisor子进程失败: {e}")

    async def activate_timeout(self, minutes: int | None = None) -> SupervisorTimeout:
        """传递指定分钟，并激活定时销毁任务同时关闭自动保活"""
        # 1.获取超时分钟数
        setting = get_settings()
        timeout_minutes = setting.server_timeout_minutes if minutes is None else minutes
        if timeout_minutes is None:
            raise BadRequestException("超时时间未配置, 并且未读取到系统默认超时时间")
        if timeout_minutes <= 0:
            raise BadRequestException("超时时间必须大于0分钟")

        async with self._get_state_lock():
            return self._activate_timeout_locked(timeout_minutes)

    def _activate_timeout_locked(self, timeout_minutes: int) -> SupervisorTimeout:
        """在已持有状态锁时激活超时计时器，并关闭自动保活。

        关保活收在锁内而不是端点层：任何调用方（HTTP 端点/内部任务）行为都一致，
        也避免"先激活后关保活"的顺序依赖只散落在调用方。activate 仅由端点调用，
        收在这里没有副作用；extend 与中间件的活动保活共用同一路径，
        故那边的开关处理必须与 keep_alive 分开，详见 keep_alive。
        """
        self._auto_extend = False
        self._arm_timer(timeout_minutes)
        return SupervisorTimeout(
            status="timeout_activated",
            active=True,
            shutdown_time=self._deadline.isoformat(),
            timeout_minutes=timeout_minutes,
            remaining_seconds=(self._deadline - datetime.now()).total_seconds(),
        )

    async def keep_alive(self) -> None:
        """活动保活：由中间件在每次 API 请求时调用，把销毁时间顺延固定时长。

        必须与 extend_timeout 分开：extend_timeout 是用户的显式操作，会关闭自动保活；
        而本方法由请求流量反复触发，绝不能改动开关——否则第一个请求就把保活关掉，
        之后的请求不再续期，保活退化成"只生效一次"。
        """
        async with self._get_state_lock():
            # 用户已显式接管生命周期时，请求流量不得再推动销毁时间。
            if not self._auto_extend:
                return
            self._extend_locked(AUTO_EXTEND_MINUTES)

    async def extend_timeout(self, minutes: int | None = AUTO_EXTEND_MINUTES) -> SupervisorTimeout:
        """显式延长超时销毁时间（用户接管生命周期，同时关闭自动保活）"""
        if minutes is None:
            raise BadRequestException("超时时间未配置, 请核实后重试")
        if minutes <= 0:
            raise BadRequestException("延长时间必须大于0分钟")
        async with self._get_state_lock():
            self._auto_extend = False
            return self._extend_locked(minutes)

    def _extend_locked(self, minutes: int) -> SupervisorTimeout:
        """在已持有状态锁时顺延销毁时间：有生效计划则在剩余时间上叠加，否则从当前时刻起算。"""
        if self._deadline is None:
            # 无生效计划（如刚 cancel 过）：不能去叠加不存在的剩余时间，直接按本时长起算。
            status = "timeout_activated"
            timeout_minutes = minutes
        else:
            remaining = self._deadline - datetime.now()
            status = "timeout_extended"
            timeout_minutes = round(max(0, remaining.total_seconds()) / 60) + minutes
        self._arm_timer(timeout_minutes)
        return SupervisorTimeout(
            status=status,
            active=True,
            shutdown_time=self._deadline.isoformat(),
            timeout_minutes=timeout_minutes,
            remaining_seconds=(self._deadline - datetime.now()).total_seconds(),
        )

    async def cancel_timeout(self) -> SupervisorTimeout:
        """取消超时销毁设置"""
        async with self._get_state_lock():
            if self._deadline is None:
                return SupervisorTimeout(status="no_timeout_active", active=False)
            self._cancel_timeout_timer()
            self._deadline = None
            # 重新打开保活开关。但销毁计划已清空，should_auto_extend 的 timeout_active
            # 前提会挡住请求保活，于是 cancel 后沙箱进入"无销毁计划"状态，不会因流量被续期。
            self._auto_extend = True
            return SupervisorTimeout(status="timeout_cancelled", active=False)

    async def get_timeout_status(self) -> SupervisorTimeout:
        """获取当前supervisor的超时状态"""
        async with self._get_state_lock():
            if self._deadline is None:
                return SupervisorTimeout(active=False)
            remaining = self._deadline - datetime.now()
            return SupervisorTimeout(
                active=True,
                shutdown_time=self._deadline.isoformat(),
                remaining_seconds=max(0, remaining.total_seconds()),
            )

    async def aclose(self) -> None:
        """释放超时资源：取消待执行的销毁定时器任务。

        仅取消本地 asyncio 任务，绝不调用 supervisor.shutdown()——
        那会关闭 supervisord 主进程并拖垮整个容器。
        """
        async with self._get_state_lock():
            self._cancel_timeout_timer()
            self._deadline = None
