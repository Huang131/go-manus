"""子进程测试替身，供 FileService / ShellService 用例共用。

只实现被测代码真正触碰的接口（stdout/stderr/returncode/terminate/kill/wait/communicate），
不模拟真实进程语义。把原先散落在各用例里的重复 Process/Stream 定义收敛到一处，
差异（是否置 returncode、communicate 是否抛取消）通过构造参数表达。
"""


class FakeStream:
    """stdout/stderr 替身：按预设块依次返回，耗尽后返回 b""。

    on_read 用于构造"读取时阻塞"的场景（如请求取消用例），回调需自行 await。
    """

    def __init__(self, chunks=(), *, on_read=None):
        self._chunks = iter(chunks)
        self._on_read = on_read

    async def read(self, size=-1):
        if self._on_read is not None:
            await self._on_read()
        return next(self._chunks, b"")


class FakeProcess:
    """asyncio 子进程替身。

    - terminate_returncode：terminate/kill 后进程的退出码；
    - wait_returncode：wait() 完成时写入的退出码（None 表示不改动 returncode）；
    - communicate_error：communicate() 抛出的异常实例（如 asyncio.CancelledError）。
    """

    def __init__(
        self,
        *,
        returncode=None,
        stdout=None,
        stderr=None,
        terminate_returncode=None,
        wait_returncode=None,
        communicate_error=None,
        communicate_result=(b"", b""),
    ) -> None:
        self.returncode = returncode
        self.stdout = stdout if stdout is not None else FakeStream([])
        self.stderr = stderr if stderr is not None else FakeStream([])
        self.terminated = False
        self._terminate_returncode = terminate_returncode
        self._wait_returncode = wait_returncode
        self._communicate_error = communicate_error
        self._communicate_result = communicate_result

    def terminate(self):
        self.terminated = True
        if self._terminate_returncode is not None:
            self.returncode = self._terminate_returncode

    def kill(self):
        self.terminate()

    async def wait(self):
        if self._wait_returncode is not None:
            self.returncode = self._wait_returncode
        return self.returncode

    async def communicate(self, data=None):
        if self._communicate_error is not None:
            raise self._communicate_error
        return self._communicate_result
