#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
@Time    : 2025/05/13 09:00
@Author  : thezehui@gmail.com
@File    : browser.py
"""
from typing import List, Optional

from pydantic import BaseModel, Field


class BrowserElement(BaseModel):
    """页面可交互元素。

    index 是快照中的稳定编号，click/input 直接复用该编号定位元素，
    避免模型凭 HTML 猜测"第几个 a 标签"。
    """
    index: int = Field(..., description="元素编号，用于 click/input 定位")
    tag: str = Field(..., description="标签名，如 a/button/input")
    type: Optional[str] = Field(default=None, description="input 的 type 属性")
    text: Optional[str] = Field(default=None, description="元素可见文本或占位提示")
    href: Optional[str] = Field(default=None, description="链接地址")
    value: Optional[str] = Field(default=None, description="输入框当前值")
    checked: Optional[bool] = Field(default=None, description="是否选中")
    disabled: Optional[bool] = Field(default=None, description="是否禁用")


class BrowserScrollPosition(BaseModel):
    """页面滚动位置，用于判断是否还需要继续滚动"""
    y: int = Field(default=0, description="当前纵向滚动距离")
    height: int = Field(default=0, description="页面总高度")
    viewport: int = Field(default=0, description="视口高度")


class BrowserPageResult(BaseModel):
    """页面动作通用结果：URL、标题与动作窗口内的控制台日志"""
    url: str = Field(default="", description="动作执行后的页面地址")
    title: str = Field(default="", description="页面标题")
    warnings: List[str] = Field(default_factory=list, description="非致命告警，如导航等待超时")
    logs: List[str] = Field(default_factory=list, description="动作执行窗口内的控制台日志")


class BrowserSnapshotResult(BrowserPageResult):
    """页面快照结果"""
    elements: List[BrowserElement] = Field(default_factory=list, description="可交互元素列表")
    scroll: Optional[BrowserScrollPosition] = Field(default=None, description="滚动位置")


class BrowserScrollResult(BrowserPageResult):
    """滚动结果"""
    scroll: Optional[BrowserScrollPosition] = Field(default=None, description="滚动位置")


class BrowserScreenshotResult(BrowserPageResult):
    """截图结果。

    截图以 PNG 文件落在沙箱内，上层通过文件下载端点取二进制，
    不走 JSON 响应体，避免 base64 撑爆响应和上下文。
    """
    path: str = Field(default="", description="截图文件在沙箱内的绝对路径")
    bytes: int = Field(default=0, description="截图文件字节数")


class BrowserConsoleExecResult(BaseModel):
    """控制台执行 JavaScript 结果"""
    result: str = Field(default="null", description="执行返回值的 JSON 字符串")
    truncated: bool = Field(default=False, description="返回值是否被截断")
    logs: List[str] = Field(default_factory=list, description="执行窗口内的控制台日志")


class BrowserConsoleViewResult(BaseModel):
    """控制台历史日志结果"""
    lines: List[str] = Field(default_factory=list, description="最近的日志行")
    total: int = Field(default=0, description="日志总行数")