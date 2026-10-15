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

from app.interfaces.errors.exceptions import AppException, BadRequestException, NotFoundException, TimeoutException
from app.interfaces.schemas.file import FileWriteRequest
from app.interfaces.schemas.shell import ShellExecuteRequest, ShellWaitRequest
from app.models.file import FileReadResult
from app.models.shell import ConsoleRecord
from app.services.browser import BrowserService
from app.services.file import FileService
from app.services.shell import ShellService

from fakes import FakeProcess, FakeStream
from support import TempFileMixin, process_gone, wait_until


class ServiceRegressionTests(TempFileMixin, unittest.IsolatedAsyncioTestCase):
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
        # 条件等待“进程已消失”，替代固定次数轮询；/proc 仅作为僵尸态的补充判据。
        await wait_until(
            lambda: process_gone(child_pid),
            timeout=1.0,
            interval=0.05,
            msg=f"后台子进程仍存活: {child_pid}",
        )

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
        await wait_until(lambda: "concurrent-session" in service.active_shells)
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
        await wait_until(lambda: "kill-concurrent-session" in service.active_shells)
        termination = asyncio.create_task(service.kill_process("kill-concurrent-session"))
        # 让出一次调度：确保 kill 已进入终止流程（信号已发出）后再并发读取。
        # 这是调度让步而非等待时长的 sleep，不受机器负载影响。
        await asyncio.sleep(0)
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
        process = FakeProcess(
            stdout=FakeStream([b"first line\n"]),
            stderr=FakeStream([]),
            terminate_returncode=0,
            wait_returncode=0,
        )
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

        process = FakeProcess(
            stdout=FakeStream([b"x" * 32]),
            stderr=FakeStream([]),
            terminate_returncode=0,
            wait_returncode=0,
        )
        with patch.object(file_service_module, "MAX_PENDING_BYTES", 16), \
                patch.object(file_service_module.asyncio, "create_subprocess_exec", return_value=process):
            result = await FileService().search_in_file("/tmp/large.txt", "target", sudo=True)

        self.assertTrue(result.truncated)
        self.assertTrue(process.terminated)

    async def test_sudo_search_allows_many_short_lines_in_one_chunk(self):
        import app.services.file as file_service_module

        process = FakeProcess(
            stdout=FakeStream([b"target\n" * 4, b""]),
            stderr=FakeStream([]),
            wait_returncode=0,
        )
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

        async def block_on_read():
            read_started.set()
            await asyncio.Event().wait()

        process = FakeProcess(
            stdout=FakeStream(on_read=block_on_read),
            stderr=FakeStream([]),
            terminate_returncode=-15,
            wait_returncode=-15,
        )
        with patch("app.services.file.asyncio.create_subprocess_exec", return_value=process):
            request = asyncio.create_task(FileService.read_file("/tmp/cancelled.txt", sudo=True))
            await read_started.wait()
            request.cancel()
            with self.assertRaises(asyncio.CancelledError):
                await request

        self.assertTrue(process.terminated)

    async def test_sudo_write_terminates_process_when_request_is_cancelled(self):
        process = FakeProcess(
            terminate_returncode=-15,
            wait_returncode=-15,
            communicate_error=asyncio.CancelledError(),
        )
        with patch("app.services.file.asyncio.create_subprocess_exec", return_value=process):
            with self.assertRaises(asyncio.CancelledError):
                await FileService().write_file("/tmp/cancelled.txt", "content", sudo=True)

        self.assertTrue(process.terminated)

    async def test_sudo_delete_terminates_process_when_request_is_cancelled(self):
        process = FakeProcess(
            terminate_returncode=-15,
            wait_returncode=-15,
            communicate_error=asyncio.CancelledError(),
        )
        with patch("app.services.file.asyncio.create_subprocess_exec", return_value=process):
            with self.assertRaises(asyncio.CancelledError):
                await FileService().delete_file("/tmp/cancelled.txt", sudo=True)

        self.assertTrue(process.terminated)

    async def test_browser_action_terminates_child_process_when_request_is_cancelled(self):
        process = FakeProcess(terminate_returncode=-9, wait_returncode=-9)
        process_started = asyncio.Event()

        async def wait_for_action():
            process_started.set()
            await asyncio.Event().wait()

        process.communicate = wait_for_action
        service = BrowserService()
        with patch("app.services.browser.asyncio.create_subprocess_exec", return_value=process):
            action = asyncio.create_task(service._run_process("snapshot", {}, timeout=30))
            await process_started.wait()
            action.cancel()
            with self.assertRaises(asyncio.CancelledError):
                await action

        self.assertTrue(process.terminated)

    async def test_browser_screenshot_removes_partial_file_when_action_fails(self):
        with tempfile.TemporaryDirectory() as directory:
            service = BrowserService()
            async def fail_after_writing(_op, payload, _timeout, _budget_ms):
                Path(payload["path"]).write_bytes(b"partial screenshot")
                raise TimeoutException("screenshot timed out")

            with patch("app.services.browser.SCREENSHOT_DIR", directory), \
                    patch.object(service, "_run", side_effect=fail_after_writing):
                with self.assertRaises(TimeoutException):
                    await service.screenshot(full_page=True)

            self.assertEqual(list(Path(directory).iterdir()), [])

    async def test_browser_screenshot_removes_expired_orphan_files(self):
        import app.services.browser as browser_service_module

        with tempfile.TemporaryDirectory() as directory:
            expired = Path(directory, "expired.png")
            expired.write_bytes(b"expired")
            expired_at = time.time() - browser_service_module.SCREENSHOT_TTL_SECONDS - 1
            os.utime(expired, (expired_at, expired_at))
            recent = Path(directory, "recent.png")
            recent.write_bytes(b"recent")

            with patch("app.services.browser.SCREENSHOT_DIR", directory):
                BrowserService._cleanup_expired_screenshots()

            self.assertFalse(expired.exists())
            self.assertTrue(recent.exists())

    async def test_browser_cleanup_task_is_cancelled_when_service_closes(self):
        """孤儿截图清理任务属于 BrowserService 生命周期，关闭时不能遗留后台协程。"""
        service = BrowserService()
        service.start_cleanup()
        cleanup_task = service._screenshot_cleanup_task

        await service.aclose()

        self.assertIsNotNone(cleanup_task)
        self.assertTrue(cleanup_task.cancelled())

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
        service._deadline = datetime.now() + timedelta(minutes=1)
        service._setup_timer = lambda minutes: None

        class ConcurrentCancel:
            async def __aenter__(self):
                service._deadline = None
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

        filepath = self.write_temp_file()
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

        filepath = self.write_temp_file("x" * 4096)
        with patch.object(file_service_module, "MAX_PENDING_BYTES", 128):
            result = await FileService.read_file(filepath, max_length=None)
        self.assertTrue(result.truncated)
        # content 必须保持文件真实内容，截断信号只通过 truncated 字段暴露
        self.assertFalse(result.content.endswith("(truncated)"))
        self.assertLessEqual(len(result.content.encode("utf-8")), 128)

    async def test_long_utf8_line_respects_byte_limit(self):
        import app.services.file as file_service_module

        filepath = self.write_temp_file("中" * 1000)
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
        filepath = self.write_temp_file("prefix target suffix\nother\n")
        result = await FileService().search_in_file(filepath, "target")
        self.assertEqual(result.line_numbers, [0])
        self.assertEqual(result.matches, ["prefix target suffix"])

    async def test_read_rejects_invalid_line_range(self):
        with self.assertRaises(BadRequestException):
            await FileService.read_file("/tmp/unused", start_line=3, end_line=2)

    async def test_search_streams_until_match_limit(self):
        import app.services.file as file_service_module

        filepath = self.write_temp_file("target\n" * 5)
        lines_read = 0

        def tracked_lines(_filepath):
            nonlocal lines_read
            for _ in range(5):
                lines_read += 1
                yield file_service_module._LineChunk("target\n")

        with patch.object(file_service_module, "MAX_SEARCH_MATCHES", 2):
            with patch.object(file_service_module.FileService, "_iter_file_lines", side_effect=tracked_lines):
                result = await FileService().search_in_file(filepath, "target")
        self.assertEqual(result.line_numbers, [0, 1])
        self.assertTrue(result.truncated)
        self.assertEqual(lines_read, 3, "达到匹配上限后只应额外读取一行来确认截断")

    async def test_search_rejects_nested_quantifiers_before_scanning(self):
        import app.services.file as file_service_module

        filepath = self.write_temp_file("a" * 20_000 + "!\n")
        with patch.object(
                file_service_module.FileService,
                "_iter_file_lines",
                side_effect=AssertionError("不应执行病态正则搜索"),
        ):
            with self.assertRaises(BadRequestException):
                await FileService().search_in_file(filepath, "(a+)+$")

    async def test_search_rejects_nested_quantifiers_inside_lookaround(self):
        """环视也是正则 AST 的子树，不能成为 ReDoS 校验绕过路径。"""
        import app.services.file as file_service_module

        filepath = self.write_temp_file("a" * 20_000 + "!\n")
        with patch.object(
                file_service_module.FileService,
                "_iter_file_lines",
                side_effect=AssertionError("不应执行环视中的病态正则搜索"),
        ):
            with self.assertRaises(BadRequestException):
                await FileService().search_in_file(filepath, "(?=(a+)+$)")

    async def test_search_rejects_unbounded_quantifier_before_scanning(self):
        import app.services.file as file_service_module

        filepath = self.write_temp_file("a" * 20_000 + "!\n")
        with patch.object(
                file_service_module.FileService,
                "_iter_file_lines",
                side_effect=AssertionError("不应执行未受限正则搜索"),
        ):
            with self.assertRaises(BadRequestException):
                await FileService().search_in_file(filepath, "a+")

    async def test_search_truncates_lines_above_regex_work_limit(self):
        import app.services.file as file_service_module

        filepath = self.write_temp_file("a" * (file_service_module.MAX_SEARCH_LINE_BYTES + 1) + "\n")
        result = await FileService().search_in_file(filepath, "a{1,64}")
        self.assertEqual(result.matches, [])
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

    def test_namespec_handles_grouped_and_standalone_processes(self):
        """getAllProcessInfo 返回拆开的 name/group，RPC 需要拼好的 namespec。"""
        from app.services.supervisor import SupervisorService

        self.assertEqual(
            SupervisorService._namespec({"name": "chrome", "group": "services"}),
            "services:chrome",
        )
        # 未分组时 group == name，用单名即可被 supervisor 解析
        self.assertEqual(
            SupervisorService._namespec({"name": "app", "group": "app"}), "app"
        )
        self.assertEqual(SupervisorService._namespec({"name": "app"}), "app")

    async def test_restart_passes_group_namespec_to_rpc(self):
        """回归：进程归属 [group:services] 后，裸传短名会被 supervisor 判为 BAD_NAME。

        真实 getAllProcessInfo 返回 name=chrome / group=services，
        stopProcess/startProcess 只接受 services:chrome 这种 namespec。
        """
        from app.services.supervisor import SupervisorService

        service = SupervisorService.__new__(SupervisorService)
        service.server = type("Server", (), {})()
        service.server.supervisor = type("Supervisor", (), {})()
        service.server.supervisor.getAllProcessInfo = lambda: [
            {"name": "app", "group": "services", "statename": "RUNNING"},
            {"name": "chrome", "group": "services", "statename": "RUNNING"},
            {"name": "xvfb", "group": "services", "statename": "STOPPED"},
        ]
        stopped_names = []
        started_names = []

        def stop_process(name, wait):
            stopped_names.append(name)
            return [name, wait]

        def start_process(name, wait):
            started_names.append(name)
            return [name, wait]

        service.server.supervisor.stopProcess = stop_process
        service.server.supervisor.startProcess = start_process

        result = await service.restart()

        self.assertEqual(result.status, "restarted")
        self.assertEqual(stopped_names, ["services:chrome"])
        self.assertEqual(started_names, ["services:chrome", "services:xvfb"])
        self.assertEqual(result.stop_result, [["services:chrome", True]])
        self.assertEqual(
            result.start_result, [["services:chrome", True], ["services:xvfb", True]]
        )

    async def test_stop_all_processes_uses_group_namespec_and_skips_app(self):
        """回归：stop_all_processes 同样要拼 namespec，且必须排除 app 自身。"""
        from app.services.supervisor import SupervisorService

        service = SupervisorService.__new__(SupervisorService)
        service.server = type("Server", (), {})()
        service.server.supervisor = type("Supervisor", (), {})()
        service.server.supervisor.getAllProcessInfo = lambda: [
            {"name": "app", "group": "services", "statename": "RUNNING"},
            {"name": "chrome", "group": "services", "statename": "RUNNING"},
            {"name": "xvfb", "group": "services", "statename": "STOPPED"},
        ]
        stopped_names = []

        def stop_process(name, wait):
            stopped_names.append(name)
            return True

        service.server.supervisor.stopProcess = stop_process

        result = await service.stop_all_processes()

        self.assertEqual(result.status, "stopped")
        self.assertEqual(result.result, {"stopped": ["chrome"]})
        self.assertEqual(stopped_names, ["services:chrome"])

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

        filepath = self.write_temp_file("abcdefgh\n" * 4)
        with patch.object(file_service_module, "MAX_READ_BYTES", 1024):
            result = await FileService.read_file(filepath, max_length=10)
        self.assertTrue(result.truncated)
        # max_length 统计的是真实返回字符数（含换行），截断到文件前 10 个字符即为 "abcdefgh\na"
        self.assertEqual(result.content, "abcdefgh\na")

    async def test_read_file_handles_a_single_line_larger_than_chunk_size(self):
        import app.services.file as file_service_module

        filepath = self.write_temp_file("x" * (file_service_module.READ_CHUNK_BYTES + 100))
        with patch.object(file_service_module, "MAX_READ_BYTES", file_service_module.READ_CHUNK_BYTES + 200):
            result = await FileService.read_file(filepath, max_length=100)
        self.assertTrue(result.truncated)
        self.assertEqual(len(result.content), 100)

    async def test_replace_rejects_truncated_source(self):
        import app.services.file as file_service_module

        filepath = self.write_temp_file("content\n" * 4)
        with patch.object(file_service_module, "MAX_READ_BYTES", 8):
            with self.assertRaises(BadRequestException):
                await FileService().replace_in_file(filepath, "content", "changed")

    async def test_replace_rejects_oversized_result(self):
        import app.services.file as file_service_module

        filepath = self.write_temp_file("old")
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
        service._deadline = None
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
        service._deadline = datetime.now() + timedelta(minutes=1)
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
        service._deadline = None
        service.shutdown_task = None
        with self.assertRaises(BadRequestException):
            await service.activate_timeout(0)

    async def test_activate_and_cancel_timeout_toggle_auto_extend(self):
        from app.services.supervisor import SupervisorService

        service = SupervisorService.__new__(SupervisorService)
        service._deadline = None
        service.shutdown_task = None
        service._auto_extend = True
        service._setup_timer = lambda minutes: None

        # 活动保活由中间件在每次请求触发，绝不能改动开关：
        # 一旦改动，开关在首个请求后即被关掉，后续请求不再续期（保活只生效一次）。
        service._deadline = datetime.now() + timedelta(minutes=1)
        await service.keep_alive()
        self.assertTrue(service._auto_extend)

        # 显式设置销毁时间视为用户接管生命周期，自动保活必须让位；
        # 该开关原先由端点层在 service 调用后手动关闭，收进 service 后此处锁定归属。
        await service.activate_timeout(5)
        self.assertFalse(service._auto_extend)
        self.assertFalse(service.should_auto_extend("/api/shell/exec-command"))

        # 硬截止模式下保活是空操作，不得反过来把用户设定的开关打开。
        await service.keep_alive()
        self.assertFalse(service._auto_extend)

        # cancel 重新打开开关，但销毁计划已清空，should_auto_extend 的
        # timeout_active 前提会挡住保活——与 init 提交的既定设计一致。
        await service.cancel_timeout()
        self.assertTrue(service._auto_extend)
        self.assertFalse(service.should_auto_extend("/api/shell/exec-command"))


if __name__ == "__main__":
    unittest.main()
