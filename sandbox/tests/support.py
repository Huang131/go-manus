"""测试共享辅助：自动清理的临时文件、条件等待、进程存活判定。

现有用例均基于 unittest.IsolatedAsyncioTestCase，pytest fixture 无法注入到
unittest 测试方法，因此以 mixin / 异步函数形式提供，避免为同一件事重复造轮子。
"""
import asyncio
import os
import tempfile
from pathlib import Path


class TempFileMixin:
    """为 unittest 用例提供自动清理的临时文件。

    pytest fixture 到不了 unittest.TestCase 的方法签名，这里用 addCleanup
    保证用例结束（含失败）后文件被删除。
    """

    def write_temp_file(self, content: str = "", *, encoding: str = "utf-8", suffix: str = "") -> str:
        with tempfile.NamedTemporaryFile(
                mode="w", encoding=encoding, suffix=suffix, delete=False
        ) as file:
            file.write(content)
            path = file.name
        self.addCleanup(Path(path).unlink, missing_ok=True)
        return path


async def wait_until(
        predicate,
        *,
        timeout: float = 2.0,
        interval: float = 0.01,
        msg: str = "条件等待超时",
) -> None:
    """等待 predicate 变为真，超时抛 AssertionError。

    等待的是状态变化本身，而不是固定时长：慢机器上不会因 sleep 不足而偶发失败，
    快机器上也不会白等。用于替代 asyncio.sleep 做同步。
    """
    loop = asyncio.get_running_loop()
    deadline = loop.time() + timeout
    while not predicate():
        if loop.time() >= deadline:
            raise AssertionError(f"{msg}（{timeout}s）")
        await asyncio.sleep(interval)


def process_gone(pid: int) -> bool:
    """判断进程是否已退出（含僵尸态），跨平台可用。

    主判据是 os.kill(pid, 0)：进程消失抛 ProcessLookupError 即已退出。
    Linux 容器里子进程被 PID 1 收尸前会短暂停留在僵尸态，此时 kill(pid, 0)
    仍成功，故在 /proc 存在时额外识别 stat 的 Z 状态；/proc 不存在（如 macOS）
    则跳过该补充判据。
    """
    try:
        os.kill(pid, 0)
    except ProcessLookupError:
        return True
    except PermissionError:
        return False
    stat_path = Path(f"/proc/{pid}/stat")
    if stat_path.exists():
        try:
            return stat_path.read_text(encoding="utf-8").split()[2] == "Z"
        except (OSError, IndexError):
            return False
    return False
