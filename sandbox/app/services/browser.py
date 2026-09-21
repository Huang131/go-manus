#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
@Time    : 2025/05/13 09:10
@Author  : thezehui@gmail.com
@File    : browser.py
"""
import asyncio
import json
import logging
import os
import uuid
from typing import Any, Dict, Optional

from app.interfaces.errors.exceptions import AppException, BadRequestException
from app.models.browser import (
    BrowserConsoleExecResult,
    BrowserConsoleViewResult,
    BrowserPageResult,
    BrowserScreenshotResult,
    BrowserScrollResult,
    BrowserSnapshotResult,
)

logger = logging.getLogger(__name__)

# Node CLI 与本模块同目录分发，避免依赖工作目录
BROWSER_CLI_PATH = os.path.join(os.path.dirname(os.path.abspath(__file__)), "browser_cli.js")
NODE_BINARY = "node"
# 截图落盘目录，与 Node CLI 默认控制台日志目录同源
SCREENSHOT_DIR = "/tmp/browser/screenshots"

# 动作超时：导航和截图最慢，控制台日志读取不连浏览器、只需本地 IO
ACTION_TIMEOUT_SECONDS = 60
NAVIGATE_TIMEOUT_SECONDS = 90
SCREENSHOT_TIMEOUT_SECONDS = 90
CONSOLE_VIEW_TIMEOUT_SECONDS = 15

# 环境类失败（CDP 连不上、依赖缺失）返回 503，业务动作失败返回 400，
# 便于上层把两类错误分开处理（环境失败上报服务异常，动作失败回灌模型自行修正）
ENV_ERROR_STATUS = 503
TIMEOUT_ERROR_STATUS = 504


class BrowserService:
    """浏览器自动化服务。

    一个动作启动一个 Node 子进程，进程内通过 CDP 复用 Supervisor 托管的 Chrome：
    1. 动作之间不共享进程状态，避免长进程状态漂移；
    2. Chrome 由 Supervisor 守护，子进程退出不会关闭浏览器；
    3. 同一沙箱内动作串行化，避免并发动作互相干扰页面状态。
    """

    def __init__(self) -> None:
        self._action_lock = asyncio.Lock()

    async def navigate(self, url: str) -> BrowserPageResult:
        """导航到指定地址并返回页面摘要"""
        data = await self._run("navigate", {"url": url}, NAVIGATE_TIMEOUT_SECONDS)
        return BrowserPageResult(**data)

    async def snapshot(self) -> BrowserSnapshotResult:
        """获取带编号的可交互元素列表，编号可直接用于 click/input"""
        data = await self._run("snapshot", {}, ACTION_TIMEOUT_SECONDS)
        return BrowserSnapshotResult(**data)

    async def screenshot(self, full_page: bool = False) -> BrowserScreenshotResult:
        """截图并写入沙箱文件，返回文件路径供上层下载二进制"""
        os.makedirs(SCREENSHOT_DIR, exist_ok=True)
        filepath = os.path.join(SCREENSHOT_DIR, f"{uuid.uuid4().hex}.png")
        data = await self._run(
            "screenshot",
            {"path": filepath, "full_page": full_page},
            SCREENSHOT_TIMEOUT_SECONDS,
        )
        return BrowserScreenshotResult(**data)

    async def click(self, index: Optional[int] = None, selector: Optional[str] = None,
                    x: Optional[float] = None, y: Optional[float] = None) -> BrowserPageResult:
        """点击元素，支持编号、CSS 选择器、坐标三种定位方式"""
        data = await self._run(
            "click",
            {"index": index, "selector": selector, "x": x, "y": y},
            ACTION_TIMEOUT_SECONDS,
        )
        return BrowserPageResult(**data)

    async def input(self, text: str, press_enter: bool = False, index: Optional[int] = None,
                    selector: Optional[str] = None, x: Optional[float] = None,
                    y: Optional[float] = None) -> BrowserPageResult:
        """向输入框写入文本，支持编号、CSS 选择器、坐标三种定位方式"""
        data = await self._run(
            "input",
            {
                "text": text,
                "press_enter": press_enter,
                "index": index,
                "selector": selector,
                "x": x,
                "y": y,
            },
            ACTION_TIMEOUT_SECONDS,
        )
        return BrowserPageResult(**data)

    async def press_key(self, key: str) -> BrowserPageResult:
        """模拟按键，如 Enter/Escape/Tab/ArrowDown"""
        data = await self._run("press_key", {"key": key}, ACTION_TIMEOUT_SECONDS)
        return BrowserPageResult(**data)

    async def scroll(self, direction: str = "down", to_end: bool = False) -> BrowserScrollResult:
        """按方向滚动一屏，或直达页面顶部/底部"""
        data = await self._run(
            "scroll",
            {"direction": direction, "to_end": to_end},
            ACTION_TIMEOUT_SECONDS,
        )
        return BrowserScrollResult(**data)

    async def console_exec(self, javascript: str) -> BrowserConsoleExecResult:
        """在页面上下文执行 JavaScript"""
        data = await self._run("console_exec", {"javascript": javascript}, ACTION_TIMEOUT_SECONDS)
        return BrowserConsoleExecResult(**data)

    async def console_view(self, max_lines: Optional[int] = None) -> BrowserConsoleViewResult:
        """读取历史控制台日志尾部（不连接浏览器）"""
        payload: Dict[str, Any] = {"max_lines": max_lines}
        data = await self._run("console_view", payload, CONSOLE_VIEW_TIMEOUT_SECONDS)
        return BrowserConsoleViewResult(**data)

    async def _run(self, op: str, payload: Dict[str, Any], timeout: int) -> Dict[str, Any]:
        """串行执行单个浏览器动作，返回 Node CLI 的 JSON 结果。"""
        params: Dict[str, Any] = {"op": op}
        # 只丢弃 None：False/0 是有效语义（如 press_enter=false），必须显式传给 CLI
        params.update({key: value for key, value in payload.items() if value is not None})
        async with self._action_lock:
            return await self._run_process(op, params, timeout)

    async def _run_process(self, op: str, params: Dict[str, Any], timeout: int) -> Dict[str, Any]:
        """启动 Node 子进程执行动作，并把失败统一转换为沙箱异常。"""
        if not os.path.isfile(BROWSER_CLI_PATH):
            raise AppException(f"浏览器 CLI 缺失: {BROWSER_CLI_PATH}", status_code=ENV_ERROR_STATUS)

        process = await asyncio.create_subprocess_exec(
            NODE_BINARY,
            BROWSER_CLI_PATH,
            json.dumps(params, ensure_ascii=False),
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )

        try:
            stdout, stderr = await asyncio.wait_for(process.communicate(), timeout=timeout)
        except asyncio.TimeoutError:
            await self._terminate(process)
            logger.error(f"浏览器动作超时: op={op}, timeout={timeout}s")
            raise AppException(
                f"浏览器动作 {op} 执行超时（{timeout}秒），页面可能仍在加载",
                status_code=TIMEOUT_ERROR_STATUS,
            )

        return self._parse_result(op, stdout, stderr)

    @staticmethod
    def _parse_result(op: str, stdout: bytes, stderr: bytes) -> Dict[str, Any]:
        """解析 CLI 输出：最后一行是结果 JSON，其余输出仅用于诊断。"""
        stderr_text = stderr.decode("utf-8", errors="replace").strip()
        lines = [line for line in stdout.decode("utf-8", errors="replace").splitlines() if line.strip()]
        if not lines:
            detail = stderr_text or "无输出"
            logger.error(f"浏览器动作无输出: op={op}, detail={detail}")
            raise AppException(f"浏览器动作 {op} 未返回结果: {detail}", status_code=ENV_ERROR_STATUS)

        try:
            result = json.loads(lines[-1])
        except json.JSONDecodeError:
            logger.error(f"浏览器动作输出无法解析: op={op}, output={lines[-1][:500]}")
            raise AppException(
                f"浏览器动作 {op} 返回了非法结果，请检查沙箱浏览器环境",
                status_code=ENV_ERROR_STATUS,
            )

        if not isinstance(result, dict) or "ok" not in result:
            logger.error(f"浏览器动作结果结构异常: op={op}, result={str(result)[:500]}")
            raise AppException(
                f"浏览器动作 {op} 返回了非法结果，请检查沙箱浏览器环境",
                status_code=ENV_ERROR_STATUS,
            )

        if result.get("ok"):
            return result

        message = str(result.get("error") or "浏览器动作执行失败")
        if result.get("kind") == "action":
            # 元素找不到、参数缺失属于业务动作失败：回 400 让模型按最新快照自行修正
            raise BadRequestException(message)
        logger.error(f"浏览器动作环境失败: op={op}, error={message}")
        raise AppException(message, status_code=ENV_ERROR_STATUS)

    @staticmethod
    async def _terminate(process: asyncio.subprocess.Process) -> None:
        """超时后回收子进程，避免残留 Node 进程继续占用 CDP 连接。"""
        if process.returncode is not None:
            return
        try:
            process.kill()
        except ProcessLookupError:
            return
        try:
            await asyncio.wait_for(process.wait(), timeout=5)
        except asyncio.TimeoutError:
            logger.error("终止浏览器子进程超时")