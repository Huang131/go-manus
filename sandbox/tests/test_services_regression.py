import asyncio
import os
import tempfile
import time
import unittest
from pathlib import Path
from unittest.mock import AsyncMock, patch

from app.interfaces.errors.exceptions import BadRequestException
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

    async def test_read_rejects_invalid_line_range(self):
        with self.assertRaises(BadRequestException):
            await FileService.read_file("/tmp/unused", start_line=3, end_line=2)

    async def test_search_reports_truncation(self):
        import app.services.file as file_service_module

        with tempfile.NamedTemporaryFile(mode="w", encoding="utf-8", delete=False) as file:
            file.write("target\n" * 3)
            filepath = file.name
        self.addCleanup(Path(filepath).unlink, missing_ok=True)
        with patch.object(file_service_module, "MAX_SEARCH_MATCHES", 2):
            result = await FileService().search_in_file(filepath, "target")
        self.assertEqual(result.line_numbers, [0, 1])
        self.assertTrue(result.truncated)

    async def test_find_files_reports_truncation(self):
        import app.services.file as file_service_module

        with tempfile.TemporaryDirectory() as directory:
            Path(directory, "a.txt").touch()
            Path(directory, "b.txt").touch()
            with patch.object(file_service_module, "MAX_FIND_FILES", 1):
                result = await FileService.find_files(directory, "*.txt")
        self.assertEqual(len(result.files), 1)
        self.assertTrue(result.truncated)

    async def test_upload_rejects_oversized_file_without_partial_target(self):
        import app.services.file as file_service_module

        class Upload:
            def __init__(self):
                self.file = __import__("io").BytesIO(b"12345")

        with tempfile.TemporaryDirectory() as directory:
            target = str(Path(directory, "uploaded.txt"))
            with patch.object(file_service_module, "MAX_UPLOAD_BYTES", 4):
                with self.assertRaises(BadRequestException):
                    await FileService.upload_file(Upload(), target)
            self.assertFalse(Path(target).exists())

    async def test_settings_default_does_not_expire_sandbox(self):
        from app.core.config import get_settings

        get_settings.cache_clear()
        with patch.dict(os.environ, {}, clear=False):
            os.environ.pop("SERVER_TIMEOUT_MINUTES", None)
            settings = get_settings()
        self.assertIsNone(settings.server_timeout_minutes)
        get_settings.cache_clear()

    async def test_restart_skips_fastapi_process(self):
        from app.services.supervisor import SupervisorService

        service = SupervisorService.__new__(SupervisorService)
        service.server = type("Server", (), {})()
        service.server.supervisor = type("Supervisor", (), {})()
        service.server.supervisor.getAllProcessInfo = lambda: [
            {"name": "app"}, {"name": "chrome"}, {"name": "xvfb"}
        ]
        service.server.supervisor.stopProcess = lambda name, wait: [name, wait]
        service.server.supervisor.startProcess = lambda name, wait: [name, wait]
        result = await service.restart()
        self.assertEqual(result.status, "restarted")
        self.assertEqual(result.stop_result, [["xvfb", True], ["chrome", True]])
        self.assertEqual(result.start_result, [["chrome", True], ["xvfb", True]])


if __name__ == "__main__":
    unittest.main()
