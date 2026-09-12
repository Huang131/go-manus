#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
@Time    : 2025/05/11 23:25
@Author  : thezehui@gmail.com
@File    : shell.py
"""
import asyncio
import codecs
import getpass
import logging
import os.path
import re
import signal
import socket
import uuid
import time
from typing import Dict, Optional, List

from app.interfaces.errors.exceptions import (
    BadRequestException,
    AppException,
    NotFoundException,
)
from app.models.shell import (
    Shell,
    ConsoleRecord,
    ShellWaitResult,
    ShellWriteResult,
    ShellKillResult, ShellReadResult, ShellExecuteResult,
)

logger = logging.getLogger(__name__)

# 单个会话输出缓冲上限，超出后截断头部保留尾部，防止长会话内存无限增长
MAX_OUTPUT_CHARS = 512 * 1024
MAX_CONSOLE_RECORDS = 100
SESSION_IDLE_SECONDS = 30 * 60


class ShellService:
    """Shell命令服务"""
    active_shells: Dict[str, Shell]

    def __init__(self) -> None:
        self.active_shells = {}
        # 输出读取器 task 引用：必须持有引用，否则可能被事件循环 GC 中途取消
        self.reader_tasks: Dict[str, asyncio.Task] = {}
        self.session_locks: Dict[str, asyncio.Lock] = {}
        self.session_last_access: Dict[str, float] = {}

    def _get_session_lock(self, session_id: str) -> asyncio.Lock:
        """为同一会话复用锁，串行化进程切换和输入写入。"""
        return self.session_locks.setdefault(session_id, asyncio.Lock())

    def _cleanup_stale_sessions(self) -> None:
        """回收已结束且长期未访问的会话，防止全局注册表无限增长。"""
        now = time.monotonic()
        stale_ids = []
        for session_id, last_access in self.session_last_access.items():
            lock = self.session_locks.get(session_id)
            if lock is not None and lock.locked():
                continue
            shell = self.active_shells.get(session_id)
            if (
                    now - last_access > SESSION_IDLE_SECONDS
                    and shell is not None
                    and shell.process.returncode is not None
            ):
                stale_ids.append(session_id)
        for session_id in stale_ids:
            self.active_shells.pop(session_id, None)
            self.session_last_access.pop(session_id, None)
            lock = self.session_locks.get(session_id)
            # 当前请求可能仍持有锁；仅在锁空闲时一并回收，避免下一请求拿到两把锁。
            if lock is None or not lock.locked():
                self.session_locks.pop(session_id, None)

    def _touch_session(self, session_id: str) -> None:
        self.session_last_access[session_id] = time.monotonic()

    @staticmethod
    def _append_output(current: str, output: str) -> str:
        """保留最新输出，避免长期会话无限占用内存。"""
        combined = current + output
        if len(combined) <= MAX_OUTPUT_CHARS:
            return combined
        return combined[-MAX_OUTPUT_CHARS:]

    def _start_output_reader_task(
            self, session_id: str, process: asyncio.subprocess.Process
    ) -> None:
        """启动并持有输出读取任务，保证一个会话只消费当前进程的输出。"""
        previous_task = self.reader_tasks.pop(session_id, None)
        if previous_task and not previous_task.done():
            previous_task.cancel()

        task = asyncio.create_task(self._start_output_reader(session_id, process))
        self.reader_tasks[session_id] = task

        def remove_completed_task(completed_task: asyncio.Task) -> None:
            if self.reader_tasks.get(session_id) is completed_task:
                self.reader_tasks.pop(session_id, None)
            if not completed_task.cancelled() and completed_task.exception() is not None:
                logger.error(
                    f"会话 {session_id} 的输出读取器异常结束: {completed_task.exception()}"
                )

        task.add_done_callback(remove_completed_task)

    async def _stop_output_reader(self, session_id: str) -> None:
        """取消并等待输出读取器，避免旧进程向新命令的记录写入输出。"""
        task = self.reader_tasks.pop(session_id, None)
        if task is None or task.done():
            return
        task.cancel()
        try:
            await task
        except asyncio.CancelledError:
            pass

    @classmethod
    def _get_display_path(cls, path: str) -> str:
        """获取显示路径，将~替换成用户主目录"""
        # 1.使用程序获取跨平台下用户的主目录
        home_dir = os.path.expanduser("~")
        logger.debug(f"主目录: {home_dir}, 路径: {path}")

        # 2.判断传递进来的路径是否是主路径，如果是则替换成~
        if path.startswith(home_dir):
            return path.replace(home_dir, "~", 1)
        return path

    def _format_ps1(self, exec_dir: str) -> str:
        """格式化命令结构提示，增强交互体验，例如: root@myserver:/var/log $"""
        username = getpass.getuser()
        hostname = socket.gethostname()
        display_dir = self._get_display_path(exec_dir)
        return f"{username}@{hostname}:{display_dir} $"

    @classmethod
    async def _create_process(cls, exec_dir: str, command: str) -> asyncio.subprocess.Process:
        """根据传递的执行目录+命令创建一个asyncio管理的子进程"""
        # 1.ubuntu系统统一使用/bin/bash这个解释器
        logger.debug(f"在目录 {exec_dir} 下使用命令 {command} 创建一个子进程")
        shell_exec = "/bin/bash"

        # 3.创建一个系统级的子进程用来执行shell命令
        return await asyncio.create_subprocess_shell(
            command,  # 要执行的命令
            executable=shell_exec,  # 执行解释器
            cwd=exec_dir,
            start_new_session=True,  # 独立进程组：terminate/kill 需覆盖命令派生的子进程
            stdout=asyncio.subprocess.PIPE,  # 创建管道以捕获标准输出
            stderr=asyncio.subprocess.STDOUT,  # 将标准错误重定向到标准输出流
            stdin=asyncio.subprocess.PIPE,  # 创建管道以允许标准输入
            limit=1024 * 1024,  # 设置缓冲区大小并限制为1MB
        )

    async def _start_output_reader(self, session_id: str, process: asyncio.subprocess.Process) -> None:
        """启动协程以连续读取进程输出并将其存储到会话中"""
        # 1.ubuntu系统统一使用utf-8编码
        logger.debug(f"正在启用会话输出读取器: {session_id}")
        encoding = "utf-8"

        # 2.创建增量编码器（解决字符被切断的问题）
        decoder = codecs.getincrementaldecoder(encoding)(errors="replace")
        try:
            while True:
            # 3.判断子进程是否有标准输出管道
                if process.stdout:
                    try:
                    # 4.读取缓存区的数据，假设一次读取4096
                        buffer = await process.stdout.read(4096)
                        if not buffer:
                            break

                    # 5.使用编码器进行编码，同时设置final=False标识未结束
                        output = decoder.decode(buffer, final=False)

                    # 6.判断会话是否存在
                        shell = self.active_shells.get(session_id)
                        if shell and shell.process is process:
                        # 7.更新会话输出和控制台记录
                            shell.output = self._append_output(shell.output, output)
                            if shell.console_records:
                                record = shell.console_records[-1]
                                record.output = self._append_output(record.output, output)
                    except Exception as e:
                        logger.error(f"读取进程输出时错误: {str(e)}")
                        break
                else:
                    break
        except asyncio.CancelledError:
            logger.debug(f"会话 {session_id} 的输出读取器已取消")
            raise

        logger.debug(f"会话 {session_id} 的输出读取器已完成")

    @classmethod
    def _remove_ansi_escape_codes(cls, text: str) -> str:
        """从文本中删除ANSI转义字符"""
        ansi_escape = re.compile(r'\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])')
        return ansi_escape.sub("", text)

    @classmethod
    def create_session_id(cls) -> str:
        """创建会话id，使用uuid4生成唯一值"""
        session_id = str(uuid.uuid4())
        logger.info(f"创建一个新的Shell会话ID: {session_id}")
        return session_id

    def get_console_records(self, session_id: str) -> List[ConsoleRecord]:
        """从指定会话中获取控制台记录"""
        # 1.判断下传递的会话是否存在
        logger.debug(f"正在获取Shell会话的控制台记录: {session_id}")
        self._cleanup_stale_sessions()
        if session_id not in self.active_shells:
            logger.error(f"Shell会话不存在: {session_id}")
            raise NotFoundException(f"Shell会话不存在: {session_id}")

        # 2.获取原始的控制台记录列表
        self._touch_session(session_id)
        return self._get_console_records_unlocked(session_id)

    def _get_console_records_unlocked(self, session_id: str) -> List[ConsoleRecord]:
        """读取控制台记录的内部实现，调用方需确保会话状态不会并发变更。"""
        console_records = self.active_shells[session_id].console_records
        clean_console_records = []

        # 3.执行循环处理所有记录输出
        for console_record in console_records:
            clean_console_records.append(ConsoleRecord(
                ps1=console_record.ps1,
                command=console_record.command,
                output=self._remove_ansi_escape_codes(console_record.output),
            ))

        return clean_console_records


    @staticmethod
    def _terminate_process_tree(process: asyncio.subprocess.Process) -> None:
        """终止进程及其派生的整组子进程（start_new_session 使 bash 成为组长）。"""
        try:
            os.killpg(os.getpgid(process.pid), signal.SIGTERM)
        except (ProcessLookupError, PermissionError):
            process.terminate()

    @staticmethod
    def _kill_process_tree(process: asyncio.subprocess.Process) -> None:
        """强制杀死整组进程。"""
        try:
            os.killpg(os.getpgid(process.pid), signal.SIGKILL)
        except (ProcessLookupError, PermissionError):
            process.kill()

    async def wait_process(self, session_id: str, seconds: Optional[int] = None) -> ShellWaitResult:
        """按会话串行等待进程退出。"""
        self._cleanup_stale_sessions()
        async with self._get_session_lock(session_id):
            return await self._wait_process_unlocked(session_id, seconds)

    async def _wait_process_unlocked(self, session_id: str, seconds: Optional[int] = None) -> ShellWaitResult:
        """等待进程退出的内部实现，供已持有会话锁的流程调用。"""
        # 1.判断下传递的会话是否存在
        logger.debug(f"正在Shell会话中等待进程: {session_id}, 超时: {seconds}s")
        self._cleanup_stale_sessions()
        if session_id not in self.active_shells:
            logger.error(f"Shell会话不存在: {session_id}")
            raise NotFoundException(f"Shell会话不存在: {session_id}")

        # 2.获取会话和子进程
        shell = self.active_shells[session_id]
        self._touch_session(session_id)
        process = shell.process

        try:
            # 3.判断是否设置seconds
            seconds = 60 if seconds is None or seconds <= 0 else seconds
            await asyncio.wait_for(process.wait(), timeout=seconds)

            # 3.1 等待该会话的输出读取器消费完管道尾部（EOF），否则立刻读
            # console/output 可能截掉最后一截输出（读取器尚未轮转完）
            reader_task = self.reader_tasks.get(session_id)
            if reader_task and not reader_task.done():
                try:
                    await asyncio.wait_for(asyncio.shield(reader_task), timeout=2)
                except (asyncio.TimeoutError, asyncio.CancelledError, Exception):
                    pass

            # 4.记录日志并返回等待结果
            logger.info(f"进程已完成, 返回代码为: {process.returncode}")
            return ShellWaitResult(returncode=process.returncode)
        except asyncio.TimeoutError:
            # 记录日志并抛出BadRequest异常
            logger.warning(f"Shell会话进程等待超时: {seconds}s")
            raise BadRequestException(f"Shell会话进程等待超时: {seconds}s")
        except Exception as e:
            # 记录日志并抛出AppException
            logger.error(f"Shell会话进程等待过程出错: {str(e)}")
            raise AppException(f"Shell会话进程等待过程出错: {str(e)}")

    async def read_shell_output(self, session_id: str, console: bool = False) -> ShellReadResult:
        """按会话串行读取输出。"""
        self._cleanup_stale_sessions()
        async with self._get_session_lock(session_id):
            return self._read_shell_output_unlocked(session_id, console)

    def _read_shell_output_unlocked(self, session_id: str, console: bool = False) -> ShellReadResult:
        """读取输出的内部实现，供已持有会话锁的流程调用。"""
        # 1.判断下传递的会话是否存在
        logger.debug(f"查看Shell会话内容: {session_id}")
        self._cleanup_stale_sessions()
        if session_id not in self.active_shells:
            logger.error(f"Shell会话不存在: {session_id}")
            raise NotFoundException(f"Shell会话不存在: {session_id}")

        # 2.获取会话
        shell = self.active_shells[session_id]
        self._touch_session(session_id)

        # 3.获取原生输出并移除额外字符
        raw_output = shell.output
        clean_output = self._remove_ansi_escape_codes(raw_output)

        # 4.判断是否获取控制台记录
        if console:
            console_records = self._get_console_records_unlocked(session_id)
        else:
            console_records = []

        return ShellReadResult(
            session_id=session_id,
            output=clean_output,
            console_records=console_records,
        )

    async def exec_command(
            self,
            session_id: str,
            exec_dir: Optional[str],
            command: str,
    ) -> ShellExecuteResult:
        """按会话串行执行命令，避免并发请求相互替换进程。"""
        self._cleanup_stale_sessions()
        async with self._get_session_lock(session_id):
            return await self._exec_command_unlocked(session_id, exec_dir, command)

    async def _exec_command_unlocked(
            self,
            session_id: str,
            exec_dir: Optional[str],
            command: str,
    ) -> ShellExecuteResult:
        """传递会话id+执行目录+命令在沙箱中执行后返回"""
        # 1.记录日志并判断执行目录是否存在
        logger.info(f"正在会话 {session_id} 中执行命令: {command}")
        self._cleanup_stale_sessions()
        self._touch_session(session_id)
        if not exec_dir or exec_dir == "":
            exec_dir = os.path.expanduser("~")
        if exec_dir and not os.path.isdir(exec_dir):
            logger.error(f"执行目录不存在或不是目录: {exec_dir}")
            raise BadRequestException(f"执行目录不存在或不是目录: {exec_dir}")

        try:
            # 2.格式化生成ps1格式
            ps1 = self._format_ps1(exec_dir)

            # 3.判断当前Shell会话是否存在
            if session_id not in self.active_shells:
                # 4.创建一个新的进程
                logger.debug(f"创建一个新的Shell会话: {session_id}")
                process = await self._create_process(exec_dir, command)
                self.active_shells[session_id] = Shell(
                    process=process,
                    exec_dir=exec_dir,
                    output="",
                    console_records=[ConsoleRecord(ps1=ps1, command=command, output="")],
                )

                # 5.输出读取器独立运行，命令超时后仍持续收集输出。
                self._start_output_reader_task(session_id, process)
            else:
                # 6.该会话已存在则读取数据
                logger.debug(f"使用现有的Shell会话: {session_id}")
                shell = self.active_shells[session_id]
                old_process = shell.process

                # 7.判断旧进程是否还在运行，如果是则先停止旧进程在执行新命令
                if old_process.returncode is None:
                    logger.debug(f"正在终止会话中的上一个进程: {session_id}")
                    try:
                        # 8.结束旧进程组并优雅等待1s
                        self._terminate_process_tree(old_process)
                        await asyncio.wait_for(old_process.wait(), timeout=1)
                    except Exception as e:
                        # 9.结束旧进程出现错误并记录日志调用kill强制关闭进程
                        logger.warning(f"强制终止Shell会话中的进程 {session_id} 失败: {str(e)}")
                        self._kill_process_tree(old_process)
                        await old_process.wait()

                await self._stop_output_reader(session_id)

                # 10.关闭之后创建一个新的进程
                process = await self._create_process(exec_dir, command)

                # 11.更新会话信息
                shell.process = process
                shell.exec_dir = exec_dir
                shell.output = ""
                shell.console_records.append(ConsoleRecord(ps1=ps1, command=command, output=""))
                if len(shell.console_records) > MAX_CONSOLE_RECORDS:
                    shell.console_records = shell.console_records[-MAX_CONSOLE_RECORDS:]

                # 12.创建后台输出读取器，不等待进程结束。
                self._start_output_reader_task(session_id, process)

            try:

                # 13.尝试等待子进程执行(最多等待5s)
                logger.debug(f"正在等待会话中的进程完成: {session_id}")
                wait_result = await self._wait_process_unlocked(session_id, seconds=5)

                # 14.判断返回代码是否非空(已结束)则同步返回执行结果
                if wait_result.returncode is not None:
                    # 15.记录日志并查看结果
                    logger.debug(f"Shell会话进程已结束, 代码: {wait_result.returncode}")
                    view_result = self._read_shell_output_unlocked(session_id)

                    return ShellExecuteResult(
                        session_id=session_id,
                        command=command,
                        status="completed",
                        returncode=wait_result.returncode,
                        output=view_result.output,
                    )
            except BadRequestException as _:
                # 16.等待超时，记录日志不做额外处理让命令在后台继续运行
                logger.warning(f"进程在会话超时后仍在运行: {session_id}")
                pass
            except Exception as e:
                # 17.其他异常忽略并让程序继续进行
                logger.warning(f"等待进程时出现异常: {str(e)}")
                pass

            # 18.返回正在等待Shell执行结果
            return ShellExecuteResult(
                session_id=session_id,
                command=command,
                status="running",
            )
        except Exception as e:
            # 19.执行过程中出现异常并记录日志后返回自定义异常
            logger.error(f"命令执行失败: {str(e)}", exc_info=True)
            raise AppException(
                msg=f"命令执行失败: {str(e)}",
                data={"session_id": session_id, "command": command}
            )

    async def write_shell_input(
            self,
            session_id: str,
            input_text: str,
            press_enter: bool
    ) -> ShellWriteResult:
        """按会话串行写入输入，避免与命令切换并发执行。"""
        self._cleanup_stale_sessions()
        async with self._get_session_lock(session_id):
            return await self._write_shell_input_unlocked(session_id, input_text, press_enter)

    async def _write_shell_input_unlocked(
            self,
            session_id: str,
            input_text: str,
            press_enter: bool
    ) -> ShellWriteResult:
        """根据传递的数据向指定子进程写入数据"""
        # 1.判断下传递的会话是否存在
        logger.debug(f"写入Shell会话中的子进程: {session_id}, 是否按下回车键: {press_enter}")
        self._cleanup_stale_sessions()
        if session_id not in self.active_shells:
            logger.error(f"Shell会话不存在: {session_id}")
            raise NotFoundException(f"Shell会话不存在: {session_id}")

        # 2.获取会话和子进程
        shell = self.active_shells[session_id]
        process = shell.process
        self._touch_session(session_id)

        try:
            # 3.检查子进程是否结束
            if process.returncode is not None:
                logger.error(f"子进程已结束, 无法写入输入: {session_id}")
                raise BadRequestException("子进程已结束, 无法写入输入")

            # 4.ubuntu系统统一使用utf-8与\n
            encoding = "utf-8"
            line_ending = "\n"

            # 5.准备要发送的内容
            text_to_send = input_text
            if press_enter:
                text_to_send += line_ending

            # 6.将字符串编码为字节流(发送给进程使用)
            input_data = text_to_send.encode(encoding)

            # 7.记录日志/输出(直接使用原始字符串，不从input_data编码，避免编码不统一的情况)
            log_text = input_text + ("\n" if press_enter else "")
            shell.output = self._append_output(shell.output, log_text)
            if shell.console_records:
                record = shell.console_records[-1]
                record.output = self._append_output(record.output, log_text)

            # 8.向子进程写入数据
            process.stdin.write(input_data)
            await asyncio.wait_for(process.stdin.drain(), timeout=5)

            # 9.记录日志并返回写入结果
            logger.info("成功向子进程写入数据")
            return ShellWriteResult(status="success")
        except UnicodeError as e:
            # 10.捕获编码异常
            logger.error(f"编码错误: {str(e)}")
            raise AppException(f"编码错误: {str(e)}")
        except Exception as e:
            # 11.捕获通用异常
            logger.error(f"向子进程写入数据出错: {str(e)}")
            raise AppException(f"向子进程写入数据出错: {str(e)}")

    async def kill_process(self, session_id: str) -> ShellKillResult:
        """按会话串行终止进程。"""
        self._cleanup_stale_sessions()
        async with self._get_session_lock(session_id):
            return await self._kill_process_unlocked(session_id)

    async def _kill_process_unlocked(self, session_id: str) -> ShellKillResult:
        """根据传递的Shell会话id关闭对应进程"""
        # 1.判断下传递的会话是否存在
        logger.debug(f"正在终止会话中的进程: {session_id}")
        self._cleanup_stale_sessions()
        if session_id not in self.active_shells:
            logger.error(f"Shell会话不存在: {session_id}")
            raise NotFoundException(f"Shell会话不存在: {session_id}")

        # 2.获取会话和子进程
        shell = self.active_shells[session_id]
        process = shell.process
        self._touch_session(session_id)

        try:
            # 3.检查子进程是否还在运行
            if process.returncode is None:
                # 4.记录日志并尝试先优雅的关闭
                logger.info(f"尝试优雅终止进程: {session_id}")
                self._terminate_process_tree(process)

                try:
                    # 5.等待3秒时间
                    await asyncio.wait_for(process.wait(), timeout=3)
                except asyncio.TimeoutError as _:
                    # 6.优雅关闭失败，则强制关闭
                    logger.warning(f"尝试强制关闭进程: {session_id}")
                    self._kill_process_tree(process)
                    await process.wait()

                # 7.记录日志并返回关闭结果
                logger.info(f"进程已终止, 返回代码为: {process.returncode}")
                await self._stop_output_reader(session_id)
                return ShellKillResult(status="terminated", returncode=process.returncode)
            else:
                # 8.进程已结束无需重复关闭
                logger.info(f"进程已终止, 返回代码为: {process.returncode}")
                await self._stop_output_reader(session_id)
                return ShellKillResult(status="already_terminated", returncode=process.returncode)
        except Exception as e:
            # 9.记录日志并抛出异常
            logger.error(f"关闭进程失败: {str(e)}", exc_info=True)
            raise AppException(f"关闭进程失败: {str(e)}")
