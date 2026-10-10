from functools import lru_cache
from typing import Literal

from pydantic_settings import BaseSettings, SettingsConfigDict

# 日志等级取值范围，与 logging 模块的等级名一一对应。
# 用 Literal 约束后，配置写错（如小写 info、带空格）会在配置解析阶段报错，
# 而不是等到 getattr(logging, ...) 时抛 AttributeError。
LogLevel = Literal["DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"]


class Settings(BaseSettings):
    """沙箱API服务基础配置信息"""
    # 默认开 DEBUG 会把容器日志刷满，真正需要排查时再按环境变量临时打开。
    log_level: LogLevel = "INFO"
    # 服务超时时间单位为分钟；允许为 None 表示不预置销毁定时器。
    # 默认 60，保持原有行为：未显式配置时，沙箱空闲 60 分钟后自动关闭。
    server_timeout_minutes: int | None = 60

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        # 只忽略 .env 里的未知字段，容器环境变量不受影响（pydantic-settings 仅按字段名
        # 从 environ 挑匹配项，不会整体灌入），所以这里不会放过 os.environ 中的拼写错误。
        extra="ignore",
    )


@lru_cache()
def get_settings() -> Settings:
    return Settings()
