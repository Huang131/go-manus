#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
@Time    : 2025/05/12 14:36
@Author  : thezehui@gmail.com
@File    : file.py
"""
import asyncio
import glob
import itertools
import logging
import os.path
import re
import stat
import shutil
import tempfile
from contextlib import asynccontextmanager
from dataclasses import dataclass
from typing import AsyncIterator, Dict, Iterator, Optional

from fastapi import UploadFile

from app.interfaces.errors.exceptions import (
    NotFoundException,
    BadRequestException,
    AppException
)
from app.models.file import (
    FileReadResult,
    FileWriteResult,
    FileReplaceResult,
    FileSearchResult,
    FileFindResult,
    FileUploadResult,
    FileCheckResult,
    FileDeleteResult
)

logger = logging.getLogger(__name__)

MAX_SEARCH_MATCHES = 10_000
MAX_FIND_FILES = 10_000
MAX_UPLOAD_BYTES = 100 * 1024 * 1024
MAX_WRITE_BYTES = 10 * 1024 * 1024
MAX_READ_BYTES = 100 * 1024 * 1024
PROCESS_TERMINATE_TIMEOUT = 1
READ_CHUNK_BYTES = 64 * 1024
MAX_PENDING_BYTES = 1024 * 1024


@dataclass
class _WriteLockEntry:
    """按路径管理写锁及引用数，空闲条目可从注册表移除。"""
    lock: asyncio.Lock
    references: int = 0


@dataclass(frozen=True)
class _LineChunk:
    """文件流的一段逻辑行；truncated 表示单行超过内存上限。"""
    text: str
    truncated: bool = False


def _truncate_utf8(text: str, max_bytes: int) -> str:
    """按 UTF-8 字节上限截断文本，避免多字节字符被切成非法序列。"""
    encoded = text.encode("utf-8")
    if len(encoded) <= max_bytes:
        return text
    return encoded[:max_bytes].decode("utf-8", errors="ignore")


class FileService:
    """文件沙箱服务"""

    def __init__(self) -> None:
        # 服务无状态，保留显式构造函数便于 FastAPI 依赖注入和后续扩展。
        super().__init__()
        # 仅串行化同一文件的写入，不影响不同文件并发处理。
        self._write_locks: Dict[str, _WriteLockEntry] = {}

    @asynccontextmanager
    async def _write_lock(self, filepath: str) -> AsyncIterator[None]:
        """获取路径锁并在最后一个使用者离开后回收条目。"""
        entry = self._write_locks.get(filepath)
        if entry is None:
            entry = _WriteLockEntry(lock=asyncio.Lock())
            self._write_locks[filepath] = entry
        entry.references += 1
        acquired = False
        try:
            await entry.lock.acquire()
            acquired = True
            yield
        finally:
            if acquired:
                entry.lock.release()
            entry.references -= 1
            # 只有没有等待者且锁已释放时才能删除，避免并发请求拿到不同的锁。
            if entry.references == 0 and not entry.lock.locked():
                if self._write_locks.get(filepath) is entry:
                    self._write_locks.pop(filepath, None)

    @staticmethod
    def _ensure_regular_file(filepath: str) -> None:
        """文件操作只接受普通文件，避免目录被当作文件读取或删除。"""
        if not os.path.isfile(filepath):
            raise NotFoundException(f"文件不存在或不是普通文件: {filepath}")

    @staticmethod
    def _ensure_directory(dir_path: str) -> None:
        """目录遍历只接受目录路径，尽早返回明确的 404。"""
        if not os.path.isdir(dir_path):
            raise NotFoundException(f"目录不存在或不是目录: {dir_path}")

    @staticmethod
    def _validate_read_limit() -> None:
        """校验全局读取上限；具体文件按流式读取，超限时返回截断结果。"""
        if MAX_READ_BYTES < 1:
            raise BadRequestException("读取上限必须大于0")

    @staticmethod
    async def _stop_process(process) -> None:
        """停止达到读取上限的子进程，避免 terminate 后永久等待。"""
        if process.returncode is not None:
            return
        try:
            process.terminate()
        except ProcessLookupError:
            return
        try:
            await asyncio.wait_for(process.wait(), timeout=PROCESS_TERMINATE_TIMEOUT)
        except asyncio.TimeoutError:
            try:
                process.kill()
            except ProcessLookupError:
                return
            try:
                await process.wait()
            except ProcessLookupError:
                pass

    @classmethod
    async def read_file(
            cls,
            filepath: str,
            start_line: Optional[int] = None,
            end_line: Optional[int] = None,
            sudo: bool = False,
            max_length: Optional[int] = 10000,
    ) -> FileReadResult:
        """根据传递的文件路径+起始行号+权限+最大长度读取文件内容"""
        try:
            if max_length is not None and max_length < 1:
                raise BadRequestException("max_length 必须大于0")
            if ((start_line is not None and start_line < 0)
                    or (end_line is not None and end_line < 0)):
                raise BadRequestException("行号不能小于0")
            if start_line is not None and end_line is not None and start_line > end_line:
                raise BadRequestException("起始行不能大于结束行")
            cls._validate_read_limit()
            # 1.检测在当前权限下能否获取该文件
            if not os.path.isfile(filepath) and not sudo:
                logger.error(f"要读取的文件不存在或无权限: {filepath}")
                raise NotFoundException(f"要读取的文件不存在或无权限: {filepath}")

            # 2.ubuntu系统下统一使用utf-8编码
            encoding = "utf-8"

            # 3.判断是否为sudo，如果是sudo系统则使用命令行的形式读取文件
            if sudo:
                # 使用参数数组传递路径，避免空格和引号破坏 shell 命令。
                process = await asyncio.create_subprocess_exec(
                    "sudo", "cat", filepath,
                    stdout=asyncio.subprocess.PIPE,
                    stderr=asyncio.subprocess.PIPE,
                )

                try:
                    content, truncated, stopped_early = await cls._read_stream(
                        process.stdout, max_length, start_line, end_line
                    )
                    if truncated or stopped_early:
                        await asyncio.shield(cls._stop_process(process))
                    elif process.returncode is None:
                        await process.wait()
                    stderr = await process.stderr.read() if process.stderr else b""
                    # 主动停止 cat 是达到读取范围或上限的正常路径，不能误报为失败。
                    if process.returncode != 0 and not (truncated or stopped_early):
                        raise BadRequestException(f"阅读文件失败: {stderr.decode(errors='replace')}")
                    return FileReadResult(filepath=filepath, content=content, truncated=truncated)
                finally:
                    # CancelledError 不属于 Exception，必须在 finally 中清理 sudo 子进程。
                    if process.returncode is None:
                        await asyncio.shield(cls._stop_process(process))
            else:
                cls._ensure_regular_file(filepath)
                def read_lines() -> tuple[str, bool]:
                    try:
                        return cls._collect_lines(
                            cls._iter_file_lines(filepath), max_length, start_line, end_line
                        )
                    except Exception as exc:
                        raise AppException(msg=f"读取文件失败: {exc}") from exc
                content, truncated = await asyncio.to_thread(read_lines)
                return FileReadResult(filepath=filepath, content=content, truncated=truncated)
        except Exception as e:
            # 13.判断异常类型执行不同操作
            if isinstance(e, BadRequestException) or isinstance(e, AppException):
                raise
            raise AppException(f"文件读取失败: {str(e)}")

    @staticmethod
    def _collect_lines(lines: Iterator[_LineChunk], max_length: Optional[int], start_line: Optional[int],
                       end_line: Optional[int]) -> tuple[str, bool]:
        """逐行收集范围内内容，任何上限命中后立即停止读取。"""
        start = start_line or 0
        parts = []
        length = 0
        bytes_seen = 0
        truncated = False
        for index, chunk in enumerate(lines):
            line = chunk.text
            bytes_seen += len(line.encode("utf-8"))
            if bytes_seen > MAX_READ_BYTES or chunk.truncated:
                truncated = True
                break
            if index < start:
                continue
            if end_line is not None and index >= end_line:
                break
            value = line.rstrip("\r\n")
            separator = "\n" if parts else ""
            addition = separator + value
            if max_length is not None and length + len(addition) > max_length:
                remaining = max_length - length
                if remaining > 0:
                    parts.append(addition[:remaining])
                truncated = True
                break
            parts.append(addition)
            length += len(addition)
        return "".join(parts) + ("(truncated)" if truncated else ""), truncated

    @staticmethod
    def _iter_file_lines(filepath: str) -> Iterator[_LineChunk]:
        """按固定块解码文件，超长无换行内容达到上限即截断。"""
        import codecs
        decoder = codecs.getincrementaldecoder("utf-8")(errors="replace")
        pending = ""
        with open(filepath, "rb") as stream:
            while True:
                chunk = stream.read(READ_CHUNK_BYTES)
                if not chunk:
                    break
                pending += decoder.decode(chunk)
                while "\n" in pending:
                    line, pending = pending.split("\n", 1)
                    if len(line.encode("utf-8")) > MAX_PENDING_BYTES:
                        yield _LineChunk(_truncate_utf8(line, MAX_PENDING_BYTES), truncated=True)
                        return
                    yield _LineChunk(line + "\n")
                if len(pending.encode("utf-8")) > MAX_PENDING_BYTES:
                    yield _LineChunk(_truncate_utf8(pending, MAX_PENDING_BYTES), truncated=True)
                    return
        pending += decoder.decode(b"", final=True)
        if pending:
            if len(pending.encode("utf-8")) > MAX_PENDING_BYTES:
                yield _LineChunk(_truncate_utf8(pending, MAX_PENDING_BYTES), truncated=True)
            else:
                yield _LineChunk(pending)

    @staticmethod
    async def _read_stream(stream, max_length: Optional[int], start_line: Optional[int],
                           end_line: Optional[int]) -> tuple[str, bool, bool]:
        """按固定块读取 sudo 输出，避免超长单行触发 readline 缓冲上限。"""
        if stream is None:
            return "", False, False
        start = start_line or 0
        parts, length, bytes_seen, index = [], 0, 0, 0
        pending = ""
        truncated = False
        stopped_early = False
        while True:
            raw = await stream.read(READ_CHUNK_BYTES)
            if not raw:
                break
            bytes_seen += len(raw)
            if bytes_seen > MAX_READ_BYTES:
                truncated = True
                break
            pending += raw.decode("utf-8", errors="replace")
            while "\n" in pending:
                line, pending = pending.split("\n", 1)
                if len(line.encode("utf-8")) > MAX_PENDING_BYTES:
                    truncated = True
                    break
                if index >= start and (end_line is None or index < end_line):
                    addition = ("\n" if parts else "") + line.rstrip("\r")
                    if max_length is not None and length + len(addition) > max_length:
                        remaining = max_length - length
                        if remaining > 0:
                            parts.append(addition[:remaining])
                        truncated = True
                        break
                    parts.append(addition)
                    length += len(addition)
                index += 1
            if truncated or (end_line is not None and index >= end_line):
                stopped_early = not truncated and end_line is not None and index >= end_line
                break
            if len(pending.encode("utf-8")) > MAX_PENDING_BYTES:
                truncated = True
                break
        if not truncated and pending and (end_line is None or index < end_line):
            addition = ("\n" if parts else "") + pending.rstrip("\r")
            if max_length is not None and length + len(addition) > max_length:
                remaining = max_length - length
                if remaining > 0:
                    parts.append(addition[:remaining])
                truncated = True
            else:
                parts.append(addition)
        return "".join(parts) + ("(truncated)" if truncated else ""), truncated, stopped_early

    async def write_file(
            self,
            filepath: str,
            content: str,
            append: bool = False,
            leading_newline: bool = False,
            trailing_newline: bool = False,
            sudo: bool = False,
    ) -> FileWriteResult:
        """根据传递的文件路径+内容向指定文件写入内容"""
        async with self._write_lock(filepath):
            return await self._write_file_locked(
                filepath, content, append, leading_newline, trailing_newline, sudo
            )

    async def _write_file_locked(
            self, filepath: str, content: str, append: bool,
            leading_newline: bool, trailing_newline: bool, sudo: bool,
    ) -> FileWriteResult:
        """在文件锁内执行完整写入，保证 append 的读取、复制和替换不可交错。"""
        temp_path = None
        try:
            # 1.组装实际写入的内容
            if leading_newline:
                content = "\n" + content
            if trailing_newline:
                content = content + "\n"
            if len(content.encode("utf-8")) > MAX_WRITE_BYTES:
                raise BadRequestException(
                    f"写入内容不能超过 {MAX_WRITE_BYTES // (1024 * 1024)} MiB"
                )

            # 2.判断是否是sudo权限，如果是则使用命令行的形式先写入一个缓存文件，然后将缓存文件覆盖原始文件
            if sudo:
                # sudo tee 从 stdin 写入，路径作为独立参数，避免命令注入和转义问题。
                tee_args = ["sudo", "tee"]
                if append:
                    tee_args.append("-a")
                tee_args.append(filepath)
                process = await asyncio.create_subprocess_exec(
                    *tee_args,
                    stdout=asyncio.subprocess.PIPE,
                    stderr=asyncio.subprocess.PIPE,
                )

                try:
                    # 等待子进程执行完毕
                    stdout, stderr = await process.communicate(content.encode("utf-8"))
                    bytes_written = len(content.encode("utf-8"))

                    # 检测子进程是否正常执行
                    if process.returncode != 0:
                        raise BadRequestException(f"文件内容写入失败: {stderr.decode()}")
                finally:
                    if process.returncode is None:
                        await asyncio.shield(self._stop_process(process))

            else:
                # 非 sudo 写入先落同目录临时文件，完成后原子替换目标文件。
                parent_dir = os.path.dirname(filepath)
                if parent_dir:
                    os.makedirs(parent_dir, exist_ok=True)
                existing_mode = None
                if os.path.exists(filepath):
                    existing_mode = stat.S_IMODE(os.stat(filepath).st_mode)
                fd, temp_path = tempfile.mkstemp(prefix=".write-", dir=parent_dir or ".")
                os.close(fd)

                def async_write_file() -> int:
                    # 追加模式也通过分块复制旧文件，避免大文件整体载入内存。
                    with open(temp_path, "wb") as target:
                        if append and os.path.exists(filepath):
                            with open(filepath, "rb") as source:
                                shutil.copyfileobj(source, target, length=READ_CHUNK_BYTES)
                        encoded = content.encode("utf-8")
                        target.write(encoded)
                    return len(encoded)

                bytes_written = await asyncio.to_thread(async_write_file)
                if existing_mode is not None:
                    os.chmod(temp_path, existing_mode)
                os.replace(temp_path, filepath)
                temp_path = None

            return FileWriteResult(
                filepath=filepath,
                bytes_written=bytes_written,
            )
        except Exception as e:
            # 14.根据不同的错误执行不同的操作
            if temp_path:
                try:
                    os.unlink(temp_path)
                except FileNotFoundError:
                    pass
            logger.error(f"文件内容写入失败: {str(e)}")
            if isinstance(e, BadRequestException):
                raise
            raise AppException(f"文件内容写入失败: {str(e)}")
        finally:
            # 取消请求不会进入 except Exception，仍需清理未替换的临时文件。
            if temp_path:
                try:
                    os.unlink(temp_path)
                except FileNotFoundError:
                    pass

    async def replace_in_file(
            self,
            filepath: str,
            old_str: str,
            new_str: str,
            sudo: bool = False,
    ) -> FileReplaceResult:
        """根据传递的数据替换文件内指定的内容"""
        # 读取和写回必须处于同一把锁内，否则并发 append 可能被旧快照覆盖。
        async with self._write_lock(filepath):
            file_read_result = await self.read_file(filepath=filepath, sudo=sudo, max_length=None)
            if file_read_result.truncated:
                raise BadRequestException(
                    f"文件超过可替换上限 {MAX_READ_BYTES // (1024 * 1024)} MiB"
                )
            content = file_read_result.content
            replaced_count = content.count(old_str)
            if replaced_count == 0:
                return FileReplaceResult(filepath=filepath, replaced_count=0)

            new_content = content.replace(old_str, new_str)
            if len(new_content.encode("utf-8")) > MAX_WRITE_BYTES:
                raise BadRequestException(
                    f"替换后的文件不能超过 {MAX_WRITE_BYTES // (1024 * 1024)} MiB"
                )
            await self._write_file_locked(filepath, new_content, False, False, False, sudo)
            return FileReplaceResult(filepath=filepath, replaced_count=replaced_count)

    async def search_in_file(
            self,
            filepath: str,
            regex: str,
            sudo: bool = False,
    ) -> FileSearchResult:
        """根据传递的文件路径+匹配规则查询文件内符合的内容"""
        matches, line_numbers = [], []
        # ReDoS 防护：限制模式长度（无复杂度分析能力，至少挡住超长病态模式）
        if len(regex) > 256:
            raise BadRequestException("正则表达式过长（上限 256 字符）")
        try:
            pattern = re.compile(regex)
        except Exception as e:
            raise BadRequestException(f"传递正则表达式[{regex}]出错: {str(e)}")
        if not sudo:
            self._ensure_regular_file(filepath)
            self._validate_read_limit()
            def scan() -> bool:
                truncated = False
                bytes_seen = 0
                for index, chunk in enumerate(self._iter_file_lines(filepath)):
                    line = chunk.text
                    bytes_seen += len(line.encode("utf-8"))
                    if bytes_seen > MAX_READ_BYTES:
                        truncated = True
                        break
                    if pattern.search(line.rstrip("\r\n")):
                        if len(matches) >= MAX_SEARCH_MATCHES:
                            return True
                        matches.append(line.rstrip("\r\n"))
                        line_numbers.append(index)
                    if chunk.truncated:
                        truncated = True
                        break
                return truncated
            truncated = await asyncio.to_thread(scan)
        else:
            process = await asyncio.create_subprocess_exec(
                "sudo", "cat", filepath,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
            )
            truncated = False
            bytes_seen = 0
            index = 0
            pending = ""
            try:
                while process.stdout:
                    raw = await process.stdout.read(READ_CHUNK_BYTES)
                    if not raw:
                        break
                    bytes_seen += len(raw)
                    if bytes_seen > MAX_READ_BYTES:
                        truncated = True
                        await asyncio.shield(self._stop_process(process))
                        break
                    pending += raw.decode("utf-8", errors="replace")
                    while "\n" in pending and not truncated:
                        line, pending = pending.split("\n", 1)
                        if len(line.encode("utf-8")) > MAX_PENDING_BYTES:
                            truncated = True
                            await asyncio.shield(self._stop_process(process))
                            break
                        line = line.rstrip("\r")
                        if pattern.search(line):
                            if len(matches) >= MAX_SEARCH_MATCHES:
                                truncated = True
                                await asyncio.shield(self._stop_process(process))
                                break
                            matches.append(line)
                            line_numbers.append(index)
                        index += 1
                    if len(pending.encode("utf-8")) > MAX_PENDING_BYTES:
                        truncated = True
                        await asyncio.shield(self._stop_process(process))
                        break
                if not truncated and pending:
                    if pattern.search(pending.rstrip("\r")):
                        if len(matches) >= MAX_SEARCH_MATCHES:
                            truncated = True
                        else:
                            matches.append(pending.rstrip("\r"))
                            line_numbers.append(index)
                if process.returncode is None:
                    await process.wait()
                if process.returncode != 0 and not truncated:
                    stderr = await process.stderr.read() if process.stderr else b""
                    raise BadRequestException(f"搜索文件失败: {stderr.decode(errors='replace')}")
            finally:
                # 取消或异常退出时，确保 sudo cat 不会遗留。
                if process.returncode is None:
                    await asyncio.shield(self._stop_process(process))

        return FileSearchResult(
            filepath=filepath,
            matches=matches,
            line_numbers=line_numbers,
            truncated=truncated,
        )

    @classmethod
    async def find_files(cls, dir_path: str, glob_pattern: str) -> FileFindResult:
        """根据传递的文件夹路径+glob规则查询文件列表"""
        # 0.glob 必须是相对模式：以 / 开头会 join 出目录逃逸（如 /etc/**）
        if os.path.isabs(glob_pattern):
            raise BadRequestException("glob_pattern 必须是相对路径")
        normalized_pattern = glob_pattern.replace("\\", "/")
        if any(part == ".." for part in normalized_pattern.split("/")):
            raise BadRequestException("glob_pattern 不允许包含父目录")

        # 1.检测下传递进来的目录是否存在
        cls._ensure_directory(dir_path)

        # 2.定义一个异步函数使用asyncio子线程运行避免IO阻塞
        def async_glob():
            search_pattern = os.path.join(dir_path, glob_pattern)
            files = list(itertools.islice(
                glob.iglob(search_pattern, recursive=True), MAX_FIND_FILES + 1
            ))
            return files[:MAX_FIND_FILES], len(files) > MAX_FIND_FILES

        # 3.创建子线程完成任务
        files, truncated = await asyncio.to_thread(async_glob)

        return FileFindResult(dir_path=dir_path, files=files, truncated=truncated)

    @classmethod
    async def upload_file(cls, file: UploadFile, filepath: str) -> FileUploadResult:
        """根据传递的文件源+路径将文件上传至沙箱"""
        temp_path = None
        try:
            # 1.定义分块上传，每次只上传8k
            chunk_size = 1024 * 8
            file_size = 0

            # 2.确保上传文件所在的目录存在
            parent_dir = os.path.dirname(filepath) or "."
            os.makedirs(parent_dir, exist_ok=True)
            existing_mode = None
            if os.path.exists(filepath):
                existing_mode = stat.S_IMODE(os.stat(filepath).st_mode)
            fd, temp_path = tempfile.mkstemp(prefix=".upload-", dir=parent_dir)
            os.close(fd)

            # 3.定义一个异步函数用于上传文件避免阻塞进程
            def async_write_file():
                nonlocal file_size
                with open(temp_path, "wb") as f:
                    while True:
                        chunk = file.file.read(chunk_size)
                        if not chunk:
                            break
                        if file_size + len(chunk) > MAX_UPLOAD_BYTES:
                            raise BadRequestException(
                                f"上传文件不能超过 {MAX_UPLOAD_BYTES // (1024 * 1024)} MiB"
                            )
                        f.write(chunk)
                        file_size += len(chunk)

            # 4.使用asyncio子线程完成函数调用
            await asyncio.to_thread(async_write_file)
            if existing_mode is not None:
                os.chmod(temp_path, existing_mode)
            os.replace(temp_path, filepath)
            temp_path = None

            return FileUploadResult(
                filepath=filepath,
                file_size=file_size,
                success=True,
            )
        except Exception as e:
            if temp_path:
                try:
                    os.unlink(temp_path)
                except FileNotFoundError:
                    pass
            logger.error(f"上传文件到沙箱出错: {str(e)}")
            if isinstance(e, BadRequestException):
                raise
            raise AppException(f"上传文件到沙箱出错: {str(e)}")
        finally:
            if temp_path:
                try:
                    os.unlink(temp_path)
                except FileNotFoundError:
                    pass

    @classmethod
    async def ensure_file(cls, filepath: str) -> None:
        """传递filepath用于确保当前文件存在"""
        cls._ensure_regular_file(filepath)

    @classmethod
    async def check_file_exists(cls, filepath: str) -> FileCheckResult:
        """根据传递的路径判断文件是否存在"""
        return FileCheckResult(
            filepath=filepath,
            exists=os.path.isfile(filepath),
        )

    async def delete_file(self, filepath: str, sudo: bool = False) -> FileDeleteResult:
        """根据传递的路径+sudo删除指定文件"""
        # sudo 模式下普通用户可能无法看到文件，交给 rm 返回权限和不存在错误。
        try:
            async with self._write_lock(filepath):
                if not sudo:
                    await self.ensure_file(filepath)
                if sudo:
                    process = await asyncio.create_subprocess_exec(
                        "sudo", "rm", "--", filepath,
                        stdout=asyncio.subprocess.PIPE,
                        stderr=asyncio.subprocess.PIPE,
                    )
                    try:
                        _, stderr = await process.communicate()
                        if process.returncode != 0:
                            raise NotFoundException(f"删除文件失败: {stderr.decode(errors='replace').strip()}")
                    finally:
                        if process.returncode is None:
                            await asyncio.shield(self._stop_process(process))
                else:
                    await asyncio.to_thread(os.remove, filepath)
                return FileDeleteResult(filepath=filepath, deleted=True)
        except Exception as e:
            logger.error(f"删除文件{filepath}失败: {str(e)}")
            if isinstance(e, (BadRequestException, NotFoundException)):
                raise
            raise AppException(f"删除文件{filepath}失败: {str(e)}")
