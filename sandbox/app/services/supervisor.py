#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
@Time    : 2025/05/12 14:32
@Author  : thezehui@gmail.com
@File    : file.py
"""
import asyncio
import http.client
import logging
import socket
import xmlrpc.client
from datetime import datetime, timedelta
from typing import List, Any, Optional

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


class UnixStreamHTTPConnection(http.client.HTTPConnection):
    """基于Unix流的HTTP连接处理器"""

    def __init__(self, host: str, socket_path: str, timeout=None) -> None:
        """构造函数，完成连接处理器初始化"""
        http.client.HTTPConnection.__init__(self, host, timeout)
        self.socket_path = socket_path

    def connect(self) -> None:
        """重写连接方法，欺骗xml-rpc库让其觉得自己正在进行网络连接"""
        self.sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        self.sock.connect(self.socket_path)


class UnixStreamTransport(xmlrpc.client.Transport):
    """基于Unix流传输层的适配器/转换器"""

    def __init__(self, socket_path: str) -> None:
        """构造函数，完成传输适配器的初始化"""
        xmlrpc.client.Transport.__init__(self)
        self.socket_path = socket_path

    def make_connection(self, host) -> http.client.HTTPConnection:
        return UnixStreamHTTPConnection(host, self.socket_path)


class SupervisorService:
    """Supervisor服务"""

    def __init__(self) -> None:
        """构造函数，完成supervisor服务链接"""
        # 1.连接supervisor配置
        self.rpc_url = "/tmp/supervisor.sock"
        self._connect_rpc()

        # 2.supervisor超时配置
        settings = get_settings()
        self.timeout_active = settings.server_timeout_minutes is not None
        self.shutdown_task = None
        self.shutdown_time = None
        self._timer_generation = 0
        self._expand_enabled = True  # 是否自动保活(每调用一次接口就增加时间)
        self._state_lock = asyncio.Lock()
        self._rpc_lock = asyncio.Lock()

        # 3.检测是否配置了自动销毁
        if settings.server_timeout_minutes is not None:
            # 4.设置销毁时间+定时器
            self.shutdown_time = datetime.now() + timedelta(minutes=settings.server_timeout_minutes)
            self._setup_timer(settings.server_timeout_minutes)

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
    def expand_enabled(self) -> bool:
        """只读属性，返回是否自动保活"""
        return self._expand_enabled

    def enable_expand(self) -> None:
        """开启自动保活"""
        self._expand_enabled = True

    def disable_expand(self) -> None:
        """关闭自动保活"""
        self._expand_enabled = False

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

        try:
            loop = asyncio.get_running_loop()
        except RuntimeError:
            # SupervisorService 在 FastAPI 请求事件循环中创建；没有运行中的循环时
            # 无法安全调度异步 RPC，保留截止时间并等待后续 activate/extend 调度。
            logger.warning("当前没有运行中的事件循环，Supervisor 超时任务暂未调度")
            return
        task = loop.create_task(shutdown_after_timeout())
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
                transport=UnixStreamTransport(self.rpc_url),
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

    async def get_all_processes(self) -> List[ProcessInfo]:
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
                    await self._call_rpc(self.server.supervisor.stopProcess, name, True)
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
            running = [process["name"] for process in managed
                       if process.get("statename", "RUNNING") == "RUNNING"]
            for name in reversed(running):
                stopped.append(await self._call_rpc(self.server.supervisor.stopProcess, name, True))
            for process in managed:
                name = process["name"]
                started.append(await self._call_rpc(self.server.supervisor.startProcess, name, True))
            return SupervisorActionResult(status="restarted", stop_result=stopped, start_result=started)
        except Exception as e:
            logger.error(f"重启Supervisor子进程失败: {e}")
            raise AppException(f"重启Supervisor子进程失败: {e}")

    async def activate_timeout(self, minutes: Optional[int] = None) -> SupervisorTimeout:
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
        """在已持有状态锁时激活超时计时器。"""
        self.timeout_active = True
        self.shutdown_time = datetime.now() + timedelta(minutes=timeout_minutes)
        self._setup_timer(timeout_minutes)
        return SupervisorTimeout(
            status="timeout_activated",
            active=True,
            shutdown_time=self.shutdown_time.isoformat(),
            timeout_minutes=timeout_minutes,
            remaining_seconds=(self.shutdown_time - datetime.now()).total_seconds(),
        )

    async def extend_timeout(self, minutes: Optional[int] = 3) -> SupervisorTimeout:
        """传递指定的时长，延长超时销毁的时间，单默认延长3分钟"""
        # 1.获取超时分钟数
        if minutes is None:
            raise BadRequestException("超时时间未配置, 请核实后重试")
        if minutes <= 0:
            raise BadRequestException("延长时间必须大于0分钟")
        async with self._get_state_lock():
            # 检查必须与状态更新共用同一把锁，避免 cancel_timeout 并发清空截止时间。
            if getattr(self, "shutdown_time", None) is None:
                return self._activate_timeout_locked(minutes)
            remaining = self.shutdown_time - datetime.now()
            timeout_minutes = round(max(0, remaining.total_seconds()) / 60) + minutes
            self.timeout_active = True
            self.shutdown_time = datetime.now() + timedelta(minutes=timeout_minutes)
            self._setup_timer(timeout_minutes)
            return SupervisorTimeout(
                status="timeout_extended",
                active=True,
                shutdown_time=self.shutdown_time.isoformat(),
                timeout_minutes=timeout_minutes,
                remaining_seconds=(self.shutdown_time - datetime.now()).total_seconds(),
            )

    async def cancel_timeout(self) -> SupervisorTimeout:
        """取消超时销毁设置"""
        async with self._get_state_lock():
            if not self.timeout_active:
                return SupervisorTimeout(status="no_timeout_active", active=False)
            self._cancel_timeout_timer()
            self.timeout_active = False
            self.shutdown_time = None
            self._expand_enabled = True
            return SupervisorTimeout(status="timeout_cancelled", active=False)

    async def get_timeout_status(self) -> SupervisorTimeout:
        """获取当前supervisor的超时状态"""
        async with self._get_state_lock():
            if not self.timeout_active:
                return SupervisorTimeout(active=False)
            remaining_seconds = 0
            if self.shutdown_time:
                remaining = self.shutdown_time - datetime.now()
                remaining_seconds = max(0, remaining.total_seconds())
            return SupervisorTimeout(
                active=self.timeout_active,
                shutdown_time=self.shutdown_time.isoformat() if self.shutdown_time else None,
                remaining_seconds=remaining_seconds
            )
