#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
@Time    : 2025/05/26 1:52
@Author  : thezehui@gmail.com
@File    : supervisor.py
"""
from typing import Optional

from pydantic import BaseModel, Field


class TimeoutRequest(BaseModel):
    """激活超时销毁请求"""
    minutes: Optional[int] = Field(default=None, ge=1, le=7 * 24 * 60, description="分钟数")
