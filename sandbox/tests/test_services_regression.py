import asyncio
import os
import tempfile
import time
import unittest
from pathlib import Path
from unittest.mock import AsyncMock, patch

from app.interfaces.errors.exceptions import BadRequestException
from app.interfaces.schemas.file import FileWriteRequest
from app.interfaces.schemas.shell import ShellExecuteRequest, ShellWaitRequest
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

    async def test_read_output_is_not_blocked_by_process_wait(self):
        service = ShellService()
        execution = asyncio.create_task(
            service.exec_command("concurrent-session", tempfile.gettempdir(), "sleep 2")
        )
        await asyncio.sleep(0.1)
        started = time.monotonic()
        result = await service.read_shell_output("concurrent-session")
        elapsed = time.monotonic() - started
        self.assertEqual(result.session_id, "concurrent-session")
        self.assertLess(elapsed, 1.0)
        await execution
        await service.kill_process("concurrent-session")

    async def test_read_output_is_not_blocked_while_process_is_terminated(self):
        service = ShellService()
        execution = asyncio.create_task(
            service.exec_command("kill-concurrent-session", tempfile.gettempdir(), "trap '' TERM; sleep 10")
        )
        await asyncio.sleep(0.1)
        termination = asyncio.create_task(service.kill_process("kill-concurrent-session"))
        await asyncio.sleep(0.1)
        started = time.monotonic()
        result = await service.read_shell_output("kill-concurrent-session")
        self.assertEqual(result.session_id, "kill-concurrent-session")
        self.assertLess(time.monotonic() - started, 1.0)
        await termination
        await execution

    async def test_sudo_read_waits_for_process_and_supports_special_path(self):
        process = AsyncMock()
        process.returncode = 0
        process.stdout = AsyncMock()
        process.stdout.read.side_effect = [b"hello\n", b""]
        process.stderr = AsyncMock()
        process.stderr.read.return_value = b""
        with patch("app.services.file.asyncio.create_subprocess_exec", return_value=process) as create:
            result = await FileService.read_file("/tmp/path with 'quote'.txt", sudo=True)
        self.assertIsInstance(result, FileReadResult)
        create.assert_awaited_once_with(
            "sudo", "cat", "/tmp/path with 'quote'.txt",
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        process.communicate.assert_not_awaited()

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

    async def test_write_reports_utf8_byte_count(self):
        with tempfile.TemporaryDirectory() as directory:
            filepath = str(Path(directory, "utf8.txt"))
            result = await FileService.write_file(filepath, "你好")
        self.assertEqual(result.bytes_written, len("你好".encode("utf-8")))

    async def test_append_copies_existing_file_in_chunks(self):
        import app.services.file as file_service_module

        with tempfile.TemporaryDirectory() as directory:
            filepath = str(Path(directory, "append.txt"))
            Path(filepath).write_text("old-content", encoding="utf-8")
            with patch.object(file_service_module.shutil, "copyfileobj", wraps=file_service_module.shutil.copyfileobj) as copy:
                result = await FileService.write_file(filepath, "-new", append=True)
            content = Path(filepath).read_text(encoding="utf-8")
        self.assertEqual(result.bytes_written, 4)
        self.assertEqual(content, "old-content-new")
        copy.assert_called_once()

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

    async def test_restart_skips_fastapi_process(self):
        from app.services.supervisor import SupervisorService

        service = SupervisorService.__new__(SupervisorService)
        service.server = type("Server", (), {})()
        service.server.supervisor = type("Supervisor", (), {})()
        service.server.supervisor.getAllProcessInfo = lambda: [
            {"name": "app", "statename": "RUNNING"},
            {"name": "chrome", "statename": "RUNNING"},
            {"name": "xvfb", "statename": "STOPPED"},
        ]
        service.server.supervisor.stopProcess = lambda name, wait: [name, wait]
        service.server.supervisor.startProcess = lambda name, wait: [name, wait]
        result = await service.restart()
        self.assertEqual(result.status, "restarted")
        self.assertEqual(result.stop_result, [["chrome", True]])
        self.assertEqual(result.start_result, [["chrome", True], ["xvfb", True]])

    def test_request_schemas_reject_unbounded_values(self):
        with self.assertRaises(ValueError):
            FileWriteRequest(filepath="/tmp/a", content="x" * (10 * 1024 * 1024 + 1))
        with self.assertRaises(ValueError):
            ShellExecuteRequest(command="")
        with self.assertRaises(ValueError):
            ShellWaitRequest(session_id="s", seconds=24 * 60 * 60 + 1)

    async def test_file_operations_reject_directory_as_file(self):
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaises(Exception):
                await FileService.read_file(directory)
            with self.assertRaises(Exception):
                await FileService().search_in_file(directory, "anything")

    async def test_read_rejects_oversized_file_before_loading(self):
        import app.services.file as file_service_module

        with tempfile.NamedTemporaryFile() as file:
            file.write(b"x")
            file.flush()
            with patch.object(file_service_module, "MAX_READ_BYTES", 0):
                with self.assertRaises(BadRequestException):
                    await FileService.read_file(file.name)

    async def test_read_file_streams_and_marks_truncated_content(self):
        import app.services.file as file_service_module

        with tempfile.NamedTemporaryFile(mode="w", encoding="utf-8", delete=False) as file:
            file.write("abcdefgh\n" * 4)
            filepath = file.name
        self.addCleanup(Path(filepath).unlink, missing_ok=True)
        with patch.object(file_service_module, "MAX_READ_BYTES", 1024):
            result = await FileService.read_file(filepath, max_length=10)
        self.assertTrue(result.truncated)
        self.assertTrue(result.content.endswith("(truncated)"))
        self.assertLessEqual(len(result.content), 21)

    async def test_read_file_handles_a_single_line_larger_than_chunk_size(self):
        import app.services.file as file_service_module

        with tempfile.NamedTemporaryFile(mode="w", encoding="utf-8", delete=False) as file:
            file.write("x" * (file_service_module.READ_CHUNK_BYTES + 100))
            filepath = file.name
        self.addCleanup(Path(filepath).unlink, missing_ok=True)
        with patch.object(file_service_module, "MAX_READ_BYTES", file_service_module.READ_CHUNK_BYTES + 200):
            result = await FileService.read_file(filepath, max_length=100)
        self.assertTrue(result.truncated)
        self.assertEqual(len(result.content), 100 + len("(truncated)"))

    async def test_search_streams_until_match_limit(self):
        import app.services.file as file_service_module

        with tempfile.NamedTemporaryFile(mode="w", encoding="utf-8", delete=False) as file:
            file.write("target\n" * 5)
            filepath = file.name
        self.addCleanup(Path(filepath).unlink, missing_ok=True)
        with patch.object(file_service_module, "MAX_SEARCH_MATCHES", 2):
            result = await FileService().search_in_file(filepath, "target")
        self.assertEqual(result.line_numbers, [0, 1])
        self.assertTrue(result.truncated)

    async def test_replace_rejects_truncated_source(self):
        import app.services.file as file_service_module

        with tempfile.NamedTemporaryFile(mode="w", encoding="utf-8", delete=False) as file:
            file.write("content\n" * 4)
            filepath = file.name
        self.addCleanup(Path(filepath).unlink, missing_ok=True)
        with patch.object(file_service_module, "MAX_READ_BYTES", 8):
            with self.assertRaises(BadRequestException):
                await FileService().replace_in_file(filepath, "content", "changed")

    async def test_find_files_rejects_file_as_directory(self):
        with tempfile.NamedTemporaryFile() as file:
            with self.assertRaises(Exception):
                await FileService.find_files(file.name, "*")

    async def test_check_file_exists_returns_false_for_directory(self):
        with tempfile.TemporaryDirectory() as directory:
            result = await FileService.check_file_exists(directory)
        self.assertFalse(result.exists)

    async def test_sudo_delete_uses_argument_array(self):
        process = AsyncMock()
        process.returncode = 0
        process.communicate.return_value = (b"", b"")
        with patch("app.services.file.asyncio.create_subprocess_exec", return_value=process) as create:
            result = await FileService().delete_file("/tmp/path with 'quote'.txt", sudo=True)
        self.assertTrue(result.deleted)
        create.assert_awaited_once_with(
            "sudo", "rm", "--", "/tmp/path with 'quote'.txt",
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )

    async def test_activate_timeout_zero_does_not_fall_back_to_default(self):
        from app.services.supervisor import SupervisorService

        service = SupervisorService.__new__(SupervisorService)
        service.timeout_active = False
        service.shutdown_time = None
        service.shutdown_task = None
        service.shutdown_timer = None
        with self.assertRaises(BadRequestException):
            await service.activate_timeout(0)


if __name__ == "__main__":
    unittest.main()
