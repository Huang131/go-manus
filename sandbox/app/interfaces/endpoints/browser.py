#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
@Time    : 2025/05/13 09:20
@Author  : thezehui@gmail.com
@File    : browser.py
"""
from typing import Optional

from fastapi import APIRouter, Depends, Header

from app.interfaces.schemas.base import Response
from app.interfaces.schemas.browser import (
    BrowserConsoleExecRequest,
    BrowserConsoleViewRequest,
    BrowserInputRequest,
    BrowserNavigateRequest,
    BrowserPressKeyRequest,
    BrowserScreenshotRequest,
    BrowserScrollRequest,
    BrowserTargetRequest,
)
from app.interfaces.service_dependencies import get_browser_service
from app.models.browser import (
    BrowserConsoleExecResult,
    BrowserConsoleViewResult,
    BrowserPageResult,
    BrowserScreenshotResult,
    BrowserScrollResult,
    BrowserSnapshotResult,
)
from app.services.browser import BrowserService

router = APIRouter(prefix="/browser", tags=["浏览器模块"])

# Deadline 传播：Go 侧把剩余预算（毫秒）放在该请求头下发，
# 沙箱据此把"锁等待 + 动作执行"切成两段预算，各自耗尽前主动返回语义化状态码。
TIMEOUT_HEADER = "X-Sandbox-Timeout-Ms"


def _budget_ms(raw: Optional[str]) -> Optional[int]:
    """把请求头值解析为剩余预算毫秒数，非法或非正值一律回落 None（走无预算兜底）。"""
    if raw is None:
        return None
    try:
        value = int(raw)
    except (TypeError, ValueError):
        return None
    return value if value > 0 else None


@router.post(
    path="/navigate",
    response_model=Response[BrowserPageResult],
)
async def navigate(
        request: BrowserNavigateRequest,
        browser_service: BrowserService = Depends(get_browser_service),
        raw_budget: Optional[str] = Header(default=None, alias=TIMEOUT_HEADER),
) -> Response[BrowserPageResult]:
    """导航到指定地址，返回页面地址与标题"""
    result = await browser_service.navigate(request.url, budget_ms=_budget_ms(raw_budget))
    return Response.success(msg=f"已导航至: {result.url}", data=result)


@router.post(
    path="/snapshot",
    response_model=Response[BrowserSnapshotResult],
)
async def snapshot(
        browser_service: BrowserService = Depends(get_browser_service),
        raw_budget: Optional[str] = Header(default=None, alias=TIMEOUT_HEADER),
) -> Response[BrowserSnapshotResult]:
    """获取带编号的可交互元素列表，编号可直接用于 click/input"""
    result = await browser_service.snapshot(budget_ms=_budget_ms(raw_budget))
    return Response.success(msg=f"获取页面快照成功, 共{len(result.elements)}个可交互元素", data=result)


@router.post(
    path="/screenshot",
    response_model=Response[BrowserScreenshotResult],
)
async def screenshot(
        request: BrowserScreenshotRequest,
        browser_service: BrowserService = Depends(get_browser_service),
        raw_budget: Optional[str] = Header(default=None, alias=TIMEOUT_HEADER),
) -> Response[BrowserScreenshotResult]:
    """对当前页面截图，PNG 落在沙箱内，上层通过文件下载端点取二进制"""
    result = await browser_service.screenshot(request.full_page, budget_ms=_budget_ms(raw_budget))
    return Response.success(msg=f"截图成功, 文件大小{result.bytes}字节", data=result)


@router.post(
    path="/click",
    response_model=Response[BrowserPageResult],
)
async def click(
        request: BrowserTargetRequest,
        browser_service: BrowserService = Depends(get_browser_service),
        raw_budget: Optional[str] = Header(default=None, alias=TIMEOUT_HEADER),
) -> Response[BrowserPageResult]:
    """点击元素，index 需来自最近一次 snapshot 结果"""
    result = await browser_service.click(
        index=request.index,
        selector=request.selector,
        x=request.x,
        y=request.y,
        budget_ms=_budget_ms(raw_budget),
    )
    return Response.success(msg=f"点击完成, 当前页面: {result.url}", data=result)


@router.post(
    path="/input",
    response_model=Response[BrowserPageResult],
)
async def input_text(
        request: BrowserInputRequest,
        browser_service: BrowserService = Depends(get_browser_service),
        raw_budget: Optional[str] = Header(default=None, alias=TIMEOUT_HEADER),
) -> Response[BrowserPageResult]:
    """向输入框写入文本，index 需来自最近一次 snapshot 结果"""
    result = await browser_service.input(
        text=request.text,
        press_enter=request.press_enter,
        index=request.index,
        selector=request.selector,
        x=request.x,
        y=request.y,
        budget_ms=_budget_ms(raw_budget),
    )
    return Response.success(msg=f"输入完成, 当前页面: {result.url}", data=result)


@router.post(
    path="/press-key",
    response_model=Response[BrowserPageResult],
)
async def press_key(
        request: BrowserPressKeyRequest,
        browser_service: BrowserService = Depends(get_browser_service),
        raw_budget: Optional[str] = Header(default=None, alias=TIMEOUT_HEADER),
) -> Response[BrowserPageResult]:
    """模拟按键，如 Enter/Escape/Tab/ArrowDown"""
    result = await browser_service.press_key(request.key, budget_ms=_budget_ms(raw_budget))
    return Response.success(msg=f"按键{request.key}已发送", data=result)


@router.post(
    path="/scroll",
    response_model=Response[BrowserScrollResult],
)
async def scroll(
        request: BrowserScrollRequest,
        browser_service: BrowserService = Depends(get_browser_service),
        raw_budget: Optional[str] = Header(default=None, alias=TIMEOUT_HEADER),
) -> Response[BrowserScrollResult]:
    """按方向滚动页面，to_end 为真时直达顶部/底部"""
    result = await browser_service.scroll(request.direction, request.to_end, budget_ms=_budget_ms(raw_budget))
    return Response.success(msg=f"滚动完成, 当前纵向位置: {result.scroll.y if result.scroll else 0}", data=result)


@router.post(
    path="/console-exec",
    response_model=Response[BrowserConsoleExecResult],
)
async def console_exec(
        request: BrowserConsoleExecRequest,
        browser_service: BrowserService = Depends(get_browser_service),
        raw_budget: Optional[str] = Header(default=None, alias=TIMEOUT_HEADER),
) -> Response[BrowserConsoleExecResult]:
    """在页面上下文执行 JavaScript 并返回结果"""
    result = await browser_service.console_exec(request.javascript, budget_ms=_budget_ms(raw_budget))
    return Response.success(
        msg="JavaScript执行完成" + ("（返回值已截断）" if result.truncated else ""),
        data=result,
    )


@router.post(
    path="/console-view",
    response_model=Response[BrowserConsoleViewResult],
)
async def console_view(
        request: BrowserConsoleViewRequest,
        browser_service: BrowserService = Depends(get_browser_service),
        raw_budget: Optional[str] = Header(default=None, alias=TIMEOUT_HEADER),
) -> Response[BrowserConsoleViewResult]:
    """读取历史控制台日志尾部，用于回溯页面报错"""
    result = await browser_service.console_view(request.max_lines, budget_ms=_budget_ms(raw_budget))
    return Response.success(msg=f"控制台日志共{result.total}行, 返回最近{len(result.lines)}行", data=result)