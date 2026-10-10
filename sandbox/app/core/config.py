#!/usr/bin/env python
# -*- coding: utf-8 -*-
from functools import lru_cache
from typing import Literal

from pydantic_settings import BaseSettings, SettingsConfigDict

# 日志等级取值范围，与 logging 模块的等级名一一对应。
# 用 Literal 约束后，配置写错（如小写 info、带空格）会在配置解析阶段报错，
# 而不是等到 getattr(logging, ...) 时抛 AttributeError。
LogLevel = Literal["DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"]


class Settings(BaseSettings):
    """沙箱API服务基础配置信息"""
    log_level: LogLevel = "INFO"  # 日志等级
    server_timeout_minutes: int = 60  # 服务超时时间单位为分钟

    # 使用pydantic v2提供的写法完成环境变量信息的声明
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )


@lru_cache()
def get_settings() -> Settings:
    return Settings()
