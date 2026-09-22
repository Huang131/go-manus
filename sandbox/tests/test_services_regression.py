import asyncio
import os
import stat
import tempfile
import time
import unittest
from datetime import datetime, timedelta
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import AsyncMock, MagicMock, patch

from app.interfaces.errors.exceptions import AppException, BadRequestException, NotFoundException
from app.interfaces.schemas.file import FileWriteRequest
from app.interfaces.schemas.shell import ShellExecuteRequest, ShellWaitRequest
from app.models.file import FileReadResult
from app.models.shell import ConsoleRecord
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

    async def test_shell_shutdown_terminates_active_processes(self):
        service = ShellService()
        await service.exec_command("shutdown-session", tempfile.gettempdir(), "sleep 30")
        await service.shutdown()
        self.assertEqual(service.active_shells, {})
        self.assertEqual(service.reader_tasks, {})
        self.assertEqual(service.session_locks, {})

    async def test_kill_process_terminates_background_descendant_after_shell_exits(self):
        service = ShellService()
        with tempfile.TemporaryDirectory() as directory:
            pid_file = Path(directory, "child.pid")
            result = await service.exec_command(
                "background-child-session",
                directory,
                f"sleep 30 >/dev/null 2>&1 & echo $! > {pid_file}",
            )
            child_pid = int(pid_file.read_text(encoding="utf-8"))
            self.assertEqual(result.status, "completed")

            kill_result = await service.kill_process("background-child-session")

        self.assertEqual(kill_result.status, "terminated")
        for _ in range(20):
            try:
                os.kill(child_pid, 0)
            except ProcessLookupError:
                break
            if Path(f"/proc/{child_pid}/stat").read_text(encoding="utf-8").split()[2] == "Z":
                break
            await asyncio.sleep(0.05)
        else:
            self.fail(f"后台子进程仍存活: {child_pid}")

    async def test_terminate_and_wait_falls_back_without_process_group_id(self):
        finished = asyncio.Event()

        class Process:
            returncode = None

            async def wait(self):
                await finished.wait()
                return self.returncode

        process = Process()
        service = ShellService()

        async def timeout(awaitable, **_kwargs):
            raise asyncio.TimeoutError

        def force_kill(target, process_group_id=None):
            self.assertIsNone(process_group_id)
            target.returncode = -9
            finished.set()

        with patch("app.services.shell.asyncio.wait_for", side_effect=timeout), \
                patch.object(service, "_terminate_process_tree"), \
                patch.object(service, "_kill_process_tree", side_effect=force_kill):
            await service._terminate_and_wait(process, None)

        self.assertEqual(process.returncode, -9)

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

    async def test_sudo_read_stops_process_when_end_line_stops_consuming_output(self):
        class Stream:
            def __init__(self, chunks):
                self.chunks = iter(chunks)

            async def read(self, size=-1):
                return next(self.chunks, b"")

        class Process:
            def __init__(self):
                self.returncode = None
                self.stdout = Stream([b"first line\n"])
                self.stderr = Stream([])
                self.terminated = False

            def terminate(self):
                self.terminated = True
                self.returncode = 0

            async def wait(self):
                self.returncode = 0
                return self.returncode

        process = Process()
        with patch("app.services.file.asyncio.create_subprocess_exec", return_value=process):
            result = await FileService.read_file("/tmp/large.txt", sudo=True, end_line=1)

        self.assertEqual(result.content, "first line")
        self.assertTrue(process.terminated)

    async def test_relative_path_write_does_not_create_empty_directory(self):
        with tempfile.TemporaryDirectory() as directory:
            previous_directory = Path.cwd()
            try:
                os.chdir(directory)
                with patch("app.services.file.os.makedirs") as makedirs:
                    result = await FileService().write_file("work.txt", "content")
            finally:
                os.chdir(previous_directory)
            self.assertEqual(result.bytes_written, len("content"))
            makedirs.assert_not_called()

    async def test_sudo_write_passes_content_through_stdin(self):
        process = AsyncMock()
        process.returncode = 0
        process.communicate.return_value = (b"", b"")
        with patch("app.services.file.asyncio.create_subprocess_exec", return_value=process) as create:
            result = await FileService().write_file("/tmp/path with 'quote'.txt", "hello", sudo=True)
        self.assertEqual(result.bytes_written, len("hello"))
        create.assert_awaited_once_with(
            "sudo", "tee", "/tmp/path with 'quote'.txt",
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        process.communicate.assert_awaited_once_with(b"hello")

    async def test_sudo_search_stops_process_for_unterminated_long_line(self):
        import app.services.file as file_service_module

        class Stream:
            def __init__(self, chunks):
                self.chunks = iter(chunks)

            async def read(self, size=-1):
                return next(self.chunks, b"")

        class Process:
            def __init__(self):
                self.returncode = None
                self.stdout = Stream([b"x" * 32])
                self.stderr = Stream([])
                self.terminated = False

            def terminate(self):
                self.terminated = True
                self.returncode = 0

            async def wait(self):
                self.returncode = 0
                return self.returncode

        process = Process()
        with patch.object(file_service_module, "MAX_PENDING_BYTES", 16), \
                patch.object(file_service_module.asyncio, "create_subprocess_exec", return_value=process):
            result = await FileService().search_in_file("/tmp/large.txt", "target", sudo=True)

        self.assertTrue(result.truncated)
        self.assertTrue(process.terminated)

    async def test_sudo_search_allows_many_short_lines_in_one_chunk(self):
        import app.services.file as file_service_module

        class Stream:
            def __init__(self, chunks):
                self.chunks = iter(chunks)

            async def read(self, size=-1):
                return next(self.chunks, b"")

        class Process:
            def __init__(self):
                self.returncode = None
                self.stdout = Stream([b"target\n" * 4, b""])
                self.stderr = Stream([])
                self.terminated = False

            def terminate(self):
                self.terminated = True
                self.returncode = 0

            async def wait(self):
                self.returncode = 0
                return self.returncode

        process = Process()
        with patch.object(file_service_module, "MAX_PENDING_BYTES", 8), \
                patch.object(file_service_module.asyncio, "create_subprocess_exec", return_value=process):
            result = await FileService().search_in_file("/tmp/short-lines.txt", "target", sudo=True)

        self.assertEqual(result.line_numbers, [0, 1, 2, 3])
        self.assertFalse(result.truncated)
        self.assertFalse(process.terminated)

    async def test_write_preserves_existing_file_mode(self):
        with tempfile.TemporaryDirectory() as directory:
            filepath = Path(directory, "mode.txt")
            filepath.write_text("old", encoding="utf-8")
            filepath.chmod(0o640)
            await FileService().write_file(str(filepath), "new")
            self.assertEqual(stat.S_IMODE(filepath.stat().st_mode), 0o640)

    async def test_upload_preserves_existing_file_mode(self):
        class Upload:
            def __init__(self):
                self.file = __import__("io").BytesIO(b"new")

        with tempfile.TemporaryDirectory() as directory:
            filepath = Path(directory, "upload.txt")
            filepath.write_text("old", encoding="utf-8")
            filepath.chmod(0o640)
            await FileService.upload_file(Upload(), str(filepath))
            self.assertEqual(stat.S_IMODE(filepath.stat().st_mode), 0o640)

    async def test_find_files_rejects_parent_directory_pattern(self):
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaises(BadRequestException):
                await FileService.find_files(directory, "../*")

    async def test_sudo_read_terminates_process_when_request_is_cancelled(self):
        read_started = asyncio.Event()

        class Stream:
            async def read(self, size=-1):
                read_started.set()
                await asyncio.Event().wait()

        class Process:
            def __init__(self):
                self.returncode = None
                self.stdout = Stream()
                class EmptyStream:
                    async def read(self, size=-1):
                        return b""

                self.stderr = EmptyStream()
                self.terminated = False

            def terminate(self):
                self.terminated = True
                self.returncode = -15

            async def wait(self):
                self.returncode = -15
                return self.returncode

        process = Process()
        with patch("app.services.file.asyncio.create_subprocess_exec", return_value=process):
            request = asyncio.create_task(FileService.read_file("/tmp/cancelled.txt", sudo=True))
            await read_started.wait()
            request.cancel()
            with self.assertRaises(asyncio.CancelledError):
                await request

        self.assertTrue(process.terminated)

    async def test_sudo_write_terminates_process_when_request_is_cancelled(self):
        class Process:
            def __init__(self):
                self.returncode = None
                self.terminated = False

            async def communicate(self, data):
                raise asyncio.CancelledError

            def terminate(self):
                self.terminated = True
                self.returncode = -15

            async def wait(self):
                self.returncode = -15

        process = Process()
        with patch("app.services.file.asyncio.create_subprocess_exec", return_value=process):
            with self.assertRaises(asyncio.CancelledError):
                await FileService().write_file("/tmp/cancelled.txt", "content", sudo=True)

        self.assertTrue(process.terminated)

    async def test_sudo_delete_terminates_process_when_request_is_cancelled(self):
        class Process:
            def __init__(self):
                self.returncode = None
                self.terminated = False

            async def communicate(self):
                raise asyncio.CancelledError

            def terminate(self):
                self.terminated = True
                self.returncode = -15

            async def wait(self):
                self.returncode = -15

        process = Process()
        with patch("app.services.file.asyncio.create_subprocess_exec", return_value=process):
            with self.assertRaises(asyncio.CancelledError):
                await FileService().delete_file("/tmp/cancelled.txt", sudo=True)

        self.assertTrue(process.terminated)

    async def test_write_input_preserves_bad_request_error(self):
        service = ShellService()
        await service.exec_command("write-error-session", tempfile.gettempdir(), "exit 0")
        await service.wait_process("write-error-session", seconds=2)
        with self.assertRaises(BadRequestException):
            await service.write_shell_input("write-error-session", "input", True)

    async def test_write_input_does_not_record_when_stdin_write_fails(self):
        class BrokenStdin:
            def write(self, data):
                raise BrokenPipeError("stdin closed")

            async def drain(self):
                return None

        process = type("Process", (), {"returncode": None, "stdin": BrokenStdin()})()
        service = ShellService()
        service.active_shells["broken-stdin"] = SimpleNamespace(
            process=process,
            exec_dir=tempfile.gettempdir(),
            output="existing output",
            console_records=[ConsoleRecord(ps1="$", command="cmd", output="existing record")],
        )

        with self.assertRaises(AppException):
            await service.write_shell_input("broken-stdin", "input", True)

        shell = service.active_shells["broken-stdin"]
        self.assertEqual(shell.output, "existing output")
        self.assertEqual(shell.console_records[-1].output, "existing record")

    async def test_extend_timeout_recovers_if_state_is_cleared_before_lock(self):
        from app.services.supervisor import SupervisorService

        service = SupervisorService.__new__(SupervisorService)
        service.timeout_active = True
        service.shutdown_time = datetime.now() + timedelta(minutes=1)
        service._setup_timer = lambda minutes: None

        class ConcurrentCancel:
            async def __aenter__(self):
                service.timeout_active = False
                service.shutdown_time = None
                return self

            async def __aexit__(self, exc_type, exc, tb):
                return False

        service._state_lock = ConcurrentCancel()
        result = await service.extend_timeout(2)

        self.assertEqual(result.status, "timeout_activated")
        self.assertTrue(result.active)
        self.assertEqual(result.timeout_minutes, 2)
        self.assertIsNotNone(service.shutdown_time)

    async def test_write_reports_utf8_byte_count(self):
        with tempfile.TemporaryDirectory() as directory:
            filepath = str(Path(directory, "utf8.txt"))
            result = await FileService().write_file(filepath, "你好")
        self.assertEqual(result.bytes_written, len("你好".encode("utf-8")))

    async def test_append_copies_existing_file_in_chunks(self):
        import app.services.file as file_service_module

        with tempfile.TemporaryDirectory() as directory:
            filepath = str(Path(directory, "append.txt"))
            Path(filepath).write_text("old-content", encoding="utf-8")
            with patch.object(file_service_module.shutil, "copyfileobj", wraps=file_service_module.shutil.copyfileobj) as copy:
                result = await FileService().write_file(filepath, "-new", append=True)
            content = Path(filepath).read_text(encoding="utf-8")
        self.assertEqual(result.bytes_written, 4)
        self.assertEqual(content, "old-content-new")
        copy.assert_called_once()

    async def test_concurrent_append_preserves_both_writes(self):
        with tempfile.TemporaryDirectory() as directory:
            filepath = str(Path(directory, "concurrent.txt"))
            service = FileService()
            await asyncio.gather(
                service.write_file(filepath, "a", append=True),
                service.write_file(filepath, "b", append=True),
            )
            content = Path(filepath).read_text(encoding="utf-8")
        self.assertEqual(sorted(content), ["a", "b"])
        self.assertNotIn(filepath, service._write_locks)

    async def test_delete_file_moves_regular_remove_to_thread(self):
        import app.services.file as file_service_module

        with tempfile.NamedTemporaryFile(delete=False) as file:
            filepath = file.name
        self.addCleanup(Path(filepath).unlink, missing_ok=True)
        calls = []

        async def run_in_thread(func, *args):
            calls.append((func, args))
            return func(*args)

        with patch.object(file_service_module.asyncio, "to_thread", side_effect=run_in_thread) as to_thread:
            result = await FileService().delete_file(filepath)

        self.assertTrue(result.deleted)
        to_thread.assert_awaited_once()
        self.assertIs(calls[0][0], os.remove)
        self.assertEqual(calls[0][1], (filepath,))

    async def test_long_unterminated_line_is_bounded_and_truncated(self):
        import app.services.file as file_service_module

        with tempfile.NamedTemporaryFile(mode="w", encoding="utf-8", delete=False) as file:
            file.write("x" * 4096)
            filepath = file.name
        self.addCleanup(Path(filepath).unlink, missing_ok=True)
        with patch.object(file_service_module, "MAX_PENDING_BYTES", 128):
            result = await FileService.read_file(filepath, max_length=None)
        self.assertTrue(result.truncated)
        # content 必须保持文件真实内容，截断信号只通过 truncated 字段暴露
        self.assertFalse(result.content.endswith("(truncated)"))
        self.assertLessEqual(len(result.content.encode("utf-8")), 128)

    async def test_long_utf8_line_respects_byte_limit(self):
        import app.services.file as file_service_module

        with tempfile.NamedTemporaryFile(mode="w", encoding="utf-8", delete=False) as file:
            file.write("中" * 1000)
            filepath = file.name
        self.addCleanup(Path(filepath).unlink, missing_ok=True)
        with patch.object(file_service_module, "MAX_PENDING_BYTES", 128):
            result = await FileService.read_file(filepath, max_length=None)
        content = result.content
        self.assertTrue(result.truncated)
        self.assertLessEqual(len(content.encode("utf-8")), 128)

    async def test_replace_and_append_preserve_both_operations(self):
        with tempfile.TemporaryDirectory() as directory:
            filepath = str(Path(directory, "replace-append.txt"))
            Path(filepath).write_text("old", encoding="utf-8")
            service = FileService()
            await asyncio.gather(
                service.replace_in_file(filepath, "old", "new"),
                service.write_file(filepath, "-append", append=True),
            )
            content = Path(filepath).read_text(encoding="utf-8")
        self.assertEqual(content, "new-append")

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

    async def test_read_file_streams_and_flags_truncated_content(self):
        import app.services.file as file_service_module

        with tempfile.NamedTemporaryFile(mode="w", encoding="utf-8", delete=False) as file:
            file.write("abcdefgh\n" * 4)
            filepath = file.name
        self.addCleanup(Path(filepath).unlink, missing_ok=True)
        with patch.object(file_service_module, "MAX_READ_BYTES", 1024):
            result = await FileService.read_file(filepath, max_length=10)
        self.assertTrue(result.truncated)
        # max_length 统计的是真实返回字符数（含换行），截断到文件前 10 个字符即为 "abcdefgh\na"
        self.assertEqual(result.content, "abcdefgh\na")

    async def test_read_file_handles_a_single_line_larger_than_chunk_size(self):
        import app.services.file as file_service_module

        with tempfile.NamedTemporaryFile(mode="w", encoding="utf-8", delete=False) as file:
            file.write("x" * (file_service_module.READ_CHUNK_BYTES + 100))
            filepath = file.name
        self.addCleanup(Path(filepath).unlink, missing_ok=True)
        with patch.object(file_service_module, "MAX_READ_BYTES", file_service_module.READ_CHUNK_BYTES + 200):
            result = await FileService.read_file(filepath, max_length=100)
        self.assertTrue(result.truncated)
        self.assertEqual(len(result.content), 100)

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

    async def test_replace_rejects_oversized_result(self):
        import app.services.file as file_service_module

        with tempfile.NamedTemporaryFile(mode="w", encoding="utf-8", delete=False) as file:
            file.write("old")
            filepath = file.name
        self.addCleanup(Path(filepath).unlink, missing_ok=True)
        with patch.object(file_service_module, "MAX_WRITE_BYTES", 2):
            with self.assertRaises(BadRequestException):
                await FileService().replace_in_file(filepath, "old", "larger")

    async def test_unknown_shell_session_does_not_register_lock(self):
        service = ShellService()
        with self.assertRaises(NotFoundException):
            await service.read_shell_output("missing-session")
        self.assertNotIn("missing-session", service.session_locks)

    async def test_supervisor_timeout_updates_are_serialized(self):
        from app.services.supervisor import SupervisorService

        service = SupervisorService.__new__(SupervisorService)
        service.timeout_active = False
        service.shutdown_time = None
        service.shutdown_task = None
        service._setup_timer = lambda minutes: None
        await asyncio.gather(
            service.activate_timeout(1),
            service.activate_timeout(2),
        )
        status = await service.get_timeout_status()
        self.assertTrue(status.active)
        self.assertGreater(status.remaining_seconds, 0)

    async def test_supervisor_cancel_invalidates_existing_timer(self):
        from app.services.supervisor import SupervisorService

        service = SupervisorService.__new__(SupervisorService)
        service.timeout_active = True
        service.shutdown_time = datetime.now() + timedelta(minutes=1)
        service.shutdown_task = MagicMock()
        shutdown_task = service.shutdown_task
        service._timer_generation = 4

        result = await service.cancel_timeout()

        self.assertFalse(result.active)
        shutdown_task.cancel.assert_called_once_with()
        self.assertEqual(service._timer_generation, 5)
        self.assertIsNone(service.shutdown_task)

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
        with self.assertRaises(BadRequestException):
            await service.activate_timeout(0)


if __name__ == "__main__":
    unittest.main()
