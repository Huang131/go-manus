"""Browser CLI 纯 IO 契约回归测试。

守护「单行 JSON + 完整刷盘 + 正确退出码」这个不随 Chrome 变化的契约，
全部通过 console_view（唯一不连浏览器的动作）与 BrowserService 的预算切分
纯函数来验证，因此无需 Chrome 即可在本机/容器内运行。
"""
import asyncio
import json
import os
import tempfile
import unittest
from pathlib import Path

from app.services.browser import BROWSER_CLI_PATH, NODE_BINARY

MAX_VIEW_BYTES_DEFAULT = 20000


class BrowserCLIContractTests(unittest.IsolatedAsyncioTestCase):
    async def _run_console_view(self, max_lines=None, console_content="", extra_env=None):
        """以子进程调用 node browser_cli.js 执行 console_view，返回 (returncode, stdout, stderr)。"""
        env = dict(os.environ)
        env.pop("SANDBOX_MAX_VIEW_BYTES", None)
        if extra_env:
            env.update(extra_env)

        with tempfile.NamedTemporaryFile(mode="w", encoding="utf-8", delete=False) as log:
            log.write(console_content)
            log_path = log.name
        self.addCleanup(Path(log_path).unlink, missing_ok=True)
        env["SANDBOX_CONSOLE_LOG"] = log_path

        argv = [NODE_BINARY, BROWSER_CLI_PATH, json.dumps({"op": "console_view", "max_lines": max_lines})]
        process = await asyncio.create_subprocess_exec(
            *argv,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
            env=env,
        )
        stdout, stderr = await process.communicate()
        return process.returncode, stdout.decode("utf-8", errors="replace"), stderr.decode("utf-8", errors="replace")

    async def test_empty_log_returns_single_line_json_success(self):
        code, stdout, stderr = await self._run_console_view()
        self.assertEqual(code, 0)
        lines = [line for line in stdout.splitlines() if line.strip()]
        self.assertEqual(len(lines), 1, f"输出必须是单行 JSON: {stdout!r}")
        payload = json.loads(lines[0])
        self.assertTrue(payload["ok"])
        self.assertEqual(payload["total"], 0)
        self.assertEqual(payload["lines"], [])

    async def test_invalid_input_returns_action_error_with_nonzero_exit(self):
        argv = [NODE_BINARY, BROWSER_CLI_PATH, "not-json"]
        process = await asyncio.create_subprocess_exec(
            *argv,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout, _ = await process.communicate()
        self.assertEqual(process.returncode, 1)
        lines = [line for line in stdout.decode("utf-8", errors="replace").splitlines() if line.strip()]
        self.assertEqual(len(lines), 1)
        payload = json.loads(lines[0])
        self.assertFalse(payload["ok"])
        self.assertEqual(payload["kind"], "action")

    async def test_missing_op_returns_action_error(self):
        argv = [NODE_BINARY, BROWSER_CLI_PATH, json.dumps({})]
        process = await asyncio.create_subprocess_exec(
            *argv,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout, _ = await process.communicate()
        self.assertEqual(process.returncode, 1)
        payload = json.loads(stdout.decode("utf-8", errors="replace").strip())
        self.assertFalse(payload["ok"])
        self.assertEqual(payload["kind"], "action")

    async def test_console_view_truncates_by_byte_budget(self):
        # 25 行 × 1000 字节 = 25000 字节，超过默认预算 20000，应触发截断
        content = "\n".join("x" * 1000 for _ in range(25)) + "\n"
        code, stdout, _ = await self._run_console_view(max_lines=1000, console_content=content)
        self.assertEqual(code, 0)
        payload = json.loads(stdout.strip())
        self.assertTrue(payload["truncated"])
        self.assertLessEqual(len("\n".join(payload["lines"]).encode("utf-8")), MAX_VIEW_BYTES_DEFAULT)

    async def test_large_output_is_not_truncated_at_pipe_layer(self):
        # 构造 >64KB 管道缓冲的输出，验证 writeResult「刷盘回调再退出」：
        # 若误用 process.exit() 直接退出，输出会在刷盘前被截断，JSON 解析必然失败。
        content = "\n".join("y" * 100 for _ in range(2000)) + "\n"
        code, stdout, _ = await self._run_console_view(
            max_lines=2000,
            console_content=content,
            extra_env={"SANDBOX_MAX_VIEW_BYTES": "300000"},
        )
        self.assertEqual(code, 0)
        payload = json.loads(stdout.strip())
        self.assertFalse(payload["truncated"])
        self.assertEqual(payload["total"], 2000)

    async def test_budget_split_never_exceeds_total_budget(self):
        from app.services.browser import BrowserService

        cases = [
            (90, 100_000),  # 导航动作 90s，总预算 100s
            (60, 60_000),   # 动作超时与预算相等
            (60, 30_000),   # 预算小于动作超时：锁等待为 0，动作只能吃到剩余预算
            (15, 20_000),   # console_view
        ]
        for action_timeout, budget_ms in cases:
            budget_sec = budget_ms / 1000.0
            lock_wait = BrowserService._lock_wait_seconds(action_timeout, budget_ms)
            self.assertGreaterEqual(lock_wait, 0.0)
            for ratio in (0.0, 0.25, 0.5, 1.0):
                elapsed = lock_wait * ratio
                exec_timeout = BrowserService._exec_timeout_seconds(action_timeout, budget_ms, elapsed)
                self.assertGreaterEqual(exec_timeout, 0.0)
                self.assertLessEqual(
                    elapsed + exec_timeout,
                    budget_sec + 1e-9,
                    f"action={action_timeout}, budget={budget_ms}, elapsed={elapsed}",
                )


if __name__ == "__main__":
    unittest.main()