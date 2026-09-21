#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
@Time    : 2025/05/13 09:00
@Author  : thezehui@gmail.com
@File    : browser.py
"""
from typing import Optional

from pydantic import BaseModel, Field


class BrowserNavigateRequest(BaseModel):
    """导航请求结构体"""
    url: str = Field(..., min_length=1, max_length=2048, description="目标页面地址")


class BrowserTargetRequest(BaseModel):
    """元素定位请求结构体：编号、选择器、坐标三选一"""
    index: Optional[int] = Field(default=None, ge=0, description="元素编号，来自 snapshot 结果")
    selector: Optional[str] = Field(default=None, max_length=512, description="CSS 选择器")
    x: Optional[float] = Field(default=None, description="点击/输入的 X 坐标")
    y: Optional[float] = Field(default=None, description="点击/输入的 Y 坐标")


class BrowserInputRequest(BrowserTargetRequest):
    """输入文本请求结构体"""
    text: str = Field(..., max_length=64 * 1024, description="要输入的文本")
    press_enter: bool = Field(default=False, description="输入完成后是否按下回车键")


class BrowserScreenshotRequest(BaseModel):
    """截图请求结构体"""
    full_page: bool = Field(default=False, description="是否截取整页（默认仅当前视口）")


class BrowserPressKeyRequest(BaseModel):
    """按键请求结构体"""
    key: str = Field(..., min_length=1, max_length=64, description="按键标识，如 Enter/Escape/Tab")


class BrowserScrollRequest(BaseModel):
    """滚动请求结构体"""
    direction: str = Field(default="down", pattern="^(up|down)$", description="滚动方向")
    to_end: bool = Field(default=False, description="是否直达顶部/底部")


class BrowserConsoleExecRequest(BaseModel):
    """控制台执行 JavaScript 请求结构体"""
    javascript: str = Field(..., min_length=1, max_length=64 * 1024, description="要执行的 JavaScript 代码")


class BrowserConsoleViewRequest(BaseModel):
    """查看控制台日志请求结构体"""
    max_lines: Optional[int] = Field(default=None, ge=1, le=1000, description="返回最近多少行日志")