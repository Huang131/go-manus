import asyncio
import os
import tempfile
import time
import unittest
from pathlib import Path
from unittest.mock import AsyncMock, patch

from app.models.file import FileReadResult
from app.services.file import FileService
from app.services.shell import ShellService


class ServiceRegressionTests(unittest.IsolatedAsyncioTestCase):
    async def test_long_shell_command_returns_running_after_wait_window(self):
        service = ShellService()
        started = time.monotonic()
        result = await service.exec_command("test-session", tempfile.gettempdir(), "sleep 10")
        elapsed = time.monotonic() - started
        self.assertEqual(result.status, "running")
        self.assertLess(elapsed, 7)
        kill_result = await service.kill_process("test-session")
        self.assertIsNotNone(kill_result.returncode)

    async def test_sudo_read_waits_for_process_and_supports_special_path(self):
        process = AsyncMock()
        process.returncode = 0
        process.communicate.return_value = (b"hello", b"")
        with patch("app.services.file.asyncio.create_subprocess_exec", return_value=process) as create:
            result = await FileService.read_file("/tmp/path with 'quote'.txt", sudo=True)
        self.assertIsInstance(result, FileReadResult)
        create.assert_awaited_once_with(
            "sudo", "cat", "/tmp/path with 'quote'.txt",
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        process.communicate.assert_awaited_once()

    async def test_relative_path_write_does_not_create_empty_directory(self):
        with tempfile.TemporaryDirectory() as directory:
            previous_directory = Path.cwd()
            try:
                os.chdir(directory)
                with patch("app.services.file.os.makedirs") as makedirs:
                    result = await FileService.write_file("work.txt", "content")
            finally:
                os.chdir(previous_directory)
            self.assertEqual(result.bytes_written, len("content"))
            makedirs.assert_not_called()

    async def test_sudo_write_passes_content_through_stdin(self):
        process = AsyncMock()
        process.returncode = 0
        process.communicate.return_value = (b"", b"")
        with patch("app.services.file.asyncio.create_subprocess_exec", return_value=process) as create:
            result = await FileService.write_file("/tmp/path with 'quote'.txt", "hello", sudo=True)
        self.assertEqual(result.bytes_written, len("hello"))
        create.assert_awaited_once_with(
            "sudo", "tee", "/tmp/path with 'quote'.txt",
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        process.communicate.assert_awaited_once_with(b"hello")

    async def test_search_matches_content_beyond_line_start(self):
        with tempfile.NamedTemporaryFile(mode="w", encoding="utf-8", delete=False) as file:
            file.write("prefix target suffix\nother\n")
            filepath = file.name
        self.addCleanup(Path(filepath).unlink, missing_ok=True)
        result = await FileService().search_in_file(filepath, "target")
        self.assertEqual(result.line_numbers, [0])
        self.assertEqual(result.matches, ["prefix target suffix"])


if __name__ == "__main__":
    unittest.main()
